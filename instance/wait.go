package instance

import (
	"fmt"
	"os/exec"
	"time"

	"github.com/yext/edward/home"

	"github.com/pkg/errors"
	"github.com/shirou/gopsutil/v3/net"
	"github.com/yext/edward/services"
)

// WaitUntilRunning will block the specified service until it enters the running state
func WaitUntilRunning(dirCfg *home.EdwardConfiguration, cmd *exec.Cmd, service *services.ServiceConfig) error {
	runningChan, errChan := statusRunningChan(dirCfg, service)
	exitChan := commandExitChan(cmd)

	select {
	case <-runningChan:
		return nil
	case err := <-errChan:
		return errors.WithStack(err)
	case <-exitChan:
		statuses, err := LoadStatusForService(service, dirCfg.StateDir)
		if err == nil {
			for _, status := range statuses {
				if status.State == StateDied || status.State == StateStopped {
					return fmt.Errorf("exited with state: %v", status.State)
				}
			}
		}
		return errors.New("runner process exited")
	}
}

func statusRunningChan(dirCfg *home.EdwardConfiguration, service *services.ServiceConfig) (runningChan chan struct{}, errChan chan error) {
	runningChan = make(chan struct{})
	errChan = make(chan error)

	go func() {
		for true {
			time.Sleep(time.Millisecond)
			statuses, err := LoadStatusForService(service, dirCfg.StateDir)
			if err != nil {
				continue
			}
			for _, status := range statuses {
				if status.State == StateDied || status.State == StateStopped {
					errChan <- fmt.Errorf("exited with state: %v", status.State)
					return
				}
				if status.State == StateRunning {
					close(runningChan)
					return
				}
			}
		}
	}()

	return runningChan, errChan
}

func commandExitChan(cmd *exec.Cmd) chan struct{} {
	out := make(chan struct{})
	go func() {
		cmd.Wait()
		close(out)
	}()
	return out
}

const portStatusListen = "LISTEN"

func (c *Instance) areAnyListeningPortsOpen(ports []int) (bool, error) {

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
