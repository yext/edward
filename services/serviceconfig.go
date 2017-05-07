package services

import (
	"bufio"
	"crypto/sha1"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"runtime"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/pkg/errors"
	"github.com/theothertomelliott/gopsutil-nocgo/net"
	"github.com/theothertomelliott/gopsutil-nocgo/process"
	"github.com/yext/edward/common"
	"github.com/yext/edward/home"
	"github.com/yext/edward/tracker"
	"github.com/yext/edward/warmup"
	"github.com/yext/edward/worker"
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

	// Path to watch for updates, relative to config file. If specified, will enable hot reloading.
	WatchJSON json.RawMessage `json:"watch,omitempty"`

	// Action for warming up this service
	Warmup *warmup.Warmup `json:"warmup,omitempty"`

	// Path to config file from which this service was loaded
	// This may be the file that imported the config containing the service definition.
	ConfigFile string `json:"-"`

	// Logger for actions on this service
	Logger common.Logger `json:"-"`
}

// UnmarshalJSON provides additional handling when unmarshaling a service from config.
// Currently, this handles legacy fields and fields with multiple possible types.
func (c *ServiceConfig) UnmarshalJSON(data []byte) error {
	type Alias ServiceConfig
	aux := &struct {
		Properties *ServiceConfigProperties `json:"log_properties,omitempty"`
		*Alias
	}{
		Alias: (*Alias)(c),
	}
	if err := json.Unmarshal(data, &aux); err != nil {
		return errors.Wrap(err, "could not parse service config")
	}
	if aux.Properties != nil {
		if c.LaunchChecks != nil {
			c.LaunchChecks.LogText = aux.Properties.Started
		} else {
			c.LaunchChecks = &LaunchChecks{
				LogText: aux.Properties.Started,
			}
		}
	}

	return errors.WithStack(c.validate())
}

// validate checks if this config is allowed
func (c *ServiceConfig) validate() error {
	if c.LaunchChecks != nil {
		if len(c.LaunchChecks.LogText) > 0 && len(c.LaunchChecks.Ports) > 0 {
			return errors.New("cannot specify both a log and port launch check")
		}
	}
	return nil
}

// SetWatch sets the watch configuration for this service
func (c *ServiceConfig) SetWatch(watch ServiceWatch) error {
	msg, err := json.Marshal(watch)
	if err != nil {
		return errors.WithStack(err)
	}
	c.WatchJSON = json.RawMessage(msg)
	return nil
}

// Watch returns the watch configuration for this service
func (c *ServiceConfig) Watch() ([]ServiceWatch, error) {
	var watch = ServiceWatch{
		Service: c,
	}

	if len(c.WatchJSON) == 0 {
		return nil, nil
	}

	var err error

	// Handle multiple
	err = json.Unmarshal(c.WatchJSON, &watch)
	if err == nil {
		return []ServiceWatch{watch}, nil
	}

	// Handle string version
	var include string
	err = json.Unmarshal(c.WatchJSON, &include)
	if err != nil {
		return nil, err
	}
	if include != "" {
		watch.IncludedPaths = append(watch.IncludedPaths, include)
		return []ServiceWatch{watch}, nil
	}

	return nil, nil
}

// ServiceWatch defines a set of directories to be watched for changes to a service's source.
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

// LaunchChecks defines the mechanism for testing whether a service has started successfully
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

// ServiceConfigCommands define the commands for building, launching and stopping a service
// All commands are optional
type ServiceConfigCommands struct {
	// Command to build
	Build string `json:"build,omitempty"`
	// Command to launch
	Launch string `json:"launch,omitempty"`
	// Optional command to stop
	Stop string `json:"stop,omitempty"`
}

// GetName returns the name for this service
func (c *ServiceConfig) GetName() string {
	return c.Name
}

// Build builds this service
func (c *ServiceConfig) Build(cfg OperationConfig, overrides ContextOverride, task tracker.Task) error {
	if cfg.IsExcluded(c) {
		return nil
	}

	command, err := c.GetCommand(overrides)
	if err != nil {
		return errors.WithStack(err)
	}
	return errors.WithStack(command.BuildSync(false, task))
}

// Launch launches this service
func (c *ServiceConfig) Launch(cfg OperationConfig, overrides ContextOverride, task tracker.Task, pool *worker.Pool) error {
	if cfg.IsExcluded(c) {
		return nil
	}

	command, err := c.GetCommand(overrides)
	if err != nil {
		return errors.WithStack(err)
	}

	err = pool.Enqueue(func() error {
		return errors.WithStack(command.StartAsync(cfg, overrides, task))
	})
	return errors.WithStack(err)
}

// Start builds then launches this service
func (c *ServiceConfig) Start(cfg OperationConfig, overrides ContextOverride, task tracker.Task, pool *worker.Pool) error {
	if cfg.IsExcluded(c) {
		return nil
	}

	err := c.Build(cfg, overrides, task)
	if err != nil {
		return errors.WithStack(err)
	}
	err = c.Launch(cfg, overrides, task, pool)
	return errors.WithStack(err)
}

// Stop stops this service
func (c *ServiceConfig) Stop(cfg OperationConfig, overrides ContextOverride, task tracker.Task, pool *worker.Pool) error {
	err := pool.Enqueue(func() error {
		return errors.WithStack(c.doStop(cfg, overrides, task))
	})
	return errors.WithStack(err)
}

func (c *ServiceConfig) doStop(cfg OperationConfig, overrides ContextOverride, task tracker.Task) error {
	if cfg.IsExcluded(c) || c.Commands.Launch == "" {
		return nil
	}

	job := task.Child(c.GetName()).Child("Stop")
	job.SetState(tracker.TaskStateInProgress)

	command, err := c.GetCommand(overrides)
	if err != nil {
		return errors.WithStack(err)
	}

	if command.Pid == 0 {
		job.SetState(tracker.TaskStateWarning, "Not running")
		return nil
	}

	stopped, err := c.interruptProcess(cfg, command)
	if err != nil {
		job.SetState(tracker.TaskStateFailed, err.Error())
		return nil
	}

	if !stopped {
		c.printf("SIGINT failed to stop service, waiting for 5s before sending SIGKILL\n")
		stopped, err := waitForTerm(command, time.Second*5)
		if err != nil {
			job.SetState(tracker.TaskStateFailed, err.Error())
			return nil
		}
		if !stopped {
			stopped, err := c.killProcess(cfg, command)
			if err != nil {
				job.SetState(tracker.TaskStateFailed, err.Error())
				return nil
			}
			if stopped {
				job.SetState(tracker.TaskStateWarning, "Killed")
				return nil
			}
			job.SetState(tracker.TaskStateFailed, "Process was not killed")
			return nil
		}
	}

	// Remove leftover files
	command.clearState()
	job.SetState(tracker.TaskStateSuccess)
	return nil
}

func (c *ServiceConfig) interruptProcess(cfg OperationConfig, command *ServiceCommand) (success bool, err error) {
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

func (c *ServiceConfig) killProcess(cfg OperationConfig, command *ServiceCommand) (success bool, err error) {
	pgid, err := syscall.Getpgid(command.Pid)
	if err != nil {
		return false, errors.WithStack(err)
	}

	if pgid == 0 || pgid == 1 {
		return false, errors.WithStack(errors.New("suspect pgid: " + strconv.Itoa(pgid)))
	}

	err = KillGroup(cfg, pgid, c)
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

// Status returns the status for this service
func (c *ServiceConfig) Status() ([]ServiceStatus, error) {
	command, err := c.GetCommand(ContextOverride{})
	if err != nil {
		return nil, errors.WithStack(err)
	}

	status := ServiceStatus{
		Service: c,
		Status:  StatusStopped,
	}

	if command.Pid != 0 {
		proc, err := process.NewProcess(int32(command.Pid))
		if err != nil {
			return nil, errors.WithStack(err)
		}
		epochStart, err := proc.CreateTime()
		if err != nil {
			if strings.HasPrefix(err.Error(), "exit status") {
				return []ServiceStatus{
					status,
				}, nil
			}
			return nil, errors.WithStack(err)
		}
		status.Status = StatusRunning
		status.Pid = command.Pid
		status.StartTime = time.Unix(epochStart/1000, 0)
		status.Ports, err = c.getPorts(proc)
		status.StdoutCount, status.StderrCount = c.getLogCounts()
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

func (c *ServiceConfig) getPorts(proc *process.Process) ([]string, error) {
	ports, err := c.doGetPorts(proc)
	if err != nil {
		return nil, errors.WithStack(err)
	}
	if c.LaunchChecks != nil {
		for _, port := range c.LaunchChecks.Ports {
			ports = append(ports, strconv.Itoa(port))
		}
	}
	return ports, nil
}

func (c *ServiceConfig) getLogCounts() (int, int) {
	logFile, err := os.Open(c.GetRunLog())
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

func (c *ServiceConfig) doGetPorts(proc *process.Process) ([]string, error) {
	var err error
	if len(connectionsCache) == 0 {
		connectionsCache, err = net.Connections("all")
		if err != nil {
			return nil, errors.WithStack(err)
		}
	}

	var ports []string
	var knownPorts = make(map[int]struct{})
	if c.LaunchChecks != nil {
		for _, port := range c.LaunchChecks.Ports {
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
		childPorts, err := c.doGetPorts(child)
		if err == nil {
			ports = append(ports, childPorts...)
		}
	}
	return ports, nil
}

// IsSudo returns true if this service requires sudo to run.
// If this service is excluded by cfg, then will always return false.
func (c *ServiceConfig) IsSudo(cfg OperationConfig) bool {
	if cfg.IsExcluded(c) {
		return false
	}

	return c.RequiresSudo
}

// GetRunLog returns the path to the run log for this service
func (c *ServiceConfig) GetRunLog() string {
	dir := home.EdwardConfig.LogDir
	return path.Join(dir, c.Name+".log")
}

// GetCommand returns the ServiceCommand for this service
func (c *ServiceConfig) GetCommand(overrides ContextOverride) (*ServiceCommand, error) {
	c.printf("Building control command for: %v\n", c.Name)
	command, err := LoadServiceCommand(c, overrides)
	return command, errors.WithStack(err)
}

func (c *ServiceConfig) getPid(command *ServiceCommand, pidFile string) (int, error) {
	dat, err := ioutil.ReadFile(pidFile)
	if err != nil {
		return 0, errors.WithStack(err)
	}
	pid, err := strconv.Atoi(string(dat))
	if err != nil {
		return 0, errors.WithStack(err)
	}

	exists, err := process.PidExists(int32(pid))
	if err != nil {
		return 0, errors.WithStack(err)
	}
	if !exists {
		c.printf("Process for %v was not found, resetting.\n", c.Name)
		command.clearState()
	}

	proc, err := process.NewProcess(int32(pid))
	if err != nil {
		return 0, errors.WithStack(err)
	}
	cmdline, err := proc.Cmdline()
	if err != nil {
		return 0, errors.WithStack(err)
	}
	if !strings.Contains(cmdline, c.Name) {
		c.printf("Process for %v was not as expected (found %v), resetting.\n", c.Name, cmdline)
		command.clearState()
	}
	return pid, nil
}

func (c *ServiceConfig) getStatePath() string {
	dir := home.EdwardConfig.StateDir
	name := c.Name
	hasher := sha1.New()
	hasher.Write([]byte(c.ConfigFile))
	sha := base64.URLEncoding.EncodeToString(hasher.Sum(nil))
	return path.Join(dir, fmt.Sprintf("%v.%v.state", sha, name))
}

func (c *ServiceConfig) GetPidPathLegacy() string {
	dir := home.EdwardConfig.PidDir
	name := c.Name
	return path.Join(dir, fmt.Sprintf("%v.pid", name))
}
