package runner

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/pkg/errors"
	"github.com/urfave/cli"
	"github.com/yext/edward/commandline"
	"github.com/yext/edward/config"
	"github.com/yext/edward/services"
)

var runnerInstance = &Runner{}

// Command defines the CLI config for the 'run' command
var Command = cli.Command{
	Name:   "run",
	Hidden: true,
	Action: runnerInstance.run,
	Flags: []cli.Flag{
		cli.BoolFlag{
			Name:        "no-watch",
			Usage:       "Disable autorestart",
			Destination: &(runnerInstance.noWatch),
		},
	},
}

// Runner provides state and functions for running a given service
type Runner struct {
	service    *services.ServiceConfig
	command    *RunningCommand
	logFile    *os.File
	messageLog *Log

	commandWait sync.WaitGroup
	noWatch     bool
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
func (c *RunningCommand) Start() {
	go func() {
		c.command.Run()
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

func (r *Runner) run(c *cli.Context) error {
	args := c.Args()
	if len(args) < 1 {
		return errors.New("a service name is required")
	}

	var ok bool
	r.service, ok = config.GetServiceMap()[args[0]]
	if !ok {
		return errors.New("service not found")
	}

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
	defer r.messageLog.Printf("Service stopped\n")

	r.commandWait.Add(1)
	err = r.startService()
	if err != nil {
		return errors.WithStack(err)
	}

	if !r.noWatch {
		closeWatchers, err := BeginWatch(r.service, r.restartService, r.messageLog)
		if err != nil {
			r.messageLog.Printf("Could not enable auto-restart: %v\n", err)
		} else if closeWatchers != nil {
			r.messageLog.Printf("Auto-restart enabled. This service will restart when files in its watch directories are edited.\nThis can be disabled using the --no-watch flag.\n")
			defer closeWatchers()
		}
	}

	signalChan := make(chan os.Signal, 1)
	signal.Notify(signalChan, os.Interrupt)
	go func() {
		for range signalChan {
			r.messageLog.Printf("Received interrupt\n")
			err := r.stopService()
			if err != nil {
				r.messageLog.Printf("Could not stop service: %v", err)
			}
		}
	}()

	r.commandWait.Wait()
	return nil
}

func (r *Runner) restartService() error {
	r.messageLog.Printf("Restarting service\n")

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
	command, err := r.service.GetCommand()
	if err != nil {
		r.messageLog.Printf("Could not get service command: %v\n", err)
	}

	var scriptErr error
	var scriptOutput []byte
	if r.service.Commands.Stop != "" {
		r.messageLog.Printf("Running stop script for %v.\n", r.service.Name)
		scriptOutput, scriptErr = command.RunStopScript()
		if scriptErr != nil {
			r.messageLog.Printf("%v\n", string(scriptOutput))
			r.messageLog.Printf("Stop script failed: %v\n", scriptErr)
		}
		if r.waitForCompletionWithTimeout(1 * time.Second) {
			return nil
		}
		r.messageLog.Printf("Stop script did not effectively stop service, sending interrupt\n")
	}

	err = r.command.Interrupt()
	if err != nil {
		return errors.WithStack(err)
	}

	if r.waitForCompletionWithTimeout(2 * time.Second) {
		return nil
	}
	r.messageLog.Printf("Interrupt did not effectively stop service, sending kill\n")

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
	r.messageLog.Printf("Service starting\n")

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
	r.command.Start()

	return nil
}

// Log provides the io.Writer interface to publish service logs to file
type Log struct {
	file   *os.File
	name   string
	stream string
}

// Printf prints a message to a RunnerLog
func (r *Log) Printf(format string, a ...interface{}) {
	msg := fmt.Sprintf(format, a...)
	r.Write([]byte(msg))
}

// Write writes a slice of bytes to a RunnerLog
func (r *Log) Write(p []byte) (int, error) {
	fmt.Println(strings.TrimRight(string(p), "\n"))
	lineData := LogLine{
		Name:    r.name,
		Time:    time.Now(),
		Stream:  r.stream,
		Message: strings.TrimSpace(string(p)),
	}

	jsonContent, err := json.Marshal(lineData)
	if err != nil {
		return 0, errors.Wrap(err, "could not prepare log line")
	}

	line := fmt.Sprintln(string(jsonContent))
	count, err := r.file.Write([]byte(line))
	if err != nil {
		fmt.Println("Error")
		return count, errors.Wrap(err, "could not write log line")
	}
	return len(p), nil
}
