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
	if b.Backend.LaunchChecks != nil && len(b.Backend.LaunchChecks.LogText) > 0 {
		<-b.launchConditionMet
		return nil
	} else if b.Backend.LaunchChecks != nil && len(b.Backend.LaunchChecks.Ports) > 0 {
		return errors.WithStack(
			waitForListeningPorts(b.Backend.LaunchChecks.Ports, pid),
		)
	} else if b.Backend.LaunchChecks != nil && b.Backend.LaunchChecks.Wait != 0 {
		delay := time.NewTimer(time.Duration(b.Backend.LaunchChecks.Wait) * time.Millisecond)
		defer delay.Stop()
		select {
		case <-delay.C:
			return nil
		}
	}

	return errors.WithStack(
		waitForAnyPort(pid),
	)
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

func waitForListeningPorts(ports []int, pid int32) error {
	for true {
		time.Sleep(100 * time.Millisecond)

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

func waitForAnyPort(pid int32) error {
	for true {
		time.Sleep(100 * time.Millisecond)

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
