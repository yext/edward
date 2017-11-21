package services

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"os/exec"
	"path"
	"strings"
	"syscall"

	"github.com/pkg/errors"
	uuid "github.com/satori/go.uuid"
	"github.com/theothertomelliott/gopsutil-nocgo/process"
	"github.com/yext/edward/commandline"
	"github.com/yext/edward/common"
	"github.com/yext/edward/home"
	"github.com/yext/edward/tracker"
	"github.com/yext/edward/warmup"
)

// StartupTimeoutSeconds is the amount of time in seconds that Edward will wait
// for a service to start before timing out
var StartupTimeoutSeconds = 30

// ServiceCommand provides state and functions for managing a service
type ServiceCommand struct {
	// Parent service config
	Service *ServiceConfig `json:"service"`
	// Pid of currently running instance
	Pid int `json:"pid"`
	// Config file from which this instance was launched
	ConfigFile string `json:"configFile"`
	// The edward version under which this instance was launched
	EdwardVersion string `json:"edwardVersion"`
	// Overrides applied by the group under which this service was started
	Overrides ContextOverride `json:"overrides,omitempty"`
	// Identifier for this instance of the service
	InstanceId string

	Logger common.Logger `json:"-"`
}

// LoadServiceCommand loads the command to control the specified service
func LoadServiceCommand(service *ServiceConfig, overrides ContextOverride) (command *ServiceCommand, err error) {
	command = &ServiceCommand{
		Service:    service,
		ConfigFile: service.ConfigFile,
		InstanceId: uuid.NewV4().String(),
	}
	defer func() {
		command.Service = service
		command.Logger = service.Logger
		command.EdwardVersion = common.EdwardVersion
		command.Overrides = command.Overrides.Merge(overrides)
		err = command.checkPid()
	}()

	legacyPidFile := service.GetPidPathLegacy()
	if _, err := os.Stat(legacyPidFile); err == nil {
		command.Pid, err = service.getPid(command, legacyPidFile)
		if err != nil {
			return nil, errors.WithStack(err)
		}
		return command, nil
	}

	stateFile := service.getStatePath()
	if _, err := os.Stat(stateFile); err == nil {
		raw, err := ioutil.ReadFile(stateFile)
		if err != nil {
			return nil, errors.WithStack(err)
		}
		json.Unmarshal(raw, command)
	}

	return command, nil
}

// Env provides the combined environment variables for this service command
func (c *ServiceCommand) Env() []string {
	return append(c.Service.Env, c.Overrides.Env...)
}

// Getenv returns the environment variable value for the provided key, if present.
// Env overrides are consulted first, followed by service env settings, then the os Env.
func (c *ServiceCommand) Getenv(key string) string {
	for _, env := range c.Overrides.Env {
		if strings.HasPrefix(env, key+"=") {
			return strings.Replace(env, key+"=", "", 1)
		}
	}
	for _, env := range c.Service.Env {
		if strings.HasPrefix(env, key+"=") {
			return strings.Replace(env, key+"=", "", 1)
		}
	}
	return os.Getenv(key)
}

func (c *ServiceCommand) checkPid() error {
	if c == nil || c.Pid == 0 {
		return nil
	}
	exists, err := process.PidExists(int32(c.Pid))
	if err != nil {
		return errors.WithStack(err)
	}
	if !exists {
		c.printf("Process for %v was not found, resetting.\n", c.Service.Name)
		c.clearState()
		return nil
	}

	proc, err := process.NewProcess(int32(c.Pid))
	if err != nil {
		return errors.WithStack(err)
	}
	cmdline, err := proc.Cmdline()
	if err != nil {
		return errors.WithStack(err)
	}
	if !strings.Contains(cmdline, c.Service.Name) {
		c.printf("Process for %v was not as expected (found %v), resetting.\n", c.Service.Name, cmdline)
		c.clearState()
	}
	return nil
}

// save will store the current state of this command to a state file
func (c *ServiceCommand) save() error {
	commandJSON, _ := json.Marshal(c)
	err := ioutil.WriteFile(c.Service.getStatePath(), commandJSON, 0644)
	if err != nil {
		return errors.WithStack(err)
	}
	return nil
}

func (c *ServiceCommand) printf(format string, v ...interface{}) {
	if c.Logger == nil {
		return
	}
	c.Logger.Printf(format, v...)
}

func (c *ServiceCommand) createScript(content string, scriptType string) (*os.File, error) {
	file, err := os.Create(path.Join(home.EdwardConfig.ScriptDir, c.Service.Name+"-"+scriptType))
	if err != nil {
		return nil, err
	}
	file.WriteString(content)
	file.Close()

	err = os.Chmod(file.Name(), 0777)
	if err != nil {
		return nil, err
	}

	return file, nil
}

func (c *ServiceCommand) deleteScript(scriptType string) error {
	return errors.WithStack(
		os.Remove(
			path.Join(home.EdwardConfig.ScriptDir, c.Service.Name+"-"+scriptType),
		),
	)
}

// BuildSync will buid the service synchronously.
// If force is false, the build will be skipped if the service is already running.
func (c *ServiceCommand) BuildSync(workingDir string, force bool, task tracker.Task) error {
	name := c.Service.GetName()
	t := task.Child(name)
	return errors.WithStack(c.BuildWithTracker(workingDir, force, t))
}

// BuildWithTracker builds a service.
// If force is false, the build will be skipped if the service is already running.
func (c *ServiceCommand) BuildWithTracker(workingDir string, force bool, task tracker.Task) error {
	if c.Service.Commands.Build == "" {
		return nil
	}
	if task == nil {
		return errors.New("task is nil")
	}
	job := task.Child("Build")
	job.SetState(tracker.TaskStateInProgress)

	if !force && c.Pid != 0 {
		job.SetState(tracker.TaskStateWarning, "Already running")
		return nil
	}

	cmd, err := c.constructCommand(workingDir, c.Service.Commands.Build)
	if err != nil {
		job.SetState(tracker.TaskStateFailed, err.Error())
		return errors.WithStack(err)
	}

	out, err := cmd.CombinedOutput()
	if err != nil {
		job.SetState(tracker.TaskStateFailed, err.Error(), string(out))
		return errors.WithMessage(err, "running build command")
	}

	job.SetState(tracker.TaskStateSuccess)
	return nil
}

func (c *ServiceCommand) constructCommand(workingDir string, command string) (*exec.Cmd, error) {
	command, cmdArgs, err := commandline.ParseCommand(os.Expand(command, c.Getenv))
	if err != nil {
		return nil, errors.WithStack(err)
	}

	cmd := exec.Command(command, cmdArgs...)
	cmd.Dir = buildAbsPath(workingDir, c.Service.Path)
	return cmd, nil
}

// StartAsync starts the service in the background
// Will block until the service is known to have started successfully.
// If the service fails to launch, an error will be returned.
func (c *ServiceCommand) StartAsync(cfg OperationConfig, task tracker.Task) error {
	if c.Service.Commands.Launch == "" {
		return nil
	}

	startTask := task.Child(c.Service.GetName()).Child("Start")
	startTask.SetState(tracker.TaskStateInProgress)

	if c.Pid != 0 {
		startTask.SetState(tracker.TaskStateWarning, "Already running")
		return nil
	}

	if c.Service.LaunchChecks != nil && len(c.Service.LaunchChecks.Ports) > 0 {
		inUse, err := c.areAnyListeningPortsOpen(c.Service.LaunchChecks.Ports)
		if err != nil {
			startTask.SetState(tracker.TaskStateFailed, err.Error())
			return errors.WithStack(err)
		}
		if inUse {
			inUseErr := errors.New("one or more of the ports required by this service are in use")
			startTask.SetState(tracker.TaskStateFailed, inUseErr.Error())
			return errors.WithStack(inUseErr)
		}
	}

	os.Remove(c.Service.GetRunLog())

	cmd, err := c.getLaunchCommand(cfg)
	if err != nil {
		startTask.SetState(tracker.TaskStateFailed, err.Error())
		return errors.WithStack(err)
	}
	cmd.Env = append(os.Environ(), c.Overrides.Env...)
	cmd.Env = append(cmd.Env, c.Service.Env...)

	err = cmd.Start()
	if err != nil {
		startTask.SetState(tracker.TaskStateFailed)
		return errors.WithStack(err)
	}

	c.Pid = cmd.Process.Pid

	c.printf("%v has PID: %d.\n", c.Service.Name, c.Pid)

	err = c.save()
	if err != nil {
		startTask.SetState(tracker.TaskStateFailed)
		return errors.WithStack(err)
	}

	err = WaitUntilLive(cmd, c.Service)
	if err == nil {
		startTask.SetState(tracker.TaskStateSuccess)
		warmup.Run(c.Service.Name, c.Service.Warmup, task)
		return nil
	}

	log, readingErr := logToStringSlice(c.Service.GetRunLog())
	if readingErr != nil {
		startTask.SetState(tracker.TaskStateFailed, "Could not read log", readingErr.Error(), fmt.Sprint("Original error: ", err.Error()))
	} else {
		startTask.SetState(tracker.TaskStateFailed, log...)
	}
	stopErr := c.Service.doStop(cfg, c.Overrides, task.Child("Cleanup"))
	if stopErr != nil {
		return errors.WithStack(stopErr)
	}
	return errors.WithStack(err)
}

func readAvailableLines(r io.ReadCloser) ([]string, error) {
	var out []string
	reader := bufio.NewReader(r)
	for reader.Buffered() > 0 {
		line, _, err := reader.ReadLine()
		if err != nil {
			return out, errors.WithStack(err)
		}
		out = append(out, string(line))
	}
	return nil, nil
}

func (c *ServiceCommand) getLaunchCommand(cfg OperationConfig) (*exec.Cmd, error) {
	command := cfg.EdwardExecutable
	cmdArgs := []string{
		"run",
	}
	if cfg.NoWatch {
		cmdArgs = append(cmdArgs, "--no-watch")
	}
	for _, tag := range cfg.Tags {
		cmdArgs = append(cmdArgs, "--tag", tag)
	}
	cmdArgs = append(cmdArgs, c.Service.Name)

	cmd := exec.Command(command, cmdArgs...)
	cmd.Dir = buildAbsPath(cfg.WorkingDir, c.Service.Path)
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	return cmd, nil
}

// RunStopScript will execute the stop script for this command, returning full output
// from running the script.
// Assumes the service has a stop script configured.
func (c *ServiceCommand) RunStopScript(workingDir string) ([]byte, error) {
	c.printf("Running stop script for %v\n", c.Service.Name)
	cmd, err := c.constructCommand(workingDir, c.Service.Commands.Stop)
	if err != nil {
		return nil, errors.WithStack(err)
	}

	out, err := cmd.CombinedOutput()
	if err != nil {
		return out, errors.WithStack(err)
	}
	return nil, nil
}

func (c *ServiceCommand) clearPid() {
	c.Pid = 0
	var err error
	_ = os.Remove(c.Service.GetPidPathLegacy())
	err = os.Remove(c.Service.getStatePath())
	if err != nil {
		panic(err)
	}
}

func (c *ServiceCommand) clearState() {
	c.clearPid()
	c.deleteScript("Stop")
	c.deleteScript("Launch")
	c.deleteScript("Build")
}

func (c *ServiceCommand) validateState() (bool, error) {
	if c.Pid == 0 {
		c.clearPid()
		return false, nil
	}
	exists, err := process.PidExists(int32(c.Pid))
	if err != nil {
		return false, errors.WithStack(err)
	}
	if !exists {
		c.clearPid()
		return false, nil
	}
	return true, nil
}

// InterruptGroup sends an interrupt signal to a process group.
// Will use sudo if required by this service.
func InterruptGroup(cfg OperationConfig, pgid int, service *ServiceConfig) error {
	return errors.WithStack(signalGroup(cfg, pgid, service, "-2"))
}

// KillGroup sends a kill signal to a process group.
// Will use sudo priviledges if required by this service.
func KillGroup(cfg OperationConfig, pgid int, service *ServiceConfig) error {
	return errors.WithStack(signalGroup(cfg, pgid, service, "-9"))
}

func signalGroup(cfg OperationConfig, pgid int, service *ServiceConfig, flag string) error {
	cmdName := "kill"
	cmdArgs := []string{}
	if service.IsSudo(cfg) {
		cmdName = "sudo"
		cmdArgs = append(cmdArgs, "kill")
	}
	cmdArgs = append(cmdArgs, flag, fmt.Sprintf("-%v", pgid))
	cmd := exec.Command(cmdName, cmdArgs...)
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	err := cmd.Run()
	return errors.WithMessage(err, "signalGroup:")
}

type logLine struct {
	Stream  string
	Message string
}

func logToStringSlice(path string) ([]string, error) {
	logFile, err := os.Open(path)
	defer logFile.Close()

	if err != nil {
		return nil, errors.WithStack(err)
	}
	scanner := bufio.NewScanner(logFile)
	var lines []string
	for scanner.Scan() {
		text := scanner.Text()
		var lineData logLine
		err = json.Unmarshal([]byte(text), &lineData)
		if err != nil {
			return nil, errors.WithStack(err)
		}
		if lineData.Stream != "messages" {
			lines = append(lines, lineData.Message)
		}
	}

	// check for errors
	if err = scanner.Err(); err != nil {
		return nil, errors.WithStack(err)
	}
	return lines, nil
}

// buildAbsPath will ensure the targetPath is absolute, joining to workingDir
// if necessary.
func buildAbsPath(workingDir string, targetPath *string) string {
	if targetPath != nil {
		expandedPath := os.ExpandEnv(*targetPath)
		if !path.IsAbs(expandedPath) {
			return path.Join(workingDir, expandedPath)
		}
		*targetPath = expandedPath
		return *targetPath
	}
	return workingDir
}
