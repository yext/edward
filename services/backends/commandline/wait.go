package commandline

import (
	"time"

	"github.com/pkg/errors"
	"github.com/theothertomelliott/gopsutil-nocgo/net"
	"github.com/theothertomelliott/gopsutil-nocgo/process"
	"github.com/yext/edward/instance"
)

// waitUntilLive blocks until a command running the specified service is in the RUNNING state.
// An error will be returned if the command exits before reaching RUNNING.
func (b *buildandrun) waitUntilLive() error {
	pid := int32(b.cmd.Process.Pid)

	var startCheck func(cancel <-chan struct{}) error
	if b.Backend.LaunchChecks != nil && len(b.Backend.LaunchChecks.LogText) > 0 {
		startCheck = func(cancel <-chan struct{}) error {
			select {
			case <-b.launchConditionMet:
				return nil
			case <-cancel:
				return nil
			}
		}
	} else if b.Backend.LaunchChecks != nil && len(b.Backend.LaunchChecks.Ports) > 0 {
		startCheck = func(cancel <-chan struct{}) error {
			return errors.WithStack(
				waitForListeningPorts(b.Backend.LaunchChecks.Ports, cancel, pid),
			)
		}
	} else if b.Backend.LaunchChecks != nil && b.Backend.LaunchChecks.Wait != 0 {
		startCheck = func(cancel <-chan struct{}) error {
			delay := time.NewTimer(time.Duration(b.Backend.LaunchChecks.Wait) * time.Millisecond)
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
				waitForAnyPort(cancel, pid),
			)
		}
	}

	done := make(chan struct{})
	defer close(done)

	select {
	case result := <-cancelableWait(done, startCheck):
		return errors.WithStack(result.error)
	}

}

const portStatusListen = "LISTEN"

func areAnyListeningPortsOpen(c *instance.Instance, ports []int) (bool, error) {
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

func waitForListeningPorts(ports []int, cancel <-chan struct{}, pid int32) error {
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

func waitForAnyPort(cancel <-chan struct{}, pid int32) error {
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

		proc, err := process.NewProcess(pid)
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
