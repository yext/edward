package ui

import (
	"github.com/yext/edward/instance"
	"github.com/yext/edward/instance/servicelogs"
	"github.com/yext/edward/services"
)

type Provider interface {
	Infof(string, ...interface{})
	Errorf(string, ...interface{})

	Confirm(string, ...interface{}) bool

	List(services []services.ServiceOrGroup, groups []services.ServiceOrGroup)

	Status([]ServiceStatus)

	ShowLog(<-chan servicelogs.LogLine, bool)
}

type ServiceStatus interface {
	Status() instance.Status
	Service() *services.ServiceConfig
	Pid() int
}
