package edward

import (
	"github.com/pkg/errors"
	"github.com/yext/edward/instance"
	"github.com/yext/edward/instance/processes"
	"github.com/yext/edward/services"
	"github.com/yext/edward/ui"
)

func (c *Client) Status(names []string, all bool) error {
	sgs, err := c.getServiceList(names, all)
	if err != nil {
		return errors.WithStack(err)
	}

	if len(sgs) == 0 {
		return errors.New("no services found")
	}

	var serviceStatus []ui.ServiceStatus

	services := services.Services(sgs)
	for _, s := range services {
		status, err := c.getState(s)
		if err != nil {
			return errors.WithStack(err)
		}
		if status == nil {
			continue
		}
		serviceStatus = append(serviceStatus, status)
	}
	if len(serviceStatus) == 0 {
		c.UI.Infof("No services running")
		return nil
	}

	c.UI.Status(serviceStatus)

	return nil
}

type statusCommandTuple struct {
	status  instance.Status
	command *instance.Instance
}

func (s statusCommandTuple) Service() *services.ServiceConfig {
	return s.command.Service
}

func (s statusCommandTuple) Status() instance.Status {
	return s.status
}

func (s statusCommandTuple) Pid() int {
	return s.command.Pid
}

func (c *Client) getState(service *services.ServiceConfig) (*statusCommandTuple, error) {
	command, err := instance.Load(c.DirConfig, &processes.Processes{}, service, services.ContextOverride{})
	if err != nil {
		return nil, errors.WithMessage(err, "could not get service command")
	}

	statuses, _ := instance.LoadStatusForService(service, c.DirConfig.StateDir)
	if status, ok := statuses[command.InstanceId]; ok {
		// If the PID has been set to zero, the runner has died
		if command.Pid == 0 {
			status.State = instance.StateDied
		}
		return &statusCommandTuple{
			status:  status,
			command: command,
		}, nil
	}
	return nil, nil
}

func (c *Client) getServiceList(names []string, all bool) ([]services.ServiceOrGroup, error) {
	var sgs []services.ServiceOrGroup
	var err error

	if all {
		runningServices, err := instance.LoadRunningServices(c.DirConfig.StateDir)
		if err != nil {
			return nil, err
		}
		if len(names) == 0 {
			return runningServices, nil
		}
		for _, service := range runningServices {
			for _, name := range names {
				if name == service.GetName() {
					sgs = append(sgs, service)
				}
			}
		}
		return sgs, nil
	}

	if len(names) == 0 {
		return c.getAllServicesSorted(), nil
	}

	sgs, err = c.getServicesOrGroups(names)
	if err != nil {
		return nil, errors.WithStack(err)
	}
	return sgs, nil
}
