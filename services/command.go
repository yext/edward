package services

import (
	"errors"
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
)

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
	Logger common.Logger
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
		OutputFile: sc.Logs.Build,
		Logger:     sc.Logger,
	}
	tracker.Start()

	if !force && sc.Pid != 0 {
		tracker.SoftFail(errgo.New("Already running"))
		return nil
	}

	if sc.Scripts.Build == "" {
		tracker.SoftFail(errgo.New("No build"))
		return nil
	}

	file, err := sc.createScript(sc.Scripts.Build, "Build")
	// Build the project and wait for completion
	if err != nil {
		return err
	}
	defer sc.deleteScript("Build")

	cmd := exec.Command(file.Name())
	err = cmd.Run()
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
	t, err = tail.TailFile(sc.Logs.Run, tail.Config{Follow: true, Logger: tail.DiscardingLogger})
	if err != nil {
		return errgo.Mask(err)
	}
	for line := range t.Lines {

		select {
		case <-cancel:
			return nil
		default:
		}

		if strings.Contains(line.Text, sc.Service.Properties.Started) {
			return nil
		}
	}
	return nil
}

func (sc *ServiceCommand) checkForAnyPort(cancel <-chan struct{}, command *exec.Cmd) error {
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
		children, err := proc.Children()
		if err != nil {
			return errgo.Mask(err)
		}

		for _, connection := range connections {
			for _, child := range children {
				if connection.Status == "LISTEN" && connection.Pid == int32(child.Pid) {
					return nil
				}
			}
		}
	}
	return errors.New("exited check loop unexpectedly")
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
	if len(sc.Service.Properties.Started) > 0 {
		startCheck = func(cancel <-chan struct{}) error {
			return sc.waitForLogText(sc.Service.Properties.Started, cancel)
		}
	} else {
		startCheck = func(cancel <-chan struct{}) error {
			return sc.checkForAnyPort(cancel, command)
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
		OutputFile: sc.Logs.Run,
		Logger:     sc.Logger,
	}
	tracker.Start()

	if sc.Pid != 0 {
		tracker.SoftFail(errgo.New("Already running"))
		return nil
	}

	if sc.Scripts.Launch == "" {
		tracker.SoftFail(errgo.New("No launch"))
		return nil
	}

	// Clear logs
	os.Remove(sc.Logs.Run)

	// Start the project and get the PID
	file, err := sc.createScript(sc.Scripts.Launch, "Launch")
	if err != nil {
		return err
	}
	//defer os.Remove(file.Name())

	cmd := exec.Command(file.Name())
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
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
	} else {
		tracker.Fail(errgo.New("Timed Out"))
		err := sc.Service.Stop()
		if err != nil {
			return errgo.Mask(err)
		}
	}
	return errgo.Mask(err)
}

func (sc *ServiceCommand) StopScript() error {

	sc.printf("Running stop script for %v\n", sc.Service.Name)

	// Start the project and get the PID
	file, err := sc.createScript(sc.Scripts.Stop, "Stop")
	if err != nil {
		return err
	}
	defer os.Remove(file.Name())

	cmd := exec.Command(file.Name())
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	err = cmd.Run()
	return err
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
