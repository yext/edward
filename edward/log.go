package edward

import (
	"sort"
	"time"

	"github.com/pkg/errors"
	"github.com/yext/edward/instance"
	"github.com/yext/edward/runner"
	"github.com/yext/edward/services"
)

func (c *Client) Log(names []string) error {
	if len(names) == 0 {
		return errors.New("At least one service or group must be specified")
	}
	sgs, err := c.getServicesOrGroups(names)
	if err != nil {
		return errors.WithStack(err)
	}

	var logChannel = make(chan runner.LogLine)
	var lines []runner.LogLine
	for _, sg := range sgs {
		switch v := sg.(type) {
		case *services.ServiceConfig:
			newLines, err := followServiceLog(v, logChannel)
			if err != nil {
				return err
			}
			lines = append(lines, newLines...)
		case *services.ServiceGroupConfig:
			newLines, err := followGroupLog(v, logChannel)
			if err != nil {
				return err
			}
			lines = append(lines, newLines...)
		}
	}

	var stopChannel = make(chan runner.LogLine)
	statusTicker := time.NewTicker(time.Second * 5)
	go func() {
		for _ = range statusTicker.C {
			running, err := checkAllRunning(sgs)
			if err != nil {
				c.Logger.Printf("Error checking service state for tailing: %v", err)
				continue
			}
			// All services stopped, notify the log process
			if !running {
				statusTicker.Stop()
				close(stopChannel)
			}
		}
	}()

	// Sort initial lines
	sort.Sort(byTime(lines))
	for _, line := range lines {
		printMessage(line, services.CountServices(sgs) > 1)
	}

	var running = true
	for running {
		select {
		case logMessage := <-logChannel:
			printMessage(logMessage, services.CountServices(sgs) > 1)
		case <-stopChannel:
			running = false
		}
	}

	return nil
}

func checkAllRunning(sgs []services.ServiceOrGroup) (bool, error) {
	allServices := services.Services(sgs)
	for _, s := range allServices {
		running, err := instance.HasRunning(s)
		if err != nil {
			return false, errors.WithStack(err)
		}
		if running {
			return true, nil
		}
	}
	return false, nil
}
