package services

import (
	"os/exec"
	"strings"
	"time"

	"github.com/hpcloud/tail"
	"github.com/pkg/errors"
	"github.com/theothertomelliott/gopsutil-nocgo/net"
	"github.com/theothertomelliott/gopsutil-nocgo/process"
)

// WaitUntilLive blocks until a command running the specified service is in the RUNNING state.
// An error will be returned if the command exits before reaching RUNNING.
func WaitUntilLive(command *exec.Cmd, service *ServiceConfig) error {

	service.printf("Waiting for %v to start.\n", service.Name)

	var startCheck func(cancel <-chan struct{}) error
	if service.LaunchChecks != nil && len(service.LaunchChecks.LogText) > 0 {
		startCheck = func(cancel <-chan struct{}) error {
			return errors.WithStack(
				waitForLogText(service.LaunchChecks.LogText, cancel, service),
			)
		}
	} else if service.LaunchChecks != nil && len(service.LaunchChecks.Ports) > 0 {
		startCheck = func(cancel <-chan struct{}) error {
			return errors.WithStack(
				waitForListeningPorts(service.LaunchChecks.Ports, cancel, command),
			)
		}
	} else if service.LaunchChecks != nil && service.LaunchChecks.Wait != 0 {
		startCheck = func(cancel <-chan struct{}) error {
			delay := time.NewTimer(time.Duration(service.LaunchChecks.Wait) * time.Millisecond)
			defer delay.Stop()
			select {
			case <-cancel:
				return nil
			case <-delay.C:
				return nil
			}
		}
	} else {
		startCheck = func(cancel <-chan struct{}) error {
			return errors.WithStack(
				waitForAnyPort(cancel, command),
			)
		}
	}

	processFinished := func(cancel <-chan struct{}) error {
		// Wait until the process exists
		command.Wait()
		select {
		case <-cancel:
			return nil
		default:
		}
		return errors.New("service terminated prematurely")
	}

	timeout := time.NewTimer(time.Duration(StartupTimeoutSeconds) * time.Second)
	defer timeout.Stop()

	done := make(chan struct{})
	defer close(done)

	select {
	case result := <-cancelableWait(done, startCheck):
		return errors.WithStack(result.error)
	case result := <-cancelableWait(done, processFinished):
		return errors.WithStack(result.error)
	case <-timeout.C:
		return errors.New("Waiting for service timed out")
	}

}

func waitForLogText(line string, cancel <-chan struct{}, service *ServiceConfig) error {
	// Read output until we get the success
	var t *tail.Tail
	var err error
	t, err = tail.TailFile(service.GetRunLog(), tail.Config{
		Follow: true,
		ReOpen: true,
		Poll:   true,
		Logger: tail.DiscardingLogger,
	})
	if err != nil {
		return errors.WithStack(err)
	}
	for logLine := range t.Lines {

		select {
		case <-cancel:
			return nil
		default:
		}

		if strings.Contains(logLine.Text, line) {
			return nil
		}
	}
	return nil
}

const portStatusListen = "LISTEN"

func (c *ServiceCommand) areAnyListeningPortsOpen(ports []int) (bool, error) {

	var matchedPorts = make(map[int]struct{})
	for _, port := range ports {
		matchedPorts[port] = struct{}{}
	}

	connections, err := net.Connections("all")
	if err != nil {
		return false, errors.WithStack(err)
	}
	for _, connection := range connections {
		if connection.Status == portStatusListen {
			if _, ok := matchedPorts[int(connection.Laddr.Port)]; ok {
				return true, nil
			}
		}
	}
	return false, nil
}

func waitForListeningPorts(ports []int, cancel <-chan struct{}, command *exec.Cmd) error {
	for true {
		time.Sleep(100 * time.Millisecond)

		select {
		case <-cancel:
			return nil
		default:
		}

		var matchedPorts = make(map[int]struct{})

		connections, err := net.Connections("all")
		if err != nil {
			return errors.WithStack(err)
		}
		for _, connection := range connections {
			if connection.Status == portStatusListen {
				matchedPorts[int(connection.Laddr.Port)] = struct{}{}
			}
		}
		allMatched := true
		for _, port := range ports {
			if _, ok := matchedPorts[port]; !ok {
				allMatched = false
			}
		}
		if allMatched {
			return nil
		}
	}
	return errors.New("exited check loop unexpectedly")
}

func waitForAnyPort(cancel <-chan struct{}, command *exec.Cmd) error {
	for true {
		time.Sleep(100 * time.Millisecond)

		select {
		case <-cancel:
			return nil
		default:
		}

		connections, err := net.Connections("all")
		if err != nil {
			return errors.WithStack(err)
		}

		proc, err := process.NewProcess(int32(command.Process.Pid))
		if err != nil {
			return errors.WithStack(err)
		}
		if hasPort(proc, connections) {
			return nil
		}
	}
	return errors.New("exited check loop unexpectedly")
}

func hasPort(proc *process.Process, connections []net.ConnectionStat) bool {
	for _, connection := range connections {
		if connection.Status == portStatusListen && connection.Pid == int32(proc.Pid) {
			return true
		}
	}

	children, err := proc.Children()
	if err == nil {
		for _, child := range children {
			if hasPort(child, connections) {
				return true
			}
		}
	}
	return false
}

func cancelableWait(cancel chan struct{}, task func(cancel <-chan struct{}) error) <-chan struct{ error } {
	finished := make(chan struct{ error })
	go func() {
		defer close(finished)
		err := task(cancel)
		finished <- struct{ error }{err}
	}()
	return finished
}
