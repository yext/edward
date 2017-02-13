package services

import (
	"os"
	"os/exec"
	"path"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/hpcloud/tail"
	"github.com/pkg/errors"
	"github.com/shirou/gopsutil/net"
	"github.com/shirou/gopsutil/process"
	"github.com/yext/edward/commandline"
	"github.com/yext/edward/common"
	"github.com/yext/edward/home"
	"github.com/yext/edward/warmup"
)

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

func (sc *ServiceCommand) createScript(content string, scriptType string) (*os.File, error) {
	file, err := os.Create(path.Join(home.EdwardConfig.ScriptDir, sc.Service.Name+"-"+scriptType))
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

func (sc *ServiceCommand) deleteScript(scriptType string) error {
	return errors.WithStack(
		os.Remove(
			path.Join(home.EdwardConfig.ScriptDir, sc.Service.Name+"-"+scriptType),
		),
	)
}

// BuildSync will buid the service synchronously.
// If force is false, the build will be skipped if the service is already running.
func (sc *ServiceCommand) BuildSync(force bool) error {
	tracker := &CommandTracker{
		Name:   "Building " + sc.Service.Name,
		Logger: sc.Logger,
	}
	return errors.WithStack(sc.BuildWithTracker(force, tracker))
}

// If force is false, the build will be skipped if the service is already running.
func (sc *ServiceCommand) BuildWithTracker(force bool, tracker OperationTracker) error {
	tracker.Start()

	if !force && sc.Pid != 0 {
		tracker.SoftFail(errors.New("Already running"))
		return nil
	}

	if sc.Service.Commands.Build == "" {
		tracker.SoftFail(errors.New("No build"))
		return nil
	}

	cmd, err := sc.constructCommand(sc.Service.Commands.Build)
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

func (sc *ServiceCommand) constructCommand(command string) (*exec.Cmd, error) {
	command, cmdArgs, err := commandline.ParseCommand(os.ExpandEnv(command))
	if err != nil {
		return nil, errors.WithStack(err)
	}
	cmd := exec.Command(command, cmdArgs...)
	if sc.Service.Path != nil {
		cmd.Dir = os.ExpandEnv(*sc.Service.Path)
	}
	return cmd, nil
}

func (sc *ServiceCommand) waitForLogText(line string, cancel <-chan struct{}) error {
	// Read output until we get the success
	var t *tail.Tail
	var err error
	t, err = tail.TailFile(sc.Service.GetRunLog(), tail.Config{Follow: true, Logger: tail.DiscardingLogger})
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

func (sc *ServiceCommand) areAnyListeningPortsOpen(ports []int) (bool, error) {

	var matchedPorts = make(map[int]struct{})
	for _, port := range ports {
		matchedPorts[port] = struct{}{}
	}

	connections, err := net.Connections("all")
	if err != nil {
		return false, errors.WithStack(err)
	}
	for _, connection := range connections {
		if connection.Status == "LISTEN" {
			if _, ok := matchedPorts[int(connection.Laddr.Port)]; ok {
				return true, nil
			}
		}
	}
	return false, nil
}

func (sc *ServiceCommand) waitForListeningPorts(ports []int, cancel <-chan struct{}, command *exec.Cmd) error {
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
			if connection.Status == "LISTEN" {
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

func (sc *ServiceCommand) waitForAnyPort(cancel <-chan struct{}, command *exec.Cmd) error {
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
		if connection.Status == "LISTEN" && connection.Pid == int32(proc.Pid) {
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

func (sc *ServiceCommand) waitUntilLive(command *exec.Cmd) error {

	sc.printf("Waiting for %v to start.\n", sc.Service.Name)

	var startCheck func(cancel <-chan struct{}) error
	if sc.Service.LaunchChecks != nil && len(sc.Service.LaunchChecks.LogText) > 0 {
		startCheck = func(cancel <-chan struct{}) error {
			return errors.WithStack(
				sc.waitForLogText(sc.Service.LaunchChecks.LogText, cancel),
			)
		}
	} else if sc.Service.LaunchChecks != nil && len(sc.Service.LaunchChecks.Ports) > 0 {
		startCheck = func(cancel <-chan struct{}) error {
			return errors.WithStack(
				sc.waitForListeningPorts(sc.Service.LaunchChecks.Ports, cancel, command),
			)
		}
	} else {
		startCheck = func(cancel <-chan struct{}) error {
			return errors.WithStack(
				sc.waitForAnyPort(cancel, command),
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

func (sc *ServiceCommand) StartAsync(cfg OperationConfig) error {
	tracker := CommandTracker{
		Name:       "Launching " + sc.Service.Name,
		OutputFile: sc.Service.GetRunLog(),
		Logger:     sc.Logger,
	}
	tracker.Start()

	if sc.Pid != 0 {
		tracker.SoftFail(errors.New("Already running"))
		return nil
	}

	if sc.Service.Commands.Launch == "" {
		tracker.SoftFail(errors.New("No launch"))
		return nil
	}

	if sc.Service.LaunchChecks != nil && len(sc.Service.LaunchChecks.Ports) > 0 {
		inUse, err := sc.areAnyListeningPortsOpen(sc.Service.LaunchChecks.Ports)
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

	os.Remove(sc.Service.GetRunLog())

	cmd, err := sc.GetLaunchCommand(cfg)
	if err != nil {
		tracker.Fail(err)
		return errors.WithStack(err)
	}
	cmd.Env = os.Environ()
	cmd.Env = append(cmd.Env, sc.Service.Env...)
	err = cmd.Start()
	if err != nil {
		tracker.Fail(err)
		return errors.WithStack(err)
	}

	pid := cmd.Process.Pid

	sc.printf("%v has PID: %d.\n", sc.Service.Name, pid)

	pidStr := strconv.Itoa(pid)
	f, err := os.Create(sc.getPidPath())
	if err != nil {
		return errors.WithStack(err)
	}
	f.WriteString(pidStr)
	f.Close()

	err = sc.waitUntilLive(cmd)
	if err == nil {
		tracker.Success()
		warmup.Run(sc.Service.Name, sc.Service.Warmup)
		return nil
	}

	tracker.Fail(err)
	stopErr := sc.Service.Stop(cfg)
	if stopErr != nil {
		return errors.WithStack(stopErr)
	}
	return errors.WithStack(err)
}

func (sc *ServiceCommand) GetLaunchCommand(cfg OperationConfig) (*exec.Cmd, error) {
	command := os.Args[0]
	cmdArgs := []string{
		"run",
	}
	if cfg.NoWatch {
		cmdArgs = append(cmdArgs, "--no-watch")
	}
	cmdArgs = append(cmdArgs, sc.Service.Name)

	cmd := exec.Command(command, cmdArgs...)
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	return cmd, nil
}

func (sc *ServiceCommand) RunStopScript() ([]byte, error) {
	sc.printf("Running stop script for %v\n", sc.Service.Name)
	cmd, err := sc.constructCommand(sc.Service.Commands.Stop)
	if err != nil {
		return nil, errors.WithStack(err)
	}

	out, err := cmd.CombinedOutput()
	if err != nil {
		return out, errors.WithStack(err)
	}
	return nil, nil
}

func (sc *ServiceCommand) clearPid() {
	sc.Pid = 0
	os.Remove(sc.getPidPath())
}

func (sc *ServiceCommand) clearState() {
	sc.clearPid()
	sc.deleteScript("Stop")
	sc.deleteScript("Launch")
	sc.deleteScript("Build")
}

func (sc *ServiceCommand) getPidPath() string {
	dir := home.EdwardConfig.PidDir
	return path.Join(dir, sc.Service.Name+".pid")
}

func (sc *ServiceCommand) killGroup(cfg OperationConfig, pgid int, graceful bool) error {
	killScript := "#!/bin/bash\n"
	if sc.Service.IsSudo(cfg) {
		killScript += "sudo "
	}
	if graceful {
		killScript += "kill -2 -"
	} else {
		killScript += "kill -9 -"
	}
	killScript += strconv.Itoa(pgid)
	killSignalScript, err := sc.createScript(killScript, "Kill")
	if err != nil {
		return errors.WithStack(err)
	}
	defer os.Remove(killSignalScript.Name())

	cmd := exec.Command(killSignalScript.Name())
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	err = cmd.Run()
	return errors.WithStack(err)
}
