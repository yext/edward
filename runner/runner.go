package runner

import (
	"os"
	"os/exec"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/pkg/errors"
	"github.com/yext/edward/commandline"
	"github.com/yext/edward/config"
	"github.com/yext/edward/services"
)

// Runner provides state and functions for running a given service
type Runner struct {
	service    *services.ServiceConfig
	command    *RunningCommand
	logFile    *os.File
	messageLog *Log

	commandWait sync.WaitGroup
	NoWatch     bool
	WorkingDir  string

	Logger Logger
}

func (r *Runner) Messagef(format string, a ...interface{}) {
	r.messageLog.Printf(format, a...)
	r.Logger.Printf(format, a...)
}

// NewRunningCommand creates a RunningCommand for a given service and exec.Cmd
func NewRunningCommand(service *services.ServiceConfig, cmd *exec.Cmd, commandWait *sync.WaitGroup) *RunningCommand {
	return &RunningCommand{
		service:     service,
		command:     cmd,
		done:        make(chan struct{}),
		commandWait: commandWait,
	}
}

// RunningCommand provides state and functions for running a service
type RunningCommand struct {
	service     *services.ServiceConfig
	command     *exec.Cmd
	done        chan struct{}
	commandWait *sync.WaitGroup
}

// Start starts a command running in a goroutine
func (c *RunningCommand) Start(errorLog Logger) {
	go func() {
		err := c.command.Run()
		if err != nil {
			errorLog.Printf("%v", err)
		}
		c.commandWait.Done()
		close(c.done)
	}()
}

// Interrupt sends an interrupt to a running command
func (c *RunningCommand) Interrupt() error {
	return errors.WithStack(
		services.InterruptGroup(services.OperationConfig{}, c.command.Process.Pid, c.service),
	)
}

// Kill sends a kill signal to a running command
func (c *RunningCommand) Kill() error {
	return errors.WithStack(
		services.KillGroup(services.OperationConfig{}, c.command.Process.Pid, c.service),
	)
}

// Wait blocks until this command has exited
func (c *RunningCommand) Wait() {
	<-c.done
}

// Logger provides a simple interface for logging
type Logger interface {
	Printf(format string, a ...interface{})
}

func (r *Runner) Run(args []string) error {
	if r.WorkingDir != "" {
		err := os.Chdir(r.WorkingDir)
		if err != nil {
			return errors.WithStack(err)
		}
	}

	if len(args) < 1 {
		return errors.New("a service name is required")
	}

	var ok bool
	r.service, ok = config.GetServiceMap()[args[0]]
	if !ok {
		return errors.New("service not found")
	}

	err := r.configureLogs()
	if err != nil {
		return errors.WithStack(err)
	}
	defer r.Messagef("Service stopped\n")

	r.commandWait.Add(1)
	err = r.startService()
	if err != nil {
		return errors.WithStack(err)
	}

	closeWatchers := r.configureWatch()
	if closeWatchers != nil {
		defer closeWatchers()
	}
	r.configureSignals()

	r.commandWait.Wait()
	return nil
}

func (r *Runner) configureLogs() error {
	logLocation := r.service.GetRunLog()
	os.Remove(logLocation)

	var err error
	r.logFile, err = os.Create(logLocation)
	if err != nil {
		return errors.WithStack(err)
	}

	r.messageLog = &Log{
		file:   r.logFile,
		name:   r.service.Name,
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
		closeWatchers, err := BeginWatch(r.service, r.restartService, r.messageLog)
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
	lockedService, done, err := r.service.ObtainLock("autorestart")
	if err != nil {
		return errors.WithStack(err)
	}
	oldService := r.service
	r.service = lockedService
	defer func() {
		done()
		r.service = oldService
	}()

	err = r.stopService()
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
	command, err := r.service.GetCommand(services.ContextOverride{})
	if err != nil {
		r.Messagef("Could not get service command: %v\n", err)
	}

	var scriptErr error
	var scriptOutput []byte
	if r.service.Commands.Stop != "" {
		r.Messagef("Running stop script for %v.\n", r.service.Name)
		scriptOutput, scriptErr = command.RunStopScript()
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

	standardLog := &Log{
		file:   r.logFile,
		name:   r.service.Name,
		stream: "stdout",
	}
	errorLog := &Log{
		file:   r.logFile,
		name:   r.service.Name,
		stream: "stderr",
	}

	command, cmdArgs, err := commandline.ParseCommand(os.ExpandEnv(r.service.Commands.Launch))
	if err != nil {
		return errors.WithStack(err)
	}
	cmd := exec.Command(command, cmdArgs...)
	if r.service.Path != nil {
		cmd.Dir = os.ExpandEnv(*r.service.Path)
	}
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	cmd.Stdout = standardLog
	cmd.Stderr = errorLog

	r.command = NewRunningCommand(r.service, cmd, &r.commandWait)
	r.command.Start(errorLog)

	return nil
}
