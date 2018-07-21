package terminal

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	humanize "github.com/dustin/go-humanize"
	"github.com/olekukonko/tablewriter"
	"github.com/theothertomelliott/gopsutil-nocgo/process"
	"github.com/yext/edward/services"
	"github.com/yext/edward/ui"
)

var _ ui.Provider = &Provider{}

type Provider struct {
}

func (p *Provider) Infof(format string, args ...interface{}) {
	fmt.Printf(format, args...)
	fmt.Println()
}

func (p *Provider) Errorf(format string, args ...interface{}) {
	fmt.Printf(format, args...)
	fmt.Println()
}

func (p *Provider) Confirm(format string, args ...interface{}) bool {
	reader := bufio.NewReader(os.Stdin)
	for {
		fmt.Printf(format, args...)
		fmt.Print(" [y/n]?")

		response, err := reader.ReadString('\n')
		if err != nil {
			return false
		}

		response = strings.ToLower(strings.TrimSpace(response))

		if response == "y" || response == "yes" {
			return true
		} else if response == "n" || response == "no" {
			return false
		}
	}
}

func (p *Provider) List(services []services.ServiceOrGroup, groups []services.ServiceOrGroup) {
	p.Infof("Services and groups")
	p.Infof("Groups:")
	for _, g := range groups {
		if g.GetDescription() != "" {
			p.Infof("\t%v: %v", g.GetName(), g.GetDescription())
		} else {
			p.Infof("\t%v", g.GetName())
		}
	}
	p.Infof("Services:")
	for _, s := range services {
		if s.GetDescription() != "" {
			p.Infof("\t%v: %v", s.GetName(), s.GetDescription())
		} else {
			p.Infof("\t%v", s.GetName())
		}
	}
}

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
