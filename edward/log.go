package edward

import (
	"sort"

	"github.com/pkg/errors"
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

	// Sort initial lines
	sort.Sort(byTime(lines))
	for _, line := range lines {
		printMessage(line, services.CountServices(sgs) > 1)
	}

	for logMessage := range logChannel {
		printMessage(logMessage, services.CountServices(sgs) > 1)
	}

	return nil
}
