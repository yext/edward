package services

import (
	"errors"
	"os"
	"os/exec"
	"path"
	"strconv"
	"strings"
	"sync"
	"syscall"

	"github.com/fatih/color"
	"github.com/hpcloud/tail"
	"github.com/yext/edward/common"
	"github.com/yext/edward/home"
	"github.com/yext/errgo"
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

func (sc *ServiceCommand) BuildSync() error {
	printOperation("Building " + sc.Service.Name)
	sc.printf("Building %v\n", sc.Service.Name)

	if sc.Pid != 0 {
		sc.printf("%v is already running\n", sc.Service.Name)
		printResult("Already running", color.FgYellow)
		return nil
	}

	if sc.Scripts.Build == "" {
		printResult("No build", color.FgGreen)
		sc.printf("No build needed for %v\n", sc.Service.Name)
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
		printResult("Failed", color.FgRed)
		printFile(sc.Logs.Build)
		return errgo.Mask(err)
	}

	printResult("OK", color.FgGreen)
	sc.printf("%v build succeeded.\n", sc.Service.Name)

	return nil
}

func (sc *ServiceCommand) waitUntilLive(command *exec.Cmd) error {

	sc.printf("Waiting for %v to start.\n", sc.Service.Name)

	var err error = nil
	var wg sync.WaitGroup
	wg.Add(1)

	go func() {
		// Read output until we get the success
		var t *tail.Tail
		t, err = tail.TailFile(sc.Logs.Run, tail.Config{Follow: true, Logger: tail.DiscardingLogger})
		for line := range t.Lines {
			if strings.Contains(line.Text, sc.Service.Properties.Started) {
				wg.Done()
				return
			}
		}
	}()

	go func() {
		// Wait until the process exists
		command.Wait()
		err = errors.New("Command failed!")
		wg.Done()
	}()

	wg.Wait()

	return err
}

func (sc *ServiceCommand) StartAsync() error {

	printOperation("Launching " + sc.Service.Name)
	sc.printf("Launching %v\n", sc.Service.Name)

	if sc.Pid != 0 {
		printResult("Already running", color.FgYellow)
		sc.printf("%v is already running.\n", sc.Service.Name)
		return nil
	}

	if sc.Scripts.Launch == "" {
		printResult("No launch", color.FgGreen)
		sc.printf("No launch needed for %v\n", sc.Service.Name)
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
		printResult("OK", color.FgGreen)
		sc.printf("%v start succeeded.\n", sc.Service.Name)
	} else {
		printResult("Failed!", color.FgRed)
		printFile(sc.Logs.Run)
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

func (sc *ServiceCommand) killGroup(pgid int) error {
	killScript := "#!/bin/bash\n"
	if sc.Service.IsSudo() {
		killScript = killScript + "sudo "
	}
	killScript = killScript + "kill -2 -" + strconv.Itoa(pgid)
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
