package services

import (
	"fmt"
	"io/ioutil"
	"log"
	"os"

	"github.com/fatih/color"
	"github.com/yext/edward/common"
)

type OperationTracker interface {
	Start()
	Success()
	SoftFail(err error)
	Fail(err error)
}

var _ OperationTracker = &CommandTracker{}

// CommandTracker follows an operation executed by running a shell command
type CommandTracker struct {
	Name       string
	OutputFile string
	Logger     common.Logger
	sigChan    chan os.Signal
}

func (c *CommandTracker) printf(format string, v ...interface{}) {
	if c.Logger == nil {
		return
	}
	c.Logger.Printf(format, v...)
}

func (c *CommandTracker) Start() {
	fmt.Printf("%-50s", c.Name+"...")
	c.printf("%v\n", c.Name)
}

func (c *CommandTracker) Success() {
	printResult("OK", color.FgGreen)
	c.printf("%v Succeeded\n", c.Name)
}

func (c *CommandTracker) SoftFail(err error) {
	printResult(err.Error(), color.FgYellow)
	c.printf("%v: %v\n", c.Name, err.Error())
}

func (c *CommandTracker) Fail(err error) {
	printResult("Failed", color.FgRed)
	c.printf("%v Failed: %v\n", c.Name, err.Error())
	if len(c.OutputFile) > 0 {
		printFile(c.OutputFile)
	}
}

func printResult(message string, c color.Attribute) {
	print("[")
	color.Set(c)
	print(message)
	color.Unset()
	println("]")
}

func printFile(path string) {
	dat, errRead := ioutil.ReadFile(path)
	if errRead != nil {
		log.Println(errRead)
	}
	fmt.Print(string(dat))
}
