package services

import (
	"bytes"
	"errors"
	"syscall"

	"github.com/fatih/color"
	"github.com/yext/edward/common"
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
	command := sc.GetCommand()
	return command.BuildSync()
}

func (sc ServiceConfig) Start() error {
	command := sc.GetCommand()
	return command.StartAsync()
}

func (sc ServiceConfig) Stop() error {
	printOperation("Stopping " + sc.Name)
	command := sc.GetCommand()

	var scriptErr error = nil
	if command.Scripts.Stop != "" {
		scriptErr = command.StopScript()
	}

	if command.Pid == 0 {
		printResult("Not running", color.FgYellow)
		return errors.New(sc.Name + " is not running")
	}

	pgid, err := syscall.Getpgid(command.Pid)
	if err != nil {
		printResult("Not found", color.FgRed)
		return err
	}
	// TODO: Allow stronger override
	syscall.Kill(-pgid, syscall.SIGKILL) //syscall.SIGINT)

	// Remove leftover files
	command.clearState()

	if scriptErr == nil {
		printResult("OK", color.FgGreen)
	} else {
		// TODO: For some reason this is never called
		printResult("Script failed, kill signal succeeded", color.FgYellow)
	}

	return nil
}

func (sc ServiceConfig) GetStatus() []ServiceStatus {
	command := sc.GetCommand()

	status := "STOPPED"
	if command.Pid != 0 {
		status = "RUNNING"
	}

	return []ServiceStatus{
		ServiceStatus{
			Service: &sc,
			Status:  status,
		},
	}
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
