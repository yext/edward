package ui

import "github.com/yext/edward/services"

type Provider interface {
	Infof(string, ...interface{})
	Errorf(string, ...interface{})

	Confirm(string, ...interface{}) bool

	List(services []services.ServiceOrGroup, groups []services.ServiceOrGroup)
}
