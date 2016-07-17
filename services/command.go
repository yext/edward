package services

import (
	"errors"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"path"
	"strconv"
	"strings"
	"sync"
	"syscall"

	"github.com/fatih/color"
	"github.com/hpcloud/tail"
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

	if sc.Pid != 0 {
		printResult("Already running", color.FgYellow)
		return nil
	}

	if sc.Scripts.Build == "" {
		printResult("No build", color.FgGreen)
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
		return err
	}

	printResult("OK", color.FgGreen)

	return nil
}

func (sc *ServiceCommand) waitUntilLive(command *exec.Cmd) error {

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

	if sc.Pid != 0 {
		printResult("Already running", color.FgYellow)
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
		return err
	}

	pid := cmd.Process.Pid

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
	} else {
		printResult("Failed!", color.FgRed)
		printFile(sc.Logs.Run)
	}
	return err
}

func (sc *ServiceCommand) StopScript() error {

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

func (s *ServiceConfig) GetCommand() *ServiceCommand {

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

	buildScript := s.makeScript(s.Commands.Build, logs.Build)
	startScript := s.makeScript(s.Commands.Launch, logs.Run)
	stopScript := s.makeScript(s.Commands.Stop, logs.Stop)
	command := &ServiceCommand{
		Service: s,
		Scripts: struct {
			Build  string
			Launch string
			Stop   string
		}{
			Build:  buildScript,
			Launch: startScript,
			Stop:   stopScript,
		},
		Logs: logs,
	}

	// Retrieve the PID if available
	pidFile := command.getPidPath()
	if _, err := os.Stat(pidFile); err == nil {
		dat, err := ioutil.ReadFile(pidFile)
		if err != nil {
			log.Fatal(err)
		}
		pid, err := strconv.Atoi(string(dat))
		if err != nil {
			log.Fatal(err)
		}
		command.Pid = pid

		// TODO: Check this PID is actually live
		process, err := os.FindProcess(int(pid))
		if err != nil {
			command.clearState()
		} else {
			err := process.Signal(syscall.Signal(0))
			if err != nil {
				command.clearState()
			}
		}
	}
	// TODO: Set status

	return command
}
