package runner

import (
	"os/exec"
	"sync"

	"github.com/pkg/errors"
	"github.com/yext/edward/instance"
	"github.com/yext/edward/services"
)

// NewRunningCommand creates a RunningCommand for a given service and exec.Cmd
func NewRunningCommand(service *services.ServiceConfig, cmd *exec.Cmd, commandWait *sync.WaitGroup) *RunningCommand {
	return &RunningCommand{
		service:     service,
		command:     cmd,
		started:     make(chan struct{}),
		done:        make(chan struct{}),
		commandWait: commandWait,
	}
}

// RunningCommand provides state and functions for running a service
type RunningCommand struct {
	service     *services.ServiceConfig
	command     *exec.Cmd
	started     chan struct{}
	done        chan struct{}
	commandWait *sync.WaitGroup
}

// Start starts a command running in a goroutine.
// Will block until the service has started running and has a PID.
func (c *RunningCommand) Start(errorLog Logger) {
	go func() {
		err := c.command.Start()
		close(c.started)
		if err != nil {
			errorLog.Printf("start error: %v", err)
		}
		err = c.command.Wait()
		if err != nil {
			errorLog.Printf("start error: %v", err)
		}
		c.commandWait.Done()
		close(c.done)
	}()
	<-c.started
}

// Interrupt sends an interrupt to a running command
func (c *RunningCommand) Interrupt() error {
	return errors.WithStack(
		instance.InterruptGroup(services.OperationConfig{}, c.command.Process.Pid, c.service),
	)
}

// Kill sends a kill signal to a running command
func (c *RunningCommand) Kill() error {
	return errors.WithStack(
		instance.KillGroup(services.OperationConfig{}, c.command.Process.Pid, c.service),
	)
}

// Wait blocks until this command has exited
func (c *RunningCommand) Wait() {
	<-c.done
}

// Pid returns the process id for the running command
func (c *RunningCommand) Pid() int {
	if c.command == nil || c.command.Process == nil {
		return 0
	}
	return c.command.Process.Pid
}
