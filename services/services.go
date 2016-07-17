package services

import (
	"fmt"
	"io/ioutil"
	"log"

	"github.com/fatih/color"
)

type ServiceStatus struct {
	Service *ServiceConfig
	Status  string
}

type ServiceOrGroup interface {
	GetName() string
	Build() error
	Start() error
	Stop() error
	GetStatus() []ServiceStatus
	IsSudo() bool
}

func printOperation(operation string) {
	fmt.Printf("%-50s", operation+"...")
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
