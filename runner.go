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
	fullCommand := args[2]

	command, cmdArgs, err := parseCommand(fullCommand)
	if err != nil {
		return err
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
