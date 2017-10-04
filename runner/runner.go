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
}

func (r *Runner) Messagef(format string, a ...interface{}) {
	r.messageLog.Printf(format, a...)
	r.Logger.Printf(format, a...)
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
	lockedService, done, err := r.Service.ObtainLock("autorestart")
	if err != nil {
		return errors.WithStack(err)
	}
	oldService := r.Service
	r.Service = lockedService
	defer func() {
		done()
		r.Service = oldService
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

	standardLog := &Log{
		file:   r.logFile,
		name:   r.Service.Name,
		stream: "stdout",
	}
	errorLog := &Log{
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
	cmd.Stdout = standardLog
	cmd.Stderr = errorLog

	r.command = NewRunningCommand(r.Service, cmd, &r.commandWait)
	r.command.Start(errorLog)

	return nil
}
