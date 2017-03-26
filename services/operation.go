package services

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"os/signal"
	"time"

	"github.com/fatih/color"
	"github.com/yext/edward/common"
)

// OperationTracker provides functions for tracking the progress of an operaiton on a service
type OperationTracker interface {
	Start()
	Success()
	SoftFail(err error)
	Fail(err error)
	FailWithOutput(err error, output string)
}

var _ OperationTracker = &CommandTracker{}

// CommandTracker follows an operation executed by running a shell command
type CommandTracker struct {
	Name       string
	OutputFile string
	Logger     common.Logger
	sigChan    chan os.Signal
	endChan    chan struct{}
	startTime  time.Time
}

func (c *CommandTracker) printf(format string, v ...interface{}) {
	if c.Logger == nil {
		return
	}
	c.Logger.Printf(format, v...)
}

func (c *CommandTracker) waitForInterrupt() {
	c.sigChan = make(chan os.Signal, 1)
	c.endChan = make(chan struct{}, 1)
	signal.Notify(c.sigChan, os.Interrupt)
	go func() {
		select {
		case _ = <-c.sigChan:
			c.printResult("Interrupted", color.FgRed)
			c.printf("%v Interrupted\n", c.Name)
			if len(c.OutputFile) > 0 {
				c.printFile(c.OutputFile)
			}
			os.Exit(1)
		case _ = <-c.endChan:
			signal.Reset(os.Interrupt)
			close(c.sigChan)
			return
		}
	}()
}

func (c *CommandTracker) endWait() {
	c.endChan <- struct{}{}
}

// Start is called when this shell command operation has started
func (c *CommandTracker) Start() {
	c.startTime = time.Now()
	fmt.Printf("%-50s", c.Name+"...")
	c.printf("%v\n", c.Name)
	c.waitForInterrupt()
}

// Success is called when this shell command operation has succeeded
func (c *CommandTracker) Success() {
	c.printResult("OK", color.FgGreen)
	c.printf("%v Succeeded\n", c.Name)
	c.endWait()
}

// SoftFail is called when this shell command operation fails in a warning state.
func (c *CommandTracker) SoftFail(err error) {
	c.printResult(err.Error(), color.FgYellow)
	c.printf("%v: %v\n", c.Name, err.Error())
	c.endWait()
}

// Fail is called when this shell command operation fails in an error state.
func (c *CommandTracker) Fail(err error) {
	c.printResult("Failed", color.FgRed)
	c.printf("%v Failed: %v\n", c.Name, err.Error())
	if len(c.OutputFile) > 0 {
		c.printFile(c.OutputFile)
	}
	c.endWait()
}

// FailWithOutput is called when this shell command operation fails in an error state.
// A string containing output from the command is also included to be displayed to the user.
func (c *CommandTracker) FailWithOutput(err error, output string) {
	c.printResult("Failed", color.FgRed)
	c.printf("%v Failed: %v\n", c.Name, err.Error())
	c.printf("%v\n", output)
	fmt.Println(output)
	c.endWait()
}

func (tracker *CommandTracker) printResult(message string, c color.Attribute) {
	print("[")
	color.Set(c)
	print(message)
	color.Unset()
	print("] ")
	fmt.Printf("(%v)\n", autoRoundTime(time.Since(tracker.startTime)))
}

func autoRoundTime(d time.Duration) time.Duration {
	if d > time.Hour {
		return roundTime(d, time.Second)
	}
	if d > time.Minute {
		return roundTime(d, time.Second)
	}
	if d > time.Second {
		return roundTime(d, time.Millisecond)
	}
	if d > time.Millisecond {
		return roundTime(d, time.Microsecond)
	}
	return d
}

// Based on the example at https://play.golang.org/p/QHocTHl8iR
func roundTime(d, r time.Duration) time.Duration {
	if r <= 0 {
		return d
	}
	neg := d < 0
	if neg {
		d = -d
	}
	if m := d % r; m+m < r {
		d = d - m
	} else {
		d = d + r - m
	}
	if neg {
		return -d
	}
	return d
}

type logLine struct {
	Stream  string
	Message string
}

func (c *CommandTracker) printFile(path string) {
	logFile, err := os.Open(path)
	defer logFile.Close()
	if err != nil {
		c.printf("%v", err)
		fmt.Print(err)
		return
	}
	scanner := bufio.NewScanner(logFile)
	for scanner.Scan() {
		text := scanner.Text()
		var lineData logLine
		err = json.Unmarshal([]byte(text), &lineData)
		if err != nil {
			c.printf("%v", err)
			fmt.Print(err)
			return
		}
		if lineData.Stream != "messages" {
			c.printf("%v\n", lineData.Message)
			fmt.Println(lineData.Message)
		}
	}

	// check for errors
	if err = scanner.Err(); err != nil {
		c.printf("%v", err)
		fmt.Print(err)
		return
	}
}
