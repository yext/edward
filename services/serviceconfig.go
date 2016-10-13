package services

import (
	"bytes"
	"io/ioutil"
	"os"
	"path"
	"runtime"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/juju/errgo"
	"github.com/shirou/gopsutil/process"
	"github.com/yext/edward/common"
	"github.com/yext/edward/home"
)

var _ ServiceOrGroup = &ServiceConfig{}

// ServiceConfig represents a service that can be managed by Edward
type ServiceConfig struct {
	// Service name, used to identify in commands
	Name string `json:"name"`
	// Optional path to service. If nil, uses cwd
	Path *string `json:"path,omitempty"`
	// Does this service require sudo privileges?
	RequiresSudo bool `json:"requiresSudo,omitempty"`
	// Commands for managing the service
	Commands ServiceConfigCommands `json:"commands"`
	// Service state properties that can be obtained from logs
	Properties ServiceConfigProperties `json:"log_properties,omitempty"`

	// Env holds environment variables for a service, for example: GOPATH=~/gocode/
	// These will be added to the vars in the environment under which the Edward command was run
	Env []string `json:"env,omitempty"`

	Platform string `json:"platform,omitempty"`

	Logger common.Logger `json:"-"`

	// Path to watch for updates, relative to config file. If specified, will enable hot reloading.
	Watch *string `json:"watch,omitempty"`
}

// MatchesPlatform determines whether or not this service can be run on the current OS
func (c *ServiceConfig) MatchesPlatform() bool {
	return len(c.Platform) == 0 || c.Platform == runtime.GOOS
}

func (c *ServiceConfig) printf(format string, v ...interface{}) {
	if c.Logger == nil {
		return
	}
	c.Logger.Printf(format, v...)
}

type ServiceConfigProperties struct {
	// Regex to detect a line indicating the service has started successfully
	Started string `json:"started,omitempty"`
	// Custom properties, mapping a property name to a regex
	Custom map[string]string `json:"-"`
}

type ServiceConfigCommands struct {
	// Command to build
	Build string `json:"build,omitempty"`
	// Command to launch
	Launch string `json:"launch,omitempty"`
	// Optional command to stop
	Stop string `json:"stop,omitempty"`
}

func (sc *ServiceConfig) GetName() string {
	return sc.Name
}

func (sc *ServiceConfig) Build() error {
	command, err := sc.GetCommand()
	if err != nil {
		return errgo.Mask(err)
	}
	return errgo.Mask(command.BuildSync(false))
}

func (sc *ServiceConfig) Start() error {
	command, err := sc.GetCommand()
	if err != nil {
		return errgo.Mask(err)
	}
	return errgo.Mask(command.StartAsync())
}

func (sc *ServiceConfig) Stop() error {
	tracker := CommandTracker{
		Name:       "Stopping " + sc.Name,
		Logger:     sc.Logger,
		OutputFile: "",
	}
	tracker.Start()

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
		tracker.SoftFail(errgo.New("Not running"))
		return nil
	}

	stopped, err := sc.stopProcess(command, true)
	if err != nil {
		tracker.Fail(err)
		return nil
	}

	if !stopped {
		sc.printf("SIGINT failed to stop service, waiting for 5s before sending SIGKILL\n")
		stopped, err := waitForTerm(command, time.Second*5)
		if err != nil {
			tracker.Fail(err)
			return nil
		}
		if !stopped {
			stopped, err := sc.stopProcess(command, false)
			if err != nil {
				tracker.Fail(err)
				return nil
			}
			if stopped {
				tracker.SoftFail(errgo.New("Killed"))
				return nil
			}
			tracker.Fail(errgo.New("Process was not killed"))
			return nil
		}
	}

	// Remove leftover files
	command.clearState()

	if scriptErr == nil {
		tracker.Success()
	} else {
		tracker.SoftFail(errgo.New("Script failed, kill signal succeeded"))
	}

	return nil
}

func (sc *ServiceConfig) stopProcess(command *ServiceCommand, graceful bool) (success bool, err error) {
	pgid, err := syscall.Getpgid(command.Pid)
	if err != nil {
		return false, errgo.Mask(err)
	}

	if pgid == 0 || pgid == 1 {
		return false, errgo.Mask(errgo.New("suspect pgid: " + strconv.Itoa(pgid)))
	}

	err = command.killGroup(pgid, graceful)
	if err != nil {
		return false, errgo.Mask(err)
	}

	// Check to see if the process is still running
	exists, err := process.PidExists(int32(command.Pid))
	if err != nil {
		return false, errgo.Mask(err)
	}

	return !exists, nil
}

func waitForTerm(command *ServiceCommand, timeout time.Duration) (bool, error) {
	for elapsed := time.Duration(0); elapsed <= timeout; elapsed += time.Millisecond * 100 {
		exists, err := process.PidExists(int32(command.Pid))
		if err != nil {
			return false, errgo.Mask(err)
		}
		if !exists {
			return true, nil
		}
		time.Sleep(time.Millisecond * 100)
	}
	return false, nil
}

func (sc *ServiceConfig) Status() ([]ServiceStatus, error) {
	command, err := sc.GetCommand()
	if err != nil {
		return nil, errgo.Mask(err)
	}

	status := ServiceStatus{
		Service: sc,
		Status:  "STOPPED",
	}

	if command.Pid != 0 {
		status.Status = "RUNNING"
		status.Pid = command.Pid
		proc, err := process.NewProcess(int32(command.Pid))
		if err != nil {
			return nil, errgo.Mask(err)
		}
		epochStart, err := proc.CreateTime()
		if err != nil {
			return nil, errgo.Mask(err)
		}
		status.StartTime = time.Unix(epochStart/1000, 0)
	}

	return []ServiceStatus{
		status,
	}, nil
}

func (sc *ServiceConfig) IsSudo() bool {
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

func (s *ServiceConfig) GetWatchDirs() map[string]*ServiceConfig {
	if s.Watch != nil {
		return map[string]*ServiceConfig{*s.Watch: s}
	}
	return map[string]*ServiceConfig{}
}
