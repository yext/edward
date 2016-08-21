package services

import "time"

type ServiceStatus struct {
	Service   *ServiceConfig
	Status    string
	Pid       int
	StartTime time.Time
}

type ServiceOrGroup interface {
	GetName() string
	Build() error
	Start() error
	Stop() error
	Status() ([]ServiceStatus, error)
	IsSudo() bool
}
