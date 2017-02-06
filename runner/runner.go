package runner

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/pkg/errors"
	"github.com/urfave/cli"
	"github.com/yext/edward/commandline"
	"github.com/yext/edward/config"
)

var Command = cli.Command{
	Name:   "run",
	Hidden: true,
	Action: run,
}

func run(c *cli.Context) error {
	args := c.Args()
	if len(args) < 1 {
		return errors.New("a service name is required")
	}

	service, ok := config.GetServiceMap()[args[0]]
	if !ok {
		return errors.New("service not found")
	}

	logFile := service.GetRunLog()
	os.Remove(logFile)

	command, cmdArgs, err := commandline.ParseCommand(service.Commands.Launch)
	if err != nil {
		return errors.WithStack(err)
	}

	log, err := os.Create(logFile)
	if err != nil {
		return errors.WithStack(err)
	}

	standardLog := &RunnerLog{
		file:   log,
		name:   service.Name,
		stream: "stdout",
	}
	errorLog := &RunnerLog{
		file:   log,
		name:   service.Name,
		stream: "stderr",
	}

	cmd := exec.Command(command, cmdArgs...)
	if service.Path != nil {
		cmd.Dir = os.ExpandEnv(*service.Path)
	}
	cmd.Stdout = standardLog
	cmd.Stderr = errorLog
	return cmd.Run()
}

// RunnerLog provides the io.Writer interface to publish service logs to file
type RunnerLog struct {
	file   *os.File
	name   string
	stream string
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
