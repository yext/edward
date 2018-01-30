package runner

import (
	"fmt"
	"strings"
	"time"

	"github.com/yext/edward/home"

	"github.com/hpcloud/tail"
	"github.com/pkg/errors"
	"github.com/theothertomelliott/gopsutil-nocgo/net"
	"github.com/theothertomelliott/gopsutil-nocgo/process"
	"github.com/yext/edward/instance"
	"github.com/yext/edward/services"
)

// WaitUntilLive blocks until a command running the specified service is in the RUNNING state.
// An error will be returned if the command exits before reaching RUNNING.
func WaitUntilLive(dirConfig *home.EdwardConfiguration, pid int32, service *services.ServiceConfig) error {
	logger := service.Logger
	fmt.Println("Wait until live")
	logger.Printf("Waiting for %v to start.\n", service.Name)

	var startCheck func(cancel <-chan struct{}) error
	if service.LaunchChecks != nil && len(service.LaunchChecks.LogText) > 0 {
		fmt.Printf("Waiting for log text: %v\n", service.LaunchChecks.LogText)
		logger.Printf("Waiting for log text: %v", service.LaunchChecks.LogText)
		startCheck = func(cancel <-chan struct{}) error {
			return errors.WithStack(
				waitForLogText(dirConfig, service.LaunchChecks.LogText, cancel, service),
			)
		}
	} else if service.LaunchChecks != nil && len(service.LaunchChecks.Ports) > 0 {
		fmt.Printf("Waiting for ports: %v\n", service.LaunchChecks.Ports)
		logger.Printf("Waiting for ports: %v", service.LaunchChecks.Ports)
		startCheck = func(cancel <-chan struct{}) error {
			return errors.WithStack(
				waitForListeningPorts(service.LaunchChecks.Ports, cancel, pid),
			)
		}
	} else if service.LaunchChecks != nil && service.LaunchChecks.Wait != 0 {
		fmt.Printf("Waiting for: %dms\n", service.LaunchChecks.Wait)
		logger.Printf("Waiting for: %dms", service.LaunchChecks.Wait)
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
		fmt.Printf("Waiting for any port\n")
		logger.Printf("Waiting for any port")
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
		fmt.Printf("Process started\n")
		logger.Printf("Process started")
		return errors.WithStack(result.error)
	}

}

func waitForLogText(dirConfig *home.EdwardConfiguration, line string, cancel <-chan struct{}, service *services.ServiceConfig) error {
	// Read output until we get the success
	var t *tail.Tail
	var err error
	t, err = tail.TailFile(service.GetRunLog(dirConfig.LogDir), tail.Config{
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
