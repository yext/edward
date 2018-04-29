package edward

import (
	"bytes"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	humanize "github.com/dustin/go-humanize"
	"github.com/olekukonko/tablewriter"
	"github.com/pkg/errors"
	"github.com/theothertomelliott/gopsutil-nocgo/process"
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
		"RSS",
		"VMS",
		"Swap",
		"Start Time",
	}
	if all {
		headings = append(headings, "Config")
	}
	table.SetHeader(headings)
	table.SetAlignment(tablewriter.ALIGN_LEFT)

	services := services.Services(sgs)
	for _, s := range services {
		statuses, err := c.getStates(s)
		if err != nil {
			return "", errors.WithStack(err)
		}
		for _, status := range statuses {
			if status.status.MemoryInfo == nil {
				status.status.MemoryInfo = &process.MemoryInfoStat{}
			}
			row := []string{
				strconv.Itoa(status.command.Pid),
				status.command.Service.Name,
				string(status.status.State),
				strings.Join(status.status.Ports, ","),
				strconv.Itoa(status.status.StdoutLines) + " lines",
				strconv.Itoa(status.status.StderrLines) + " lines",
				humanize.Bytes(status.status.MemoryInfo.RSS),
				humanize.Bytes(status.status.MemoryInfo.VMS),
				humanize.Bytes(status.status.MemoryInfo.Swap),
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
	command *instance.Instance
}

func (c *Client) getStates(service *services.ServiceConfig) ([]statusCommandTuple, error) {
	command, err := instance.Load(c.DirConfig, service, services.ContextOverride{})
	if err != nil {
		return nil, errors.WithMessage(err, "could not get service command")
	}

	// If the PID has been set to zero, the runner has died
	if command.Pid == 0 {
		return []statusCommandTuple{
			statusCommandTuple{
				status: instance.Status{
					State: instance.StateDied,
				},
				command: command,
			},
		}, nil
	}

	statuses, _ := instance.LoadStatusForService(service, c.DirConfig.StateDir)
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
