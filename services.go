package main

import (
	"bytes"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"strconv"
	"syscall"
)

type ServiceOrGroup interface {
	Start() error
	Stop() error
	Restart() error
}

type ServiceConfigFile struct {
	Services []ServiceConfig
	Groups   []ServiceGroupConfig
}

// ServiceGroupConfig is a group of services that can be managed together
type ServiceGroupConfig struct {
	// A name for this group, used to identify it in commands
	Name string
	// Paths to child service config files
	ServicePaths []string
	// Full services contained within this group
	Services []*ServiceConfig
}

// ServiceConfig represents a service that can be managed by Edward
type ServiceConfig struct {
	// Service name, used to identify in commands
	Name string
	// Optional path to service. If nil, uses cwd
	Path *string
	// Commands for managing the service
	Commands struct {
		// Command to build
		Build string
		// Command to launch
		Launch string
	}
	// Service state properties that can be obtained from logs
	Properties struct {
		// Regex to detect a line indicating the service has started successfully
		Started string
		// Custom properties, mapping a property name to a regex
		Custom map[string]string
	}
}

func (sc *ServiceConfig) Start() error {
	command := sc.GetCommand()
	err := command.BuildSync()
	if err != nil {
		return nil
	}
	command.StartAsync()
	return nil
}

func (sc *ServiceConfig) Stop() error {
	command := sc.GetCommand()
	// TODO: Check that the service isn't running

	pgid, err := syscall.Getpgid(command.Pid)
	if err != nil {
		return err
	}
	syscall.Kill(-pgid, syscall.SIGINT)

	return nil
}

type ServiceCommand struct {
	// Parent service config
	Service *ServiceConfig
	// Path to string
	Scripts struct {
		Build  string
		Launch string
	}
	Pid int
	Log string

	// TODO: Add status
}

func (sc *ServiceCommand) createScript(content string) (*os.File, error) {
	file, err := ioutil.TempFile(os.TempDir(), sc.Service.Name)
	if err != nil {
		return nil, err
	}
	file.WriteString(content)

	err = os.Chmod(file.Name(), 0777)
	if err != nil {
		return nil, err
	}

	return file, nil
}

func (sc *ServiceCommand) BuildSync() error {

	println("Building...")

	file, err := sc.createScript(sc.Scripts.Build)
	// Build the project and wait for completion
	if err != nil {
		log.Fatal(err)
	}
	defer os.Remove(file.Name())

	cmd := exec.Command(file.Name())
	err = cmd.Run()
	if err != nil {
		log.Fatal(err)
	}

	return nil
}

func (sc *ServiceCommand) StartAsync() {

	println("Launching...")

	// Start the project and get the PID
	file, err := sc.createScript(sc.Scripts.Launch)
	if err != nil {
		log.Fatal(err)
	}
	//defer os.Remove(file.Name())

	cmd := exec.Command(file.Name())
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	err = cmd.Start()
	if err != nil {
		log.Fatal(err)
	}

	// TODO: Wait until process is live

	pid := cmd.Process.Pid
	println("Pid = ", pid)

	pidStr := strconv.Itoa(pid)
	f, err := os.Create(sc.Service.Name + ".pid")
	if err != nil {
		log.Fatal(err)
	}
	f.WriteString(pidStr)
	f.Close()
}

func (s *ServiceConfig) makeScript(command string, logPath string) string {
	var buffer bytes.Buffer
	buffer.WriteString("#!/bin/bash\n")
	buffer.WriteString("cd ")
	buffer.WriteString(*s.Path)
	buffer.WriteString("\n")
	buffer.WriteString(command)
	buffer.WriteString(" > ")
	buffer.WriteString(logPath + ".log")
	buffer.WriteString(" 2> ")
	buffer.WriteString(logPath + "-error.log")
	buffer.WriteString("\n")
	return buffer.String()
}

func (s *ServiceConfig) GetCommand() *ServiceCommand {

	dir, err := os.Getwd()
	if err != nil {
		log.Fatal(err)
	}

	buildScript := s.makeScript(s.Commands.Build, dir+"/"+s.Name+"-build")
	startScript := s.makeScript(s.Commands.Launch, dir+"/"+s.Name+"-run")
	command := &ServiceCommand{
		Service: s,
		Scripts: struct {
			Build  string
			Launch string
		}{Build: buildScript,
			Launch: startScript,
		},
	}

	// Retrieve the PID if available
	pidFile := s.Name + ".pid"
	if _, err := os.Stat(pidFile); err == nil {
		dat, err := ioutil.ReadFile(pidFile)
		if err != nil {
			log.Fatal(err)
		}
		pid, err := strconv.Atoi(string(dat))
		if err != nil {
			log.Fatal(err)
		}
		command.Pid = pid
	}
	// TODO: Set status

	return command
}

func (s *ServiceConfig) GetProcess() *exec.Cmd {
	//return &exec.Cmd{
	//	Path: "blah",
	//	Dir:  s.Path,
	//}
	return &exec.Cmd{}
}
