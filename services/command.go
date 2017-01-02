package services

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/fatih/color"
	"github.com/hpcloud/tail"
	"github.com/juju/errgo"
	"github.com/shirou/gopsutil/net"
	"github.com/shirou/gopsutil/process"
	"github.com/yext/edward/common"
	"github.com/yext/edward/home"
	"github.com/yext/edward/warmup"
)

type ServiceCommand struct {
	// Parent service config
	Service *ServiceConfig
	// Path to string
	Scripts struct {
		Build  Script
		Launch Script
		Stop   Script
	}
	Pid    int
	Logger common.Logger
}

type Script struct {
	Path    string
	Command string
	Log     string
}

func (s *Script) WillRun() bool {
	return s.Command != ""
}

func (s *Script) GetCommand() (*exec.Cmd, error) {
	command, following, err := parseCommand(s.Command)
	if err != nil {
		return nil, err
	}
	args := []string{
		"run",
		s.Path,
		s.Log,
		command,
	}
	args = append(args, following...)

	cmd := exec.Command(os.Args[0], args...)
	cmd.Stderr = os.Stderr
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	return cmd, nil
}

func (s *Script) Run() error {
	cmd, err := s.GetCommand()
	if err != nil {
		return err
	}
	return cmd.Run()
}

// Returns the executable path and arguments
// TODO: Clean this up
func parseCommand(cmd string) (string, []string, error) {
	var args []string
	state := "start"
	current := ""
	quote := "\""
	for i := 0; i < len(cmd); i++ {
		c := cmd[i]

		if state == "quotes" {
			if string(c) != quote {
				current += string(c)
			} else {
				args = append(args, current)
				current = ""
				state = "start"
			}
			continue
		}

		if c == '"' || c == '\'' {
			state = "quotes"
			quote = string(c)
			continue
		}

		if state == "arg" {
			if c == ' ' || c == '\t' {
				args = append(args, current)
				current = ""
				state = "start"
			} else {
				current += string(c)
			}
			continue
		}

		if c != ' ' && c != '\t' {
			state = "arg"
			current += string(c)
		}
	}

	if state == "quotes" {
		return "", []string{}, errors.New(fmt.Sprintf("Unclosed quote in command line: %s", cmd))
	}

	if current != "" {
		args = append(args, current)
	}

	if len(args) <= 0 {
		return "", []string{}, errors.New("Empty command line")
	}

	if len(args) == 1 {
		return args[0], []string{}, nil
	}

	return args[0], args[1:], nil
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
	return os.Remove(path.Join(home.EdwardConfig.ScriptDir, sc.Service.Name+"-"+scriptType))
}

// BuildSync will buid the service synchronously.
// If force is false, the build will be skipped if the service is already running.
func (sc *ServiceCommand) BuildSync(force bool) error {
	tracker := CommandTracker{
		Name:       "Building " + sc.Service.Name,
		OutputFile: sc.Scripts.Build.Log,
		Logger:     sc.Logger,
	}
	tracker.Start()

	if !force && sc.Pid != 0 {
		tracker.SoftFail(errgo.New("Already running"))
		return nil
	}

	if !sc.Scripts.Build.WillRun() {
		tracker.SoftFail(errgo.New("No build"))
		return nil
	}

	err := sc.Scripts.Build.Run()
	if err != nil {
		tracker.Fail(err)
		return errgo.Mask(err)
	}

	tracker.Success()
	return nil
}

func (sc *ServiceCommand) waitForLogText(line string, cancel <-chan struct{}) error {
	// Read output until we get the success
	var t *tail.Tail
	var err error
	t, err = tail.TailFile(sc.Scripts.Launch.Log, tail.Config{Follow: true, Logger: tail.DiscardingLogger})
	if err != nil {
		return errgo.Mask(err)
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
		return false, errgo.Mask(err)
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
			return errgo.Mask(err)
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
			return errgo.Mask(err)
		}

		proc, err := process.NewProcess(int32(command.Process.Pid))
		if err != nil {
			return errgo.Mask(err)
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
			return sc.waitForLogText(sc.Service.LaunchChecks.LogText, cancel)
		}
	} else if sc.Service.LaunchChecks != nil && len(sc.Service.LaunchChecks.Ports) > 0 {
		startCheck = func(cancel <-chan struct{}) error {
			return sc.waitForListeningPorts(sc.Service.LaunchChecks.Ports, cancel, command)
		}
	} else {
		startCheck = func(cancel <-chan struct{}) error {
			return sc.waitForAnyPort(cancel, command)
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
		return result.error
	case result := <-cancelableWait(done, processFinished):
		return result.error
	case <-timeout.C:
		return errors.New("Waiting for service timed out")
	}

}

func (sc *ServiceCommand) StartAsync() error {
	tracker := CommandTracker{
		Name:       "Launching " + sc.Service.Name,
		OutputFile: sc.Scripts.Launch.Log,
		Logger:     sc.Logger,
	}
	tracker.Start()

	if sc.Pid != 0 {
		tracker.SoftFail(errgo.New("Already running"))
		return nil
	}

	if !sc.Scripts.Launch.WillRun() {
		tracker.SoftFail(errgo.New("No launch"))
		return nil
	}

	if sc.Service.LaunchChecks != nil && len(sc.Service.LaunchChecks.Ports) > 0 {
		inUse, err := sc.areAnyListeningPortsOpen(sc.Service.LaunchChecks.Ports)
		if err != nil {
			return errgo.Mask(err)
		}
		if inUse {
			return errgo.New("one or more of the ports required by this service are in use")
		}
	}

	// Clear logs
	os.Remove(sc.Scripts.Launch.Log)

	cmd, err := sc.Scripts.Launch.GetCommand()
	if err != nil {
		printResult("Failed", color.FgRed)
		return errgo.Mask(err)
	}
	cmd.Env = os.Environ()
	cmd.Env = append(cmd.Env, sc.Service.Env...)
	err = cmd.Start()
	if err != nil {
		printResult("Failed", color.FgRed)
		return errgo.Mask(err)
	}

	pid := cmd.Process.Pid

	sc.printf("%v has PID: %d.\n", sc.Service.Name, pid)

	pidStr := strconv.Itoa(pid)
	f, err := os.Create(sc.getPidPath())
	if err != nil {
		return err
	}
	f.WriteString(pidStr)
	f.Close()

	err = sc.waitUntilLive(cmd)
	if err == nil {
		tracker.Success()
		warmup.Run(sc.Service.Name, sc.Service.Warmup)
		return nil
	}

	tracker.Fail(errgo.New("Timed Out"))
	err = sc.Service.Stop()
	return errgo.Mask(err)
}

func (sc *ServiceCommand) StopScript() error {
	sc.printf("Running stop script for %v\n", sc.Service.Name)
	return sc.Scripts.Stop.Run()
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

func (sc *ServiceCommand) killGroup(pgid int, graceful bool) error {
	killScript := "#!/bin/bash\n"
	if sc.Service.IsSudo() {
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
		return errgo.Mask(err)
	}
	defer os.Remove(killSignalScript.Name())

	cmd := exec.Command(killSignalScript.Name())
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	err = cmd.Run()
	return errgo.Mask(err)
}
