package terminal

import (
	"os"
	"path/filepath"
	"strconv"
	"strings"

	humanize "github.com/dustin/go-humanize"
	"github.com/olekukonko/tablewriter"
	"github.com/theothertomelliott/gopsutil-nocgo/process"
	"github.com/yext/edward/ui"
)

func (p *Provider) Status(statuses []ui.ServiceStatus) {
	var (
		configByService []string
		configSet       = make(map[string]struct{})
	)
	for _, serviceStatus := range statuses {
		configPath := serviceStatus.Service().ConfigFile
		wd, err := os.Getwd()
		if err == nil {
			relativePath, err := filepath.Rel(wd, configPath)
			if err == nil && len(configPath) > len(relativePath) {
				configPath = relativePath
			}
		}
		configByService = append(configByService, configPath)
		configSet[configPath] = struct{}{}
	}

	table := tablewriter.NewWriter(os.Stdout)
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
	if len(configSet) > 1 {
		headings = append(headings, "Config File")
	}

	table.SetHeader(headings)
	table.SetAlignment(tablewriter.ALIGN_LEFT)

	for index, serviceStatus := range statuses {
		service := serviceStatus.Service()
		status := serviceStatus.Status()

		memoryInfo := status.MemoryInfo
		if memoryInfo == nil {
			memoryInfo = &process.MemoryInfoStat{}
		}

		row := []string{
			strconv.Itoa(serviceStatus.Pid()),
			service.Name,
			string(status.State),
			strings.Join(status.Ports, ","),
			strconv.Itoa(status.StdoutLines) + " lines",
			strconv.Itoa(status.StderrLines) + " lines",
			humanize.Bytes(memoryInfo.RSS),
			humanize.Bytes(memoryInfo.VMS),
			humanize.Bytes(memoryInfo.Swap),
			status.StartTime.Format("2006-01-02 15:04:05"),
		}

		if len(configSet) > 1 {
			row = append(row, configByService[index])
		}
		table.Append(row)
	}
	table.Render()
}
