package main

import (
	"bytes"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"path"
	"strconv"
	"strings"
	"sync"
	"syscall"

	"github.com/fatih/color"
	"github.com/hpcloud/tail"
)

var _ ServiceOrGroup = ServiceGroupConfig{}
var _ ServiceOrGroup = ServiceConfig{}

type ServiceOrGroup interface {
	Build() error
	Start() error
	Stop() error
	GetStatus() []ServiceStatus
}

type ServiceStatus struct {
	Service *ServiceConfig
	Status  string
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
	// Groups on which this group depends
	Groups []*ServiceGroupConfig
}

func (sg ServiceGroupConfig) Build() error {
	println("Building group: ", sg.Name)
	for _, group := range sg.Groups {
		err := group.Build()
		if err != nil {
			return err
		}
	}
	for _, service := range sg.Services {
		err := service.Build()
		if err != nil {
			return err
		}
	}
	return nil
}

func (sg ServiceGroupConfig) Start() error {
	println("Starting group:", sg.Name)
	for _, group := range sg.Groups {
		err := group.Start()
		if err != nil {
			// Always fail if any services in a dependant group failed
			return err
		}
	}
	var outErr error = nil
	for _, service := range sg.Services {
		err := service.Start()
		if err != nil {
			return err
		}
	}
	return outErr
}

func (sg ServiceGroupConfig) Stop() error {
	println("=== Group:", sg.Name, "===")
	// TODO: Do this in reverse
	for _, service := range sg.Services {
		_ = service.Stop()
	}
	for _, group := range sg.Groups {
		_ = group.Stop()
	}
	return nil
}

func (sg ServiceGroupConfig) GetStatus() []ServiceStatus {
	var outStatus []ServiceStatus
	for _, service := range sg.Services {
		outStatus = append(outStatus, service.GetStatus()...)
	}
	for _, group := range sg.Groups {
		outStatus = append(outStatus, group.GetStatus()...)
	}
	return outStatus
}

// ServiceConfig represents a service that can be managed by Edward
type ServiceConfig struct {
	// Service name, used to identify in commands
	Name string
	// Optional path to service. If nil, uses cwd
	Path *string
	// Commands for managing the service
	Commands ServiceConfigCommands
	// Service state properties that can be obtained from logs
	Properties ServiceConfigProperties
}

type ServiceConfigProperties struct {
	// Regex to detect a line indicating the service has started successfully
	Started string
	// Custom properties, mapping a property name to a regex
	Custom map[string]string
}

type ServiceConfigCommands struct {
	// Command to build
	Build string
	// Command to launch
	Launch string
	// Optional command to stop
	Stop string
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

	command.clearPid()

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

type ServiceCommand struct {
	// Parent service config
	Service *ServiceConfig
	// Path to string
	Scripts struct {
		Build  string
		Launch string
		Stop   string
	}
	Pid  int
	Logs struct {
		Build string
		Run   string
		Stop  string
	}

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

func printOperation(operation string) {
	print(operation, "...\t")
}

func printResult(message string, c color.Attribute) {
	print("[")
	color.Set(c)
	print(message)
	color.Unset()
	println("]")
}

func printFile(path string) {
	dat, errRead := ioutil.ReadFile(path)
	if errRead != nil {
		log.Println(errRead)
	}
	fmt.Print(string(dat))
}

func (sc *ServiceCommand) BuildSync() error {
	printOperation("Building " + sc.Service.Name)

	if sc.Pid != 0 {
		printResult("Already running", color.FgYellow)
		return nil
	}

	if sc.Scripts.Build == "" {
		printResult("No build", color.FgGreen)
		return nil
	}

	file, err := sc.createScript(sc.Scripts.Build)
	// Build the project and wait for completion
	if err != nil {
		return err
	}
	defer os.Remove(file.Name())

	cmd := exec.Command(file.Name())
	err = cmd.Run()
	if err != nil {
		printResult("Failed", color.FgRed)
		printFile(sc.Logs.Build)
		return err
	}

	printResult("OK", color.FgGreen)

	return nil
}

func (sc *ServiceCommand) waitUntilLive(command *exec.Cmd) error {

	var err error = nil
	var wg sync.WaitGroup
	wg.Add(1)

	go func() {
		// Read output until we get the success
		var t *tail.Tail
		t, err = tail.TailFile(sc.Logs.Run, tail.Config{Follow: true, Logger: tail.DiscardingLogger})
		for line := range t.Lines {
			if strings.Contains(line.Text, sc.Service.Properties.Started) {
				wg.Done()
				return
			}
		}
	}()

	go func() {
		// Wait until the process exists
		command.Wait()
		err = errors.New("Command failed!")
		wg.Done()
	}()

	wg.Wait()

	return err
}

func (sc *ServiceCommand) StartAsync() error {

	printOperation("Launching " + sc.Service.Name)

	if sc.Pid != 0 {
		printResult("Already running", color.FgYellow)
		return nil
	}
	// Clear logs
	os.Remove(sc.Logs.Run)

	// Start the project and get the PID
	file, err := sc.createScript(sc.Scripts.Launch)
	if err != nil {
		return err
	}
	//defer os.Remove(file.Name())

	cmd := exec.Command(file.Name())
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	err = cmd.Start()
	if err != nil {
		printResult("Failed", color.FgRed)
		return err
	}

	pid := cmd.Process.Pid

	pidStr := strconv.Itoa(pid)
	f, err := os.Create(sc.getPidPath())
	if err != nil {
		return err
	}
	f.WriteString(pidStr)
	f.Close()

	err = sc.waitUntilLive(cmd)
	if err == nil {
		printResult("OK", color.FgGreen)
	} else {
		printResult("Failed!", color.FgRed)
		printFile(sc.Logs.Run)
	}
	return err
}

func (sc *ServiceCommand) StopScript() error {

	// Start the project and get the PID
	file, err := sc.createScript(sc.Scripts.Stop)
	if err != nil {
		return err
	}
	defer os.Remove(file.Name())

	cmd := exec.Command(file.Name())
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	err = cmd.Run()
	return err
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
	buffer.WriteString("> ")
	buffer.WriteString(logPath)
	buffer.WriteString(" 2>&1")
	buffer.WriteString("\n")

	return buffer.String()
}

func (sc *ServiceCommand) clearPid() {
	sc.Pid = 0
	os.Remove(sc.getPidPath())
}

func (sc *ServiceCommand) getPidPath() string {
	dir := EdwardConfig.PidDir
	return path.Join(dir, sc.Service.Name+".pid")
}

func (s *ServiceConfig) GetCommand() *ServiceCommand {

	dir := EdwardConfig.LogDir

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
		Logs: logs,
	}

	// Retrieve the PID if available
	pidFile := command.getPidPath()
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

		// TODO: Check this PID is actually live
		process, err := os.FindProcess(int(pid))
		if err != nil {
			command.clearPid()
		} else {
			err := process.Signal(syscall.Signal(0))
			if err != nil {
				command.clearPid()
			}
		}
	}
	// TODO: Set status

	return command
}
