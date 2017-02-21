package runner

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"strings"
	"sync"
	"time"

	"github.com/pkg/errors"
	"github.com/urfave/cli"
	"github.com/yext/edward/commandline"
	"github.com/yext/edward/config"
	"github.com/yext/edward/services"
)

var runnerInstance = &Runner{}

var Command = cli.Command{
	Name:   "run",
	Hidden: true,
	Action: runnerInstance.run,
	Flags: []cli.Flag{
		cli.BoolFlag{
			Name:        "no-watch",
			Usage:       "Disable autorestart",
			Destination: &(runnerInstance.noWatch),
		},
	},
}

type Runner struct {
	service    *services.ServiceConfig
	command    *exec.Cmd
	logFile    *os.File
	messageLog *RunnerLog

	commandWait sync.WaitGroup
	noWatch     bool
}

type Logger interface {
	Printf(format string, a ...interface{})
}

func (r *Runner) run(c *cli.Context) error {
	args := c.Args()
	if len(args) < 1 {
		return errors.New("a service name is required")
	}

	var ok bool
	r.service, ok = config.GetServiceMap()[args[0]]
	if !ok {
		return errors.New("service not found")
	}

	logLocation := r.service.GetRunLog()
	os.Remove(logLocation)

	var err error
	r.logFile, err = os.Create(logLocation)
	if err != nil {
		return errors.WithStack(err)
	}

	r.messageLog = &RunnerLog{
		file:   r.logFile,
		name:   r.service.Name,
		stream: "messages",
	}
	defer r.messageLog.Printf("Service stopped\n")

	r.commandWait.Add(1)
	err = r.startService()
	if err != nil {
		return errors.WithStack(err)
	}

	if !r.noWatch {
		closeWatchers, err := BeginWatch(r.service, r.restartService, r.messageLog)
		if err != nil {
			r.messageLog.Printf("Could not enable auto-restart: %v\n", err)
		} else {
			r.messageLog.Printf("Auto-restart enabled. This service will restart when files in its watch directories are edited.\nThis can be disabled using the --no-watch flag.\n")
			defer closeWatchers()
		}
	}

	signalChan := make(chan os.Signal, 1)
	signal.Notify(signalChan, os.Interrupt)
	go func() {
		for _ = range signalChan {
			r.messageLog.Printf("Received interrupt\n")
			err := r.stopService()
			if err != nil {
				r.messageLog.Printf("Could not stop service: %v", err)
			}
		}
	}()

	r.commandWait.Wait()
	return nil
}

func (r *Runner) restartService() error {
	r.messageLog.Printf("Restarting service\n")

	// Increment the counter to prevent exiting unexpectedly
	r.commandWait.Add(1)

	err := r.stopService()
	if err != nil {
		return errors.WithStack(err)
	}
	err = r.startService()
	if err != nil {
		return errors.WithStack(err)
	}
	return nil
}

func (r *Runner) stopService() error {
	r.messageLog.Printf("stopService")
	command, err := r.service.GetCommand()
	if err != nil {
		r.messageLog.Printf("Could not get service command: %v\n", err)
	}

	var scriptErr error
	var scriptOutput []byte
	if r.service.Commands.Stop != "" {
		r.messageLog.Printf("Running stop script for %v.\n", r.service.Name)
		scriptOutput, scriptErr = command.RunStopScript()
		if scriptErr != nil {
			r.messageLog.Printf("%v\n", string(scriptOutput))
			r.messageLog.Printf("Stop script failed: %v\n", scriptErr)
		}
		if r.waitForCompletionWithTimeout(1 * time.Second) {
			return nil
		}
		r.messageLog.Printf("Stop script did not effectively stop service, sending interrupt\n")
	}

	err = r.command.Process.Signal(os.Interrupt)
	if err != nil {
		return errors.WithStack(err)
	}

	if r.waitForCompletionWithTimeout(1 * time.Second) {
		r.messageLog.Printf("Interrupt succeeded")
		return nil
	}
	r.messageLog.Printf("Stop script did not effectively stop service, sending kill\n")

	err = r.command.Process.Kill()
	if err != nil {
		return errors.WithStack(err)
	}

	if r.waitForCompletionWithTimeout(1 * time.Second) {
		return nil
	}
	return errors.New("kill did not stop service")
}

func (r *Runner) waitForCompletionWithTimeout(timeout time.Duration) bool {
	var completed = make(chan struct{})
	defer close(completed)
	timer := time.NewTimer(timeout)
	defer timer.Stop()

	go func() {
		r.command.Wait()
		r.messageLog.Printf("Command completed")
		completed <- struct{}{}
	}()

	select {
	case <-completed:
		return true
	case <-timer.C:
		return false
	}
}

func (r *Runner) startService() error {
	r.messageLog.Printf("Service starting\n")
	command, cmdArgs, err := commandline.ParseCommand(os.ExpandEnv(r.service.Commands.Launch))
	if err != nil {
		return errors.WithStack(err)
	}

	standardLog := &RunnerLog{
		file:   r.logFile,
		name:   r.service.Name,
		stream: "stdout",
	}
	errorLog := &RunnerLog{
		file:   r.logFile,
		name:   r.service.Name,
		stream: "stderr",
	}

	cmd := exec.Command(command, cmdArgs...)
	if r.service.Path != nil {
		cmd.Dir = os.ExpandEnv(*r.service.Path)
	}
	cmd.Stdout = standardLog
	cmd.Stderr = errorLog

	r.command = cmd

	go func() {
		cmd.Run()
		r.messageLog.Printf("command run finished, sending done")
		r.commandWait.Done()
	}()
	return nil
}

// RunnerLog provides the io.Writer interface to publish service logs to file
type RunnerLog struct {
	file   *os.File
	name   string
	stream string
}

func (r *RunnerLog) Printf(format string, a ...interface{}) {
	msg := fmt.Sprintf(format, a...)
	r.Write([]byte(msg))
}

func (r *RunnerLog) Write(p []byte) (int, error) {
	fmt.Println(strings.TrimRight(string(p), "\n"))
	lineData := LogLine{
		Name:    r.name,
		Time:    time.Now(),
		Stream:  r.stream,
		Message: strings.TrimSpace(string(p)),
	}

	jsonContent, err := json.Marshal(lineData)
	if err != nil {
		return 0, errors.Wrap(err, "could not prepare log line")
	}

	line := fmt.Sprintln(string(jsonContent))
	count, err := r.file.Write([]byte(line))
	if err != nil {
		fmt.Println("Error")
		return count, errors.Wrap(err, "could not write log line")
	}
	return len(p), nil
}
