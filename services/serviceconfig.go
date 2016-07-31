package services

import (
	"bytes"
	"io/ioutil"
	"os"
	"path"
	"strconv"
	"strings"
	"syscall"

	"github.com/fatih/color"
	"github.com/shirou/gopsutil/process"
	"github.com/yext/edward/common"
	"github.com/yext/edward/home"
	"github.com/yext/errgo"
)

var _ ServiceOrGroup = ServiceConfig{}

// ServiceConfig represents a service that can be managed by Edward
type ServiceConfig struct {
	// Service name, used to identify in commands
	Name string `json:"name"`
	// Optional path to service. If nil, uses cwd
	Path *string `json:"path"`
	// Does this service require sudo privileges?
	RequiresSudo bool `json:"requiresSudo,omitempty"`
	// Commands for managing the service
	Commands ServiceConfigCommands `json:"commands"`
	// Service state properties that can be obtained from logs
	Properties ServiceConfigProperties `json:"log_properties"`

	// Env holds environment variables for a service, for example: GOPATH=~/gocode/
	// These will be added to the vars in the environment under which the Edward command was run
	Env []string

	Logger common.Logger `json:"-"`
}

func (c *ServiceConfig) printf(format string, v ...interface{}) {
	if c.Logger == nil {
		return
	}
	c.Logger.Printf(format, v...)
}

type ServiceConfigProperties struct {
	// Regex to detect a line indicating the service has started successfully
	Started string `json:"started"`
	// Custom properties, mapping a property name to a regex
	Custom map[string]string `json:"-"`
}

type ServiceConfigCommands struct {
	// Command to build
	Build string `json:"build"`
	// Command to launch
	Launch string `json:"launch"`
	// Optional command to stop
	Stop string `json:"stop"`
}

func (sc ServiceConfig) GetName() string {
	return sc.Name
}

func (sc ServiceConfig) Build() error {
	command, err := sc.GetCommand()
	if err != nil {
		return errgo.Mask(err)
	}
	return errgo.Mask(command.BuildSync())
}

func (sc ServiceConfig) Start() error {
	command, err := sc.GetCommand()
	if err != nil {
		return errgo.Mask(err)
	}
	return errgo.Mask(command.StartAsync())
}

func (sc ServiceConfig) Stop() error {
	printOperation("Stopping " + sc.Name)
	sc.printf("Stopping %v.\n", sc.Name)

	command, err := sc.GetCommand()
	if err != nil {
		return errgo.Mask(err)
	}

	var scriptErr error = nil
	if command.Scripts.Stop != "" {
		sc.printf("Running stop script for %v.\n", sc.Name)
		scriptErr = command.StopScript()
	}

	if command.Pid == 0 {
		printResult("Not running", color.FgYellow)
		sc.printf("%v is not running, PID == 0.\n", sc.Name)
		return errgo.New(sc.Name + " is not running")
	}

	pgid, err := syscall.Getpgid(command.Pid)
	if err != nil {
		printResult("Not found", color.FgRed)
		return errgo.Mask(err)
	}

	err = command.killGroup(pgid)
	if err != nil {
		printResult("Kill failed", color.FgRed)
		return errgo.Mask(err)
	}

	// Remove leftover files
	command.clearState()

	if scriptErr == nil {
		printResult("OK", color.FgGreen)
		sc.printf("%v stopped successfully.\n", sc.Name)
	} else {
		printResult("Script failed, kill signal succeeded", color.FgYellow)
		sc.printf("%v killed, but stop script failed: %v.\n", sc.Name, scriptErr)
	}

	return nil
}

func (sc ServiceConfig) Status() ([]ServiceStatus, error) {
	command, err := sc.GetCommand()
	if err != nil {
		return nil, errgo.Mask(err)
	}

	status := "STOPPED"
	if command.Pid != 0 {
		status = "RUNNING"
	}

	return []ServiceStatus{
		ServiceStatus{
			Service: &sc,
			Status:  status,
		},
	}, nil
}

func (sc ServiceConfig) IsSudo() bool {
	return sc.RequiresSudo
}

func (s *ServiceConfig) makeScript(command string, logPath string) string {
	if command == "" {
		return ""
	}

	var buffer bytes.Buffer
	buffer.WriteString("#!/bin/bash\n")
	if s.Path != nil {
		buffer.WriteString("cd ")
		buffer.WriteString(*s.Path)
		buffer.WriteString("\n")
	}
	buffer.WriteString(command)
	buffer.WriteString(" > ")
	buffer.WriteString(logPath)
	buffer.WriteString(" 2>&1")
	buffer.WriteString("\n")

	return buffer.String()
}

func (s *ServiceConfig) GetCommand() (*ServiceCommand, error) {

	s.printf("Building control command for: %v\n", s.Name)

	dir := home.EdwardConfig.LogDir

	logs := struct {
		Build string
		Run   string
		Stop  string
	}{
		Build: path.Join(dir, s.Name+"-build.log"),
		Run:   path.Join(dir, s.Name+".log"),
		Stop:  path.Join(dir, s.Name+"-stop.log"),
	}

	buildScript := s.makeScript(s.Commands.Build, logs.Build)
	startScript := s.makeScript(s.Commands.Launch, logs.Run)
	stopScript := s.makeScript(s.Commands.Stop, logs.Stop)
	command := &ServiceCommand{
		Service: s,
		Scripts: struct {
			Build  string
			Launch string
			Stop   string
		}{
			Build:  buildScript,
			Launch: startScript,
			Stop:   stopScript,
		},
		Logs:   logs,
		Logger: s.Logger,
	}

	// Retrieve the PID if available
	pidFile := command.getPidPath()
	s.printf("Checking pidfile for %v: %v\n", s.Name, pidFile)
	if _, err := os.Stat(pidFile); err == nil {
		dat, err := ioutil.ReadFile(pidFile)
		if err != nil {
			return nil, errgo.Mask(err)
		}
		pid, err := strconv.Atoi(string(dat))
		if err != nil {
			return nil, errgo.Mask(err)
		}
		command.Pid = pid

		exists, err := process.PidExists(int32(command.Pid))
		if err != nil {
			return nil, errgo.Mask(err)
		}
		if !exists {
			s.printf("Process for %v was not found, resetting.\n", s.Name)
			command.clearState()
		}

		proc, err := process.NewProcess(int32(command.Pid))
		if err != nil {
			return nil, errgo.Mask(err)
		}
		cmdline, err := proc.Cmdline()
		if err != nil {
			return nil, errgo.Mask(err)
		}
		if !strings.Contains(cmdline, s.Name) {
			s.printf("Process for %v was not as expected (found %v), resetting.\n", s.Name, cmdline)
			command.clearState()
		}

	} else {
		s.printf("No pidfile for %v", s.Name)
	}
	// TODO: Set status

	return command, nil
}
