package services

import (
	"fmt"
	"os"
	"os/exec"
	"path"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/hpcloud/tail"
	"github.com/pkg/errors"
	"github.com/theothertomelliott/gopsutil-nocgo/net"
	"github.com/theothertomelliott/gopsutil-nocgo/process"
	"github.com/yext/edward/commandline"
	"github.com/yext/edward/common"
	"github.com/yext/edward/home"
	"github.com/yext/edward/warmup"
)

// ServiceCommand provides state and functions for managing a service
type ServiceCommand struct {
	// Parent service config
	Service *ServiceConfig
	Pid     int
	Logger  common.Logger
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
func (c *ServiceCommand) BuildSync(force bool) error {
	tracker := &CommandTracker{
		Name:   "Building " + c.Service.Name,
		Logger: c.Logger,
	}
	return errors.WithStack(c.BuildWithTracker(force, tracker))
}

// BuildWithTracker builds a service.
// If force is false, the build will be skipped if the service is already running.
func (c *ServiceCommand) BuildWithTracker(force bool, tracker OperationTracker) error {
	tracker.Start()

	if !force && c.Pid != 0 {
		tracker.SoftFail(errors.New("Already running"))
		return nil
	}

	if c.Service.Commands.Build == "" {
		tracker.SoftFail(errors.New("No build"))
		return nil
	}

	cmd, err := c.constructCommand(c.Service.Commands.Build)
	if err != nil {
		tracker.Fail(err)
		return errors.WithStack(err)
	}

	out, err := cmd.CombinedOutput()
	if err != nil {
		tracker.FailWithOutput(err, string(out))
		return errors.WithStack(err)
	}

	tracker.Success()
	return nil
}

func (c *ServiceCommand) constructCommand(command string) (*exec.Cmd, error) {
	command, cmdArgs, err := commandline.ParseCommand(os.ExpandEnv(command))
	if err != nil {
		return nil, errors.WithStack(err)
	}
	cmd := exec.Command(command, cmdArgs...)
	if c.Service.Path != nil {
		cmd.Dir = os.ExpandEnv(*c.Service.Path)
	}
	return cmd, nil
}

func (c *ServiceCommand) waitForLogText(line string, cancel <-chan struct{}) error {
	// Read output until we get the success
	var t *tail.Tail
	var err error
	t, err = tail.TailFile(c.Service.GetRunLog(), tail.Config{Follow: true, Logger: tail.DiscardingLogger})
	if err != nil {
		return errors.WithStack(err)
	}
	for logLine := range t.Lines {

		select {
		case <-cancel:
			return nil
		default:
		}

		if strings.Contains(logLine.Text, line) {
			return nil
		}
	}
	return nil
}

const portStatusListen = "LISTEN"

func (c *ServiceCommand) areAnyListeningPortsOpen(ports []int) (bool, error) {

	var matchedPorts = make(map[int]struct{})
	for _, port := range ports {
		matchedPorts[port] = struct{}{}
	}

	connections, err := net.Connections("all")
	if err != nil {
		return false, errors.WithStack(err)
	}
	for _, connection := range connections {
		if connection.Status == portStatusListen {
			if _, ok := matchedPorts[int(connection.Laddr.Port)]; ok {
				return true, nil
			}
		}
	}
	return false, nil
}

func (c *ServiceCommand) waitForListeningPorts(ports []int, cancel <-chan struct{}, command *exec.Cmd) error {
	for true {
		time.Sleep(100 * time.Millisecond)

		select {
		case <-cancel:
			return nil
		default:
		}

		var matchedPorts = make(map[int]struct{})

		connections, err := net.Connections("all")
		if err != nil {
			return errors.WithStack(err)
		}
		for _, connection := range connections {
			if connection.Status == portStatusListen {
				matchedPorts[int(connection.Laddr.Port)] = struct{}{}
			}
		}
		allMatched := true
		for _, port := range ports {
			if _, ok := matchedPorts[port]; !ok {
				allMatched = false
			}
		}
		if allMatched {
			return nil
		}
	}
	return errors.New("exited check loop unexpectedly")
}

func (c *ServiceCommand) waitForAnyPort(cancel <-chan struct{}, command *exec.Cmd) error {
	for true {
		time.Sleep(100 * time.Millisecond)

		select {
		case <-cancel:
			return nil
		default:
		}

		connections, err := net.Connections("all")
		if err != nil {
			return errors.WithStack(err)
		}

		proc, err := process.NewProcess(int32(command.Process.Pid))
		if err != nil {
			return errors.WithStack(err)
		}
		if hasPort(proc, connections) {
			return nil
		}
	}
	return errors.New("exited check loop unexpectedly")
}

func hasPort(proc *process.Process, connections []net.ConnectionStat) bool {
	for _, connection := range connections {
		if connection.Status == portStatusListen && connection.Pid == int32(proc.Pid) {
			return true
		}
	}

	children, err := proc.Children()
	if err == nil {
		for _, child := range children {
			if hasPort(child, connections) {
				return true
			}
		}
	}
	return false
}

func cancelableWait(cancel chan struct{}, task func(cancel <-chan struct{}) error) <-chan struct{ error } {
	finished := make(chan struct{ error })
	go func() {
		defer close(finished)
		err := task(cancel)
		finished <- struct{ error }{err}
	}()
	return finished
}

func (c *ServiceCommand) waitUntilLive(command *exec.Cmd) error {

	c.printf("Waiting for %v to start.\n", c.Service.Name)

	var startCheck func(cancel <-chan struct{}) error
	if c.Service.LaunchChecks != nil && len(c.Service.LaunchChecks.LogText) > 0 {
		startCheck = func(cancel <-chan struct{}) error {
			return errors.WithStack(
				c.waitForLogText(c.Service.LaunchChecks.LogText, cancel),
			)
		}
	} else if c.Service.LaunchChecks != nil && len(c.Service.LaunchChecks.Ports) > 0 {
		startCheck = func(cancel <-chan struct{}) error {
			return errors.WithStack(
				c.waitForListeningPorts(c.Service.LaunchChecks.Ports, cancel, command),
			)
		}
	} else {
		startCheck = func(cancel <-chan struct{}) error {
			return errors.WithStack(
				c.waitForAnyPort(cancel, command),
			)
		}
	}

	processFinished := func(cancel <-chan struct{}) error {
		// Wait until the process exists
		command.Wait()
		select {
		case <-cancel:
			return nil
		default:
		}
		return errors.New("service terminated prematurely")
	}

	timeout := time.NewTimer(30 * time.Second)
	defer timeout.Stop()

	done := make(chan struct{})
	defer close(done)

	select {
	case result := <-cancelableWait(done, startCheck):
		return errors.WithStack(result.error)
	case result := <-cancelableWait(done, processFinished):
		return errors.WithStack(result.error)
	case <-timeout.C:
		return errors.New("Waiting for service timed out")
	}

}

// StartAsync starts the service in the background
// Will block until the service is known to have started successfully.
// If the service fails to launch, an error will be returned.
func (c *ServiceCommand) StartAsync(cfg OperationConfig) error {
	tracker := CommandTracker{
		Name:       "Launching " + c.Service.Name,
		OutputFile: c.Service.GetRunLog(),
		Logger:     c.Logger,
	}
	tracker.Start()

	if c.Pid != 0 {
		tracker.SoftFail(errors.New("Already running"))
		return nil
	}

	if c.Service.Commands.Launch == "" {
		tracker.SoftFail(errors.New("No launch"))
		return nil
	}

	if c.Service.LaunchChecks != nil && len(c.Service.LaunchChecks.Ports) > 0 {
		inUse, err := c.areAnyListeningPortsOpen(c.Service.LaunchChecks.Ports)
		if err != nil {
			tracker.Fail(err)
			return errors.WithStack(err)
		}
		if inUse {
			inUseErr := errors.New("one or more of the ports required by this service are in use")
			tracker.Fail(inUseErr)
			return errors.WithStack(inUseErr)
		}
	}

	os.Remove(c.Service.GetRunLog())

	cmd, err := c.getLaunchCommand(cfg)
	if err != nil {
		tracker.Fail(err)
		return errors.WithStack(err)
	}
	cmd.Env = os.Environ()
	cmd.Env = append(cmd.Env, c.Service.Env...)
	err = cmd.Start()
	if err != nil {
		tracker.Fail(err)
		return errors.WithStack(err)
	}

	pid := cmd.Process.Pid

	c.printf("%v has PID: %d.\n", c.Service.Name, pid)

	pidStr := strconv.Itoa(pid)
	f, err := os.Create(c.getPidPath())
	if err != nil {
		return errors.WithStack(err)
	}
	f.WriteString(pidStr)
	f.Close()

	err = c.waitUntilLive(cmd)
	if err == nil {
		tracker.Success()
		warmup.Run(c.Service.Name, c.Service.Warmup)
		return nil
	}

	tracker.Fail(err)
	stopErr := c.Service.Stop(cfg)
	if stopErr != nil {
		return errors.WithStack(stopErr)
	}
	return errors.WithStack(err)
}

func (c *ServiceCommand) getLaunchCommand(cfg OperationConfig) (*exec.Cmd, error) {
	command := os.Args[0]
	cmdArgs := []string{
		"run",
	}
	if cfg.NoWatch {
		cmdArgs = append(cmdArgs, "--no-watch")
	}
	cmdArgs = append(cmdArgs, c.Service.Name)

	cmd := exec.Command(command, cmdArgs...)
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	return cmd, nil
}

// RunStopScript will execute the stop script for this command, returning full output
// from running the script.
// Assumes the service has a stop script configured.
func (c *ServiceCommand) RunStopScript() ([]byte, error) {
	c.printf("Running stop script for %v\n", c.Service.Name)
	cmd, err := c.constructCommand(c.Service.Commands.Stop)
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
	os.Remove(c.getPidPath())
}

func (c *ServiceCommand) clearState() {
	c.clearPid()
	c.deleteScript("Stop")
	c.deleteScript("Launch")
	c.deleteScript("Build")
}

func (c *ServiceCommand) getPidPath() string {
	dir := home.EdwardConfig.PidDir
	return path.Join(dir, c.Service.Name+".pid")
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
	return errors.WithStack(err)
}
