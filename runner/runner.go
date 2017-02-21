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

var Command = cli.Command{
	Name:   "run",
	Hidden: true,
	Action: run,
	Flags: []cli.Flag{
		cli.BoolFlag{
			Name:        "no-watch",
			Usage:       "Disable autorestart",
			Destination: &(noWatch),
		},
	},
}

var runningService *services.ServiceConfig
var runningCommand *exec.Cmd
var logFile *os.File
var messageLog *RunnerLog

var commandWait sync.WaitGroup

var noWatch bool

type Logger interface {
	Printf(format string, a ...interface{})
}

func run(c *cli.Context) error {
	args := c.Args()
	if len(args) < 1 {
		return errors.New("a service name is required")
	}

	var ok bool
	runningService, ok = config.GetServiceMap()[args[0]]
	if !ok {
		return errors.New("service not found")
	}

	logLocation := runningService.GetRunLog()
	os.Remove(logLocation)

	var err error
	logFile, err = os.Create(logLocation)
	if err != nil {
		return errors.WithStack(err)
	}

	messageLog = &RunnerLog{
		file:   logFile,
		name:   runningService.Name,
		stream: "messages",
	}

	defer messageLog.Printf("Service stopped\n")

	commandWait.Add(1)
	err = startService()
	if err != nil {
		return errors.WithStack(err)
	}

	if !noWatch {
		closeWatchers, err := BeginWatch(runningService, restartService, messageLog)
		if err != nil {
			messageLog.Printf("Could not enable auto-restart: %v\n", err)
		} else {
			messageLog.Printf("Auto-restart enabled. This service will restart when files in its watch directories are edited.\nThis can be disabled using the --no-watch flag.\n")
			defer closeWatchers()
		}
	}

	signalChan := make(chan os.Signal, 1)
	signal.Notify(signalChan, os.Interrupt)
	go func() {
		for _ = range signalChan {
			_ = stopService()
		}
	}()

	commandWait.Wait()
	return nil
}

func restartService() error {
	messageLog.Printf("Restarting service\n")

	// Increment the counter to prevent exiting unexpectedly
	commandWait.Add(1)

	err := stopService()
	if err != nil {
		return errors.WithStack(err)
	}
	err = startService()
	if err != nil {
		return errors.WithStack(err)
	}
	return nil
}

func stopService() error {
	command, err := runningService.GetCommand()
	if err != nil {
		messageLog.Printf("Could not get service command: %v\n", err)
	}

	var scriptErr error
	var scriptOutput []byte
	if runningService.Commands.Stop != "" {
		messageLog.Printf("Running stop script for %v.\n", runningService.Name)
		scriptOutput, scriptErr = command.RunStopScript()
		if scriptErr != nil {
			messageLog.Printf("%v\n", string(scriptOutput))
			messageLog.Printf("Stop script failed: %v\n", scriptErr)
		}
		if waitForCompletionWithTimeout(1 * time.Second) {
			return nil
		}
		messageLog.Printf("Stop script did not effectively stop service, sending interrupt\n")
	}

	err = runningCommand.Process.Signal(os.Interrupt)
	if err != nil {
		return errors.WithStack(err)
	}

	if waitForCompletionWithTimeout(1 * time.Second) {
		return nil
	}
	messageLog.Printf("Stop script did not effectively stop service, sending kill\n")

	err = runningCommand.Process.Kill()
	if err != nil {
		return errors.WithStack(err)
	}

	if waitForCompletionWithTimeout(1 * time.Second) {
		return nil
	}
	return errors.New("kill did not stop service")
}

func waitForCompletionWithTimeout(timeout time.Duration) bool {
	var completed = make(chan struct{})
	defer close(completed)
	timer := time.NewTimer(timeout)
	defer timer.Stop()

	go func() {
		runningCommand.Wait()
		completed <- struct{}{}
	}()

	select {
	case <-completed:
		return true
	case <-timer.C:
		return false
	}
}

func startService() error {
	messageLog.Printf("Service starting\n")
	command, cmdArgs, err := commandline.ParseCommand(os.ExpandEnv(runningService.Commands.Launch))
	if err != nil {
		return errors.WithStack(err)
	}

	standardLog := &RunnerLog{
		file:   logFile,
		name:   runningService.Name,
		stream: "stdout",
	}
	errorLog := &RunnerLog{
		file:   logFile,
		name:   runningService.Name,
		stream: "stderr",
	}

	cmd := exec.Command(command, cmdArgs...)
	if runningService.Path != nil {
		cmd.Dir = os.ExpandEnv(*runningService.Path)
	}
	cmd.Stdout = standardLog
	cmd.Stderr = errorLog

	runningCommand = cmd

	go func() {
		cmd.Run()
		commandWait.Done()
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
	fmt.Printf(format, a...)
	msg := fmt.Sprintf(format, a...)
	r.Write([]byte(msg))
}

func (r *RunnerLog) Write(p []byte) (int, error) {
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
