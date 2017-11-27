package runner

import (
	"os"
	"os/exec"
	"os/signal"
	"strconv"
	"sync"
	"syscall"
	"time"

	"github.com/pkg/errors"
	"github.com/theothertomelliott/gopsutil-nocgo/net"
	"github.com/theothertomelliott/gopsutil-nocgo/process"
	"github.com/yext/edward/commandline"
	"github.com/yext/edward/home"
	"github.com/yext/edward/instance"
	"github.com/yext/edward/services"
)

// Runner provides state and functions for running a given service
type Runner struct {
	Service    *services.ServiceConfig
	command    *RunningCommand
	logFile    *os.File
	messageLog *Log

	commandWait sync.WaitGroup
	NoWatch     bool
	WorkingDir  string

	Logger Logger

	status instance.Status

	instanceId string

	standardLog *Log
	errorLog    *Log
}

func (r *Runner) Messagef(format string, a ...interface{}) {
	r.messageLog.Printf(format, a...)
	r.Logger.Printf(format, a...)
}

func (r *Runner) Run(args []string) error {

	r.status = instance.Status{
		StartTime: time.Now(),
	}

	if r.WorkingDir != "" {
		err := os.Chdir(r.WorkingDir)
		if err != nil {
			return errors.WithStack(err)
		}
	}

	// Set the instance id
	command, err := r.Service.GetCommand(services.ContextOverride{})
	if err != nil {
		r.Messagef("Could not get service command: %v\n", err)
	}
	r.instanceId = command.InstanceId

	r.updateServiceState(instance.StateStarting)

	err = r.configureLogs()
	if err != nil {
		return errors.WithStack(err)
	}
	defer func() {
		// TODO: Handle the "Died state"
		r.updateServiceState(instance.StateStopped)
		r.removeServiceState()
		r.Messagef("Service stopped\n")
	}()

	statusTick := time.NewTicker(time.Second)
	defer statusTick.Stop()
	go func() {
		for _ = range statusTick.C {
			r.updateStatusDetail()
		}
	}()

	r.commandWait.Add(1)
	err = r.startService()
	if err != nil {
		return errors.WithStack(err)
	}

	r.updateServiceState(instance.StateRunning)

	closeWatchers := r.configureWatch()
	if closeWatchers != nil {
		defer closeWatchers()
	}
	r.configureSignals()

	r.commandWait.Wait()
	return nil
}

func (r *Runner) updateServiceState(newState instance.State) {
	r.status.State = newState
	dir := home.EdwardConfig.StateDir
	err := instance.SaveStatusForService(r.Service, r.instanceId, r.status, dir)
	if err != nil {
		r.Messagef("could not save state:", err)
	}
}

func (r *Runner) updateStatusDetail() {
	r.status.StdoutLines = r.standardLog.Len()
	r.status.StderrLines = r.errorLog.Len()

	pid := r.command.Pid()
	if pid != 0 {
		proc, err := process.NewProcess(int32(pid))
		if err != nil {
			r.Messagef("could not get process:", err)
			return
		}
		r.status.MemoryInfo, err = proc.MemoryInfo()
		if err != nil {
			r.Messagef("could not get memory information:", err)
			return
		}
		ports, err := doGetPorts(proc)
		if err != nil {
			r.Messagef("could not get ports:", err)
			return
		}
		r.status.Ports = ports
	}

	dir := home.EdwardConfig.StateDir
	err := instance.SaveStatusForService(r.Service, r.instanceId, r.status, dir)
	if err != nil {
		r.Messagef("could not save state:", err)
	}
}

func doGetPorts(proc *process.Process) ([]string, error) {
	connections, err := net.Connections("all")
	if err != nil {
		return nil, errors.WithStack(err)
	}

	var ports []string
	for _, connection := range connections {
		if connection.Status == "LISTEN" && connection.Pid == proc.Pid {
			ports = append(ports, strconv.Itoa(int(connection.Laddr.Port)))
		}
	}

	children, err := proc.Children()
	// This will error out if the process has finished or has no children
	if err != nil {
		return ports, nil
	}
	for _, child := range children {
		childPorts, err := doGetPorts(child)
		if err == nil {
			ports = append(ports, childPorts...)
		}
	}
	return ports, nil
}

func (r *Runner) removeServiceState() {
	dir := home.EdwardConfig.StateDir
	err := instance.DeleteStatusForService(r.Service, r.instanceId, dir)
	if err != nil {
		r.Messagef("could not delete state:", err)
	}
}

func (r *Runner) configureLogs() error {
	logLocation := r.Service.GetRunLog()
	os.Remove(logLocation)

	var err error
	r.logFile, err = os.Create(logLocation)
	if err != nil {
		return errors.WithStack(err)
	}

	r.messageLog = &Log{
		file:   r.logFile,
		name:   r.Service.Name,
		stream: "messages",
	}
	return nil
}

func (r *Runner) configureSignals() {
	signalChan := make(chan os.Signal, 1)
	signal.Notify(signalChan, os.Interrupt)
	go func() {
		for range signalChan {
			r.Messagef("Received interrupt\n")
			err := r.stopService()
			if err != nil {
				r.Messagef("Could not stop service: %v", err)
			}
		}
	}()
}

func (r *Runner) configureWatch() func() {
	if !r.NoWatch {
		closeWatchers, err := BeginWatch(r.Service, r.restartService, r.messageLog)
		if err != nil {
			r.Messagef("Could not enable auto-restart: %v\n", err)
			return nil
		}
		if closeWatchers != nil {
			r.Messagef("Auto-restart enabled. This service will restart when files in its watch directories are edited.\nThis can be disabled using the --no-watch flag.\n")
		}
		return closeWatchers
	}
	return nil
}

func (r *Runner) restartService() error {
	r.Messagef("Restarting service\n")

	// Increment the counter to prevent exiting unexpectedly
	r.commandWait.Add(1)

	err := r.stopService()
	if err != nil {
		return errors.WithStack(err)
	}
	err = r.startService()
	if err != nil {
		return errors.WithStack(err)
	}
	return nil
}

func (r *Runner) stopService() error {
	wd, err := os.Getwd()
	if err != nil {
		return errors.WithStack(err)
	}

	command, err := r.Service.GetCommand(services.ContextOverride{})
	if err != nil {
		r.Messagef("Could not get service command: %v\n", err)
	}

	var scriptErr error
	var scriptOutput []byte
	if r.Service.Commands.Stop != "" {
		r.Messagef("Running stop script for %v.\n", r.Service.Name)
		scriptOutput, scriptErr = command.RunStopScript(wd)
		if scriptErr != nil {
			r.Messagef("%v\n", string(scriptOutput))
			r.Messagef("Stop script failed: %v\n", scriptErr)
		}
		if r.waitForCompletionWithTimeout(1 * time.Second) {
			return nil
		}
		r.Messagef("Stop script did not effectively stop service, sending interrupt\n")
	}

	err = r.command.Interrupt()
	if err != nil {
		return errors.WithStack(err)
	}

	if r.waitForCompletionWithTimeout(2 * time.Second) {
		return nil
	}
	r.Messagef("Interrupt did not effectively stop service, sending kill\n")

	err = r.command.Kill()
	if err != nil {
		return errors.WithMessage(err, "Kill failed")
	}

	if r.waitForCompletionWithTimeout(2 * time.Second) {
		return nil
	}
	return errors.New("kill did not stop service")
}

func (r *Runner) waitForCompletionWithTimeout(timeout time.Duration) bool {
	var completed = make(chan struct{})
	defer close(completed)
	timer := time.NewTimer(timeout)
	defer timer.Stop()

	go func() {
		r.command.Wait()
		completed <- struct{}{}
	}()

	select {
	case <-completed:
		return true
	case <-timer.C:
		return false
	}
}

func (r *Runner) startService() error {
	r.Messagef("Service starting\n")

	r.standardLog = &Log{
		file:   r.logFile,
		name:   r.Service.Name,
		stream: "stdout",
	}
	r.errorLog = &Log{
		file:   r.logFile,
		name:   r.Service.Name,
		stream: "stderr",
	}

	command, cmdArgs, err := commandline.ParseCommand(os.ExpandEnv(r.Service.Commands.Launch))
	if err != nil {
		return errors.WithStack(err)
	}
	cmd := exec.Command(command, cmdArgs...)
	if r.Service.Path != nil {
		cmd.Dir = os.ExpandEnv(*r.Service.Path)
	}
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	cmd.Stdout = r.standardLog
	cmd.Stderr = r.errorLog

	r.command = NewRunningCommand(r.Service, cmd, &r.commandWait)
	r.command.Start(r.errorLog)

	return nil
}
