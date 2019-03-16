package instance

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"path"
	"strings"
	"syscall"
	"time"

	"github.com/pkg/errors"
	uuid "github.com/satori/go.uuid"
	"github.com/yext/edward/commandline"
	"github.com/yext/edward/common"
	"github.com/yext/edward/home"
	"github.com/yext/edward/services"
	"github.com/yext/edward/tracker"
)

// Instance provides state and functions for managing a service
type Instance struct {
	// Parent service config
	Service *services.ServiceConfig `json:"service"`
	// Pid of currently running instance
	Pid int `json:"pid"`
	// Config file from which this instance was launched
	ConfigFile string `json:"configFile"`
	// The edward version under which this instance was launched
	EdwardVersion string `json:"edwardVersion"`
	// Overrides applied by the group under which this service was started
	Overrides services.ContextOverride `json:"overrides,omitempty"`
	// Identifier for this instance of the service
	InstanceId string

	dirConfig *home.EdwardConfiguration

	processes Processes
}

// Load loads an instance to control the specified service
func Load(
	dirConfig *home.EdwardConfiguration,
	processes Processes,
	service *services.ServiceConfig,
	overrides services.ContextOverride,
) (command *Instance, err error) {
	command = &Instance{
		Service:    service,
		ConfigFile: service.ConfigFile,
		InstanceId: uuid.NewV4().String(),
	}
	defer func() {
		command.Service = service
		command.EdwardVersion = common.EdwardVersion
		command.Overrides = command.Overrides.Merge(overrides)
		command.dirConfig = dirConfig
		command.processes = processes
		pidExists, err := processes.PidCommandMatches(command.Pid, command.Service.Name)
		if err != nil {
			err = errors.WithStack(err)
		}
		if !pidExists {
			command.Pid = 0
		}
	}()

	legacyPidFile := service.GetPidPathLegacy(dirConfig.PidDir)
	if _, err := os.Stat(legacyPidFile); err == nil {
		command.Pid, err = service.GetPid(legacyPidFile)
		if err != nil {
			return nil, errors.WithStack(err)
		}
		return command, nil
	}

	stateFile := service.GetStatePath(dirConfig.StateDir)
	if _, err := os.Stat(stateFile); err == nil {
		raw, err := ioutil.ReadFile(stateFile)
		if err != nil {
			return nil, errors.WithStack(err)
		}
		err = json.Unmarshal(raw, command)
		if err != nil {
			return command, errors.WithStack(err)
		}
	}

	return command, nil
}

// Env provides the combined environment variables for this service command
func (c *Instance) Env() []string {
	return append(c.Service.Env, c.Overrides.Env...)
}

// Getenv returns the environment variable value for the provided key, if present.
// Env overrides are consulted first, followed by service env settings, then the os Env.
func (c *Instance) Getenv(key string) string {
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

// save will store the current state of this command to a state file
func (c *Instance) save() error {
	commandJSON, _ := json.Marshal(c)
	err := ioutil.WriteFile(c.Service.GetStatePath(c.dirConfig.StateDir), commandJSON, 0644)
	if err != nil {
		return errors.WithStack(err)
	}
	return nil
}

func (c *Instance) createScript(content string, scriptType string) (*os.File, error) {
	file, err := os.Create(path.Join(c.dirConfig.ScriptDir, c.Service.Name+"-"+scriptType))
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

func (c *Instance) getLaunchCommand(cfg services.OperationConfig) (*exec.Cmd, error) {
	command := cfg.EdwardExecutable
	var cmdArgs []string
	cmdArgs = append(cmdArgs, "run", c.Service.Name)
	cmdArgs = append(cmdArgs, "-c", c.ConfigFile)
	if cfg.NoWatch {
		cmdArgs = append(cmdArgs, "--no-watch")
	}
	for _, tag := range cfg.Tags {
		cmdArgs = append(cmdArgs, "--tag", tag)
	}
	if cfg.LogFile != "" {
		cmdArgs = append(cmdArgs, "--logfile", cfg.LogFile)
	}
	for service, backend := range cfg.Backends {
		cmdArgs = append(cmdArgs, "-b", fmt.Sprintf("%s:%s", service, backend))
	}

	cmdArgs = append(cmdArgs, "--edward_home", c.dirConfig.Dir)

	log.Printf("Launching runner with args: %v", cmdArgs)
	cmd := exec.Command(command, cmdArgs...)
	cmd.Dir = commandline.BuildAbsPath(cfg.WorkingDir, c.Service.Path)
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	return cmd, nil
}

func (c *Instance) clearPid() {
	c.Pid = 0
	_ = os.Remove(c.Service.GetPidPathLegacy(c.dirConfig.PidDir))
	_ = os.Remove(c.Service.GetStatePath(c.dirConfig.StateDir))
}

func (c *Instance) clearState() {
	c.clearPid()
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

// StopSync stops this service in a synchronous manner
func (c *Instance) StopSync(cfg services.OperationConfig, overrides services.ContextOverride, task tracker.Task) error {
	if cfg.IsExcluded(c.Service) {
		return nil
	}

	if !c.Service.Backend().HasLaunchStep() {
		return nil
	}

	// Clean up when this function returns
	defer c.clearState()

	job := task.Child(c.Service.GetName()).Child("Stop")
	job.SetState(tracker.TaskStateInProgress)
	if c.Pid == 0 {
		job.SetState(tracker.TaskStateWarning, "Not running")
		return nil
	}

	log.Printf("Interrupting process for %v\n", c.Service.Name)
	stopped, err := c.interruptProcess(cfg)
	if err != nil {
		job.SetState(tracker.TaskStateFailed, err.Error())
		return nil
	}
	log.Printf("Interrupt succeeded, was process stopped? %v\n", stopped)

	if !stopped {
		timeout := c.Service.GetTerminationTimeout()
		log.Printf("SIGINT failed to stop service, waiting for %v before sending SIGKILL\n", timeout)
		stopped, err := c.waitForTerm(timeout)
		if err != nil {
			job.SetState(tracker.TaskStateFailed, "Waiting for termination failed", err.Error())
			return nil
		}
		if !stopped {
			stopped, err := c.killProcess(cfg)
			if err != nil {
				job.SetState(tracker.TaskStateFailed, "Kill failed", err.Error())
				return nil
			}
			if stopped {
				job.SetState(tracker.TaskStateWarning, fmt.Sprintf("Killed after %v", timeout))
				return nil
			}
			job.SetState(tracker.TaskStateFailed, "Process was not killed")
			return nil
		}
	}

	log.Printf("Cleaning up state after shutdown")
	job.SetState(tracker.TaskStateSuccess)
	return nil
}

func (c *Instance) killProcess(cfg services.OperationConfig) (success bool, err error) {
	err = c.processes.KillGroup(c.Pid, c.Service.IsSudo(cfg))
	if err != nil {
		return false, errors.WithStack(err)
	}
	return true, nil
}

func (c *Instance) interruptProcess(cfg services.OperationConfig) (success bool, err error) {
	err = c.processes.SendSignal(c.Pid, syscall.SIGINT)
	if err != nil {
		return false, errors.WithStack(err)
	}

	// Check to see if the process is still running
	exists, err := c.processes.PidExists(c.Pid)
	if err != nil {
		return false, errors.WithStack(err)
	}
	return !exists, nil
}

func (c *Instance) waitForTerm(timeout time.Duration) (bool, error) {
	for elapsed := time.Duration(0); elapsed <= timeout; elapsed += time.Millisecond * 100 {
		exists, err := c.processes.PidExists(c.Pid)
		if err != nil {
			return false, errors.WithStack(err)
		}
		if !exists {
			log.Printf("Process no longer exists")
			return true, nil
		}
		time.Sleep(time.Millisecond * 100)
		log.Printf("Slept for %v", elapsed)
	}
	return false, nil
}
