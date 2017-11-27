package edward

import (
	"bytes"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/olekukonko/tablewriter"
	"github.com/pkg/errors"
	"github.com/yext/edward/home"
	"github.com/yext/edward/instance"
	"github.com/yext/edward/services"
)

func (c *Client) Status(names []string, all bool) (string, error) {
	sgs, err := c.getServiceList(names, all)
	if err != nil {
		return "", errors.WithStack(err)
	}

	if len(sgs) == 0 {
		return "No services found\n", nil
	}

	buf := new(bytes.Buffer)

	table := tablewriter.NewWriter(buf)
	headings := []string{
		"PID",
		"Name",
		"Status",
		"Ports",
		"Stdout",
		"Stderr",
		"Start Time",
	}
	if all {
		headings = append(headings, "Config")
	}
	table.SetHeader(headings)
	table.SetAlignment(tablewriter.ALIGN_LEFT)

	for _, s := range sgs {
		statuses, err := c.getStates(s)
		if err != nil {
			return "", errors.WithStack(err)
		}
		for _, status := range statuses {
			row := []string{
				strconv.Itoa(status.command.Pid),
				status.command.Service.Name,
				string(status.status.State),
				strings.Join(status.status.Ports, ","),
				strconv.Itoa(status.status.StdoutLines) + " lines",
				strconv.Itoa(status.status.StderrLines) + " lines",
				status.status.StartTime.Format("2006-01-02 15:04:05"),
			}
			if all {
				configPath := status.command.Service.ConfigFile
				wd, err := os.Getwd()
				if err == nil {
					relativePath, err := filepath.Rel(wd, configPath)
					if err == nil && len(configPath) > len(relativePath) {
						configPath = relativePath
					}
				}
				row = append(row, configPath)
			}
			table.Append(row)
		}
	}
	table.Render()
	return buf.String(), nil
}

type statusCommandTuple struct {
	status  instance.Status
	command *services.ServiceCommand
}

func (c *Client) getStates(s services.ServiceOrGroup) ([]statusCommandTuple, error) {
	if service, ok := s.(*services.ServiceConfig); ok {
		command, err := service.GetCommand(services.ContextOverride{})
		if err != nil {
			return nil, errors.WithMessage(err, "could not get service command")
		}
		statuses, err := instance.LoadStatusForService(service, home.EdwardConfig.StateDir)
		if status, ok := statuses[command.InstanceId]; ok {
			return []statusCommandTuple{
				statusCommandTuple{
					status:  status,
					command: command,
				},
			}, nil
		}
		return nil, nil
	}
	var stateList []statusCommandTuple
	if group, ok := s.(*services.ServiceGroupConfig); ok {
		for _, service := range group.Services {
			serviceStates, _ := c.getStates(service)
			stateList = append(stateList, serviceStates...)
		}
	}
	return stateList, nil
}

func (c *Client) getServiceList(names []string, all bool) ([]services.ServiceOrGroup, error) {
	var sgs []services.ServiceOrGroup
	var err error

	if all {
		runningServices, err := services.LoadRunningServices()
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
