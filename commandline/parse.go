package commandline

import (
	"fmt"
)

const quotes = "quotes"
const start = "start"
const arg = "arg"

// ParseCommand returns the executable path and arguments
// TODO: Clean this up
func ParseCommand(cmd string) (string, []string, error) {
	var args []string
	state := start
	current := ""
	quote := "\""
	for i := 0; i < len(cmd); i++ {
		c := cmd[i]

		if state == quotes {
			if string(c) != quote {
				current += string(c)
			} else {
				args = append(args, current)
				current = ""
				state = start
			}
			continue
		}

		if c == '"' || c == '\'' {
			state = quotes
			quote = string(c)
			continue
		}

		if state == arg {
			if c == ' ' || c == '\t' {
				args = append(args, current)
				current = ""
				state = start
			} else {
				current += string(c)
			}
			continue
		}

		if c != ' ' && c != '\t' {
			state = arg
			current += string(c)
		}
	}

	if state == quotes {
		return "", []string{}, fmt.Errorf("Unclosed quote in command line: %s", cmd)
	}

	if current != "" {
		args = append(args, current)
	}

	if len(args) <= 0 {
		return "", []string{}, fmt.Errorf("Empty command line built from: '%s'", cmd)
	}

	if len(args) == 1 {
		return args[0], []string{}, nil
	}

	return args[0], args[1:], nil
}
