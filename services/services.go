package services

import "time"

type ServiceStatus struct {
	Service   *ServiceConfig
	Status    string
	Pid       int
	StartTime time.Time
	Ports     []string
}

type ServiceOrGroup interface {
	GetName() string
	Build() error  // Build this service/group from source
	Start() error  // Build and Launch this service/group
	Launch() error // Launch this service/group without building
	Stop() error
	Status() ([]ServiceStatus, error)
	IsSudo() bool
	Watch() ([]ServiceWatch, error)
}
