package services

import (
	"encoding/json"
	"errors"
	"io/ioutil"
	"os"
	"path"
	"runtime"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/juju/errgo"
	"github.com/shirou/gopsutil/net"
	"github.com/shirou/gopsutil/process"
	"github.com/yext/edward/common"
	"github.com/yext/edward/home"
	"github.com/yext/edward/warmup"
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

	// Checks to perform to ensure that a service has started correctly
	LaunchChecks *LaunchChecks `json:"launch_checks"`

	// Env holds environment variables for a service, for example: GOPATH=~/gocode/
	// These will be added to the vars in the environment under which the Edward command was run
	Env []string `json:"env,omitempty"`

	Platform string `json:"platform,omitempty"`

	Logger common.Logger `json:"-"`

	// Path to watch for updates, relative to config file. If specified, will enable hot reloading.
	WatchJson json.RawMessage `json:"watch,omitempty"`

	// Action for warming up this service
	Warmup *warmup.Warmup `json:"warmup,omitempty"`
}

// Handle legacy fields
func (sc *ServiceConfig) UnmarshalJSON(data []byte) error {
	type Alias ServiceConfig
	aux := &struct {
		Properties *ServiceConfigProperties `json:"log_properties,omitempty"`
		*Alias
	}{
		Alias: (*Alias)(sc),
	}
	if err := json.Unmarshal(data, &aux); err != nil {
		return err
	}
	if aux.Properties != nil {
		if sc.LaunchChecks != nil {
			sc.LaunchChecks.LogText = aux.Properties.Started
		} else {
			sc.LaunchChecks = &LaunchChecks{
				LogText: aux.Properties.Started,
			}
		}
	}

	return sc.validate()
}

// validate checks if this config is allowed
func (sc *ServiceConfig) validate() error {
	if sc.LaunchChecks != nil {
		if len(sc.LaunchChecks.LogText) > 0 && len(sc.LaunchChecks.Ports) > 0 {
			return errors.New("cannot specify both a log and port launch check")
		}
	}
	return nil
}

func (c *ServiceConfig) SetWatch(watch ServiceWatch) error {
	msg, err := json.Marshal(watch)
	if err != nil {
		return err
	}
	c.WatchJson = json.RawMessage(msg)
	return nil
}

func (c *ServiceConfig) Watch() ([]ServiceWatch, error) {
	var watch ServiceWatch = ServiceWatch{
		Service: c,
	}

	if len(c.WatchJson) == 0 {
		return nil, nil
	}

	var err error

	// Handle multiple
	err = json.Unmarshal(c.WatchJson, &watch)
	if err == nil {
		return []ServiceWatch{watch}, nil
	}

	// Handle string version
	var include string
	err = json.Unmarshal(c.WatchJson, &include)
	if err != nil {
		return nil, err
	}
	if include != "" {
		watch.IncludedPaths = append(watch.IncludedPaths, include)
		return []ServiceWatch{watch}, nil
	}

	return nil, nil
}

type ServiceWatch struct {
	Service       *ServiceConfig `json:"-"`
	IncludedPaths []string       `json:"include,omitempty"`
	ExcludedPaths []string       `json:"exclude,omitempty"`
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

type LaunchChecks struct {
	// A string to look for in the service's logs that indicates it has completed startup
	LogText string `json:"log_text,omitempty"`
	// One or more specific ports that are expected to be opened when this service starts
	Ports []int `json:"ports,omitempty"`
}

// ServiceConfigProperties provides a set of regexes to detect properties of a service
// Deprecated: This has been dropped in favour of LaunchChecks
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

func (sc *ServiceConfig) Launch() error {
	command, err := sc.GetCommand()
	if err != nil {
		return errgo.Mask(err)
	}
	err = command.StartAsync()
	if err != nil {
		return errgo.Mask(err)
	}
	return nil
}

func (sc *ServiceConfig) Start() error {
	err := sc.Build()
	if err != nil {
		return errgo.Mask(err)
	}
	err = sc.Launch()
	return errgo.Mask(err)
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
	if command.Scripts.Stop.WillRun() {
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
		status.Ports, err = sc.getPorts(proc)
		if err != nil {
			return nil, errgo.Mask(err)
		}
	}

	return []ServiceStatus{
		status,
	}, nil
}

// Connection list cache, created once per session.
var connectionsCache []net.ConnectionStat

func (sc *ServiceConfig) getPorts(proc *process.Process) ([]string, error) {
	ports, err := sc.doGetPorts(proc)
	if err != nil {
		return nil, errgo.Mask(err)
	}
	if sc.LaunchChecks != nil {
		for _, port := range sc.LaunchChecks.Ports {
			ports = append(ports, strconv.Itoa(port))
		}
	}
	return ports, nil
}

func (sc *ServiceConfig) doGetPorts(proc *process.Process) ([]string, error) {
	var err error
	if len(connectionsCache) == 0 {
		connectionsCache, err = net.Connections("all")
		if err != nil {
			return nil, errgo.Mask(err)
		}
	}

	var ports []string
	var knownPorts = make(map[int]struct{})
	if sc.LaunchChecks != nil {
		for _, port := range sc.LaunchChecks.Ports {
			knownPorts[port] = struct{}{}
		}
	}
	for _, connection := range connectionsCache {
		if connection.Status == "LISTEN" {
			if _, ok := knownPorts[int(connection.Laddr.Port)]; connection.Pid == proc.Pid && !ok {
				ports = append(ports, strconv.Itoa(int(connection.Laddr.Port)))
			}
		}
	}

	children, err := proc.Children()
	// This will error out if the process has finished or has no children
	if err != nil {
		return ports, nil
	}
	for _, child := range children {
		childPorts, err := sc.doGetPorts(child)
		if err == nil {
			ports = append(ports, childPorts...)
		}
	}
	return ports, nil
}

func (sc *ServiceConfig) IsSudo() bool {
	return sc.RequiresSudo
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

	path := ""
	if s.Path != nil {
		path = *s.Path
	}

	command := &ServiceCommand{
		Service: s,
		Scripts: struct {
			Build  Script
			Launch Script
			Stop   Script
		}{
			Build: Script{
				Path:    path,
				Command: s.Commands.Build,
				Log:     logs.Build,
			},
			Launch: Script{
				Path:    path,
				Command: s.Commands.Launch,
				Log:     logs.Run,
			},
			Stop: Script{
				Path:    path,
				Command: s.Commands.Stop,
				Log:     logs.Stop,
			},
		},
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
