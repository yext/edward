package main

import (
	"errors"
	"fmt"
	"os"
	"os/exec"

	"github.com/urfave/cli"
)

var runnerCommand = cli.Command{
	Name:   "run",
	Hidden: true,
	Action: run,
}

func run(c *cli.Context) error {
	args := c.Args()
	if len(args) < 3 {
		return errors.New("a directory, log file and command is required")
	}
	workingDir := args[0]
	logFile := args[1]
	command := args[2]
	var cmdArgs []string
	if len(args) > 3 {
		cmdArgs = args[3:]
	}

	log, err := os.Create(logFile)
	if err != nil {
		return err
	}

	cmd := exec.Command(command, cmdArgs...)
	fmt.Println(cmd.Path)
	cmd.Dir = workingDir
	cmd.Stdout = log
	cmd.Stderr = log
	return cmd.Run()
}
