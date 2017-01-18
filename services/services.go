package services

import "time"

type ServiceStatus struct {
	Service   *ServiceConfig
	Status    string
	Pid       int
	StartTime time.Time
	Ports     []string
}

// OperationConfig provides additional configuration for an operation
// on a service or group
type OperationConfig struct {
	Exclusions []string // Names of services/groups to be excluded from this operation
}

func (o *OperationConfig) IsExcluded(sg ServiceOrGroup) bool {
	name := sg.GetName()
	for _, e := range o.Exclusions {
		if name == e {
			return true
		}
	}
	return false
}

type ServiceOrGroup interface {
	GetName() string
	Build(cfg OperationConfig) error  // Build this service/group from source
	Start(cfg OperationConfig) error  // Build and Launch this service/group
	Launch(cfg OperationConfig) error // Launch this service/group without building
	Stop(cfg OperationConfig) error
	Status() ([]ServiceStatus, error)
	IsSudo(cfg OperationConfig) bool
	Watch() ([]ServiceWatch, error)
}
