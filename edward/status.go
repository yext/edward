package edward

import (
	"bytes"
	"strconv"
	"strings"

	"github.com/olekukonko/tablewriter"
	"github.com/pkg/errors"
	"github.com/yext/edward/config"
	"github.com/yext/edward/services"
)

func (c *Client) Status(names []string) (string, error) {
	var sgs []services.ServiceOrGroup
	var err error
	if len(names) == 0 {
		for _, service := range config.GetAllServicesSorted() {
			var s []services.ServiceStatus
			s, err = service.Status()
			if err != nil {
				return "", errors.WithStack(err)
			}
			for _, status := range s {
				if status.Status != services.StatusStopped {
					sgs = append(sgs, service)
				}
			}
		}
		if len(sgs) == 0 {
			return "No services are running\n", nil
		}
	} else {

		sgs, err = config.GetServicesOrGroups(names)
		if err != nil {
			return "", errors.WithStack(err)
		}
	}

	if len(sgs) == 0 {
		return "No services found\n", nil
	}

	buf := new(bytes.Buffer)

	table := tablewriter.NewWriter(buf)
	table.SetHeader([]string{
		"Name",
		"Status",
		"PID",
		"Ports",
		"Stdout",
		"Stderr",
		"Start Time",
	})
	table.SetAlignment(tablewriter.ALIGN_LEFT)

	for _, s := range sgs {
		statuses, err := s.Status()
		if err != nil {
			return "", errors.WithStack(err)
		}
		for _, status := range statuses {
			table.Append([]string{
				status.Service.Name,
				status.Status,
				strconv.Itoa(status.Pid),
				strings.Join(status.Ports, ", "),
				strconv.Itoa(status.StdoutCount) + " lines",
				strconv.Itoa(status.StderrCount) + " lines",
				status.StartTime.Format("2006-01-02 15:04:05"),
			})
		}
	}
	table.Render()
	return buf.String(), nil
}
