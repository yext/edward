package services

import (
	"bufio"
	"encoding/json"
	"io/ioutil"
	"os"
	"path"
	"runtime"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/pkg/errors"
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
		return errors.Wrap(err, "could not parse service config")
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

	return errors.WithStack(sc.validate())
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
		return errors.WithStack(err)
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

func (sc *ServiceConfig) Build(cfg OperationConfig) error {
	if cfg.IsExcluded(sc) {
		return nil
	}

	command, err := sc.GetCommand()
	if err != nil {
		return errors.WithStack(err)
	}
	return errors.WithStack(command.BuildSync(false))
}

func (sc *ServiceConfig) Launch(cfg OperationConfig) error {
	if cfg.IsExcluded(sc) {
		return nil
	}

	command, err := sc.GetCommand()
	if err != nil {
		return errors.WithStack(err)
	}
	return errors.WithStack(command.StartAsync(cfg))
}

func (sc *ServiceConfig) Start(cfg OperationConfig) error {
	if cfg.IsExcluded(sc) {
		return nil
	}

	err := sc.Build(cfg)
	if err != nil {
		return errors.WithStack(err)
	}
	err = sc.Launch(cfg)
	return errors.WithStack(err)
}

func (sc *ServiceConfig) Stop(cfg OperationConfig) error {
	if cfg.IsExcluded(sc) {
		return nil
	}

	tracker := CommandTracker{
		Name:       "Stopping " + sc.Name,
		Logger:     sc.Logger,
		OutputFile: "",
	}
	tracker.Start()

	command, err := sc.GetCommand()
	if err != nil {
		return errors.WithStack(err)
	}

	if command.Pid == 0 {
		tracker.SoftFail(errors.New("Not running"))
		return nil
	}

	stopped, err := sc.interruptProcess(cfg, command)
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
			stopped, err := sc.killProcess(cfg, command)
			if err != nil {
				tracker.Fail(err)
				return nil
			}
			if stopped {
				tracker.SoftFail(errors.New("Killed"))
				return nil
			}
			tracker.Fail(errors.New("Process was not killed"))
			return nil
		}
	}

	// Remove leftover files
	command.clearState()
	tracker.Success()
	return nil
}

func (sc *ServiceConfig) interruptProcess(cfg OperationConfig, command *ServiceCommand) (success bool, err error) {

	p, err := process.NewProcess(int32(command.Pid))
	if err != nil {
		return false, errors.WithStack(err)
	}
	err = p.SendSignal(syscall.SIGINT)
	if err != nil {
		return false, errors.WithStack(err)
	}

	// Check to see if the process is still running
	exists, err := process.PidExists(int32(command.Pid))
	if err != nil {
		return false, errors.WithStack(err)
	}
	return !exists, nil
}

func (sc *ServiceConfig) killProcess(cfg OperationConfig, command *ServiceCommand) (success bool, err error) {
	pgid, err := syscall.Getpgid(command.Pid)
	if err != nil {
		return false, errors.WithStack(err)
	}

	if pgid == 0 || pgid == 1 {
		return false, errors.WithStack(errors.New("suspect pgid: " + strconv.Itoa(pgid)))
	}

	err = command.killGroup(cfg, pgid, false)
	return true, errors.WithStack(err)
}

func waitForTerm(command *ServiceCommand, timeout time.Duration) (bool, error) {
	for elapsed := time.Duration(0); elapsed <= timeout; elapsed += time.Millisecond * 100 {
		exists, err := process.PidExists(int32(command.Pid))
		if err != nil {
			return false, errors.WithStack(err)
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
		return nil, errors.WithStack(err)
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
			return nil, errors.WithStack(err)
		}
		epochStart, err := proc.CreateTime()
		if err != nil {
			return nil, errors.WithStack(err)
		}
		status.StartTime = time.Unix(epochStart/1000, 0)
		status.Ports, err = sc.getPorts(proc)
		status.StdoutCount, status.StderrCount = sc.getLogCounts()
		if err != nil {
			return nil, errors.WithStack(err)
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
		return nil, errors.WithStack(err)
	}
	if sc.LaunchChecks != nil {
		for _, port := range sc.LaunchChecks.Ports {
			ports = append(ports, strconv.Itoa(port))
		}
	}
	return ports, nil
}

func (sc *ServiceConfig) getLogCounts() (int, int) {
	logFile, err := os.Open(sc.GetRunLog())
	if err != nil {
		return 0, 0
	}
	defer logFile.Close()
	scanner := bufio.NewScanner(logFile)
	var stdoutCount int
	var stderrCount int
	var lineData struct{ Stream string }
	for scanner.Scan() {
		text := scanner.Text()
		err := json.Unmarshal([]byte(text), &lineData)
		if err != nil {
			continue
		}
		if lineData.Stream == "stdout" {
			stdoutCount++
		}
		if lineData.Stream == "stderr" {
			stderrCount++
		}
	}
	return stdoutCount, stderrCount
}

func (sc *ServiceConfig) doGetPorts(proc *process.Process) ([]string, error) {
	var err error
	if len(connectionsCache) == 0 {
		connectionsCache, err = net.Connections("all")
		if err != nil {
			return nil, errors.WithStack(err)
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

func (sc *ServiceConfig) IsSudo(cfg OperationConfig) bool {
	if cfg.IsExcluded(sc) {
		return false
	}

	return sc.RequiresSudo
}

func (s *ServiceConfig) GetRunLog() string {
	dir := home.EdwardConfig.LogDir
	return path.Join(dir, s.Name+".log")
}

func (s *ServiceConfig) GetCommand() (*ServiceCommand, error) {

	s.printf("Building control command for: %v\n", s.Name)
	command := &ServiceCommand{
		Service: s,
		Logger:  s.Logger,
	}

	// Retrieve the PID if available
	pidFile := command.getPidPath()
	s.printf("Checking pidfile for %v: %v\n", s.Name, pidFile)
	if _, err := os.Stat(pidFile); err == nil {
		dat, err := ioutil.ReadFile(pidFile)
		if err != nil {
			return nil, errors.WithStack(err)
		}
		pid, err := strconv.Atoi(string(dat))
		if err != nil {
			return nil, errors.WithStack(err)
		}
		command.Pid = pid

		exists, err := process.PidExists(int32(command.Pid))
		if err != nil {
			return nil, errors.WithStack(err)
		}
		if !exists {
			s.printf("Process for %v was not found, resetting.\n", s.Name)
			command.clearState()
		}

		proc, err := process.NewProcess(int32(command.Pid))
		if err != nil {
			return nil, errors.WithStack(err)
		}
		cmdline, err := proc.Cmdline()
		if err != nil {
			return nil, errors.WithStack(err)
		}
		if !strings.Contains(cmdline, s.Name) {
			s.printf("Process for %v was not as expected (found %v), resetting.\n", s.Name, cmdline)
			command.clearState()
		}

	} else {
		s.printf("No pidfile for %v", s.Name)
	}

	return command, nil
}
