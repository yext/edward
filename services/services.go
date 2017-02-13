package services

import "time"

type ServiceStatus struct {
	Service     *ServiceConfig
	Status      string
	Pid         int
	StartTime   time.Time
	Ports       []string
	StderrCount int
	StdoutCount int
}

// OperationConfig provides additional configuration for an operation
// on a service or group
type OperationConfig struct {
	Exclusions []string // Names of services/groups to be excluded from this operation
	NoWatch    bool
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

func CountServices(sgs []ServiceOrGroup) int {
	var count int
	for _, sg := range sgs {
		switch v := sg.(type) {
		case *ServiceConfig:
			count++
		case *ServiceGroupConfig:
			count += countGroupServices(v)
		}
	}
	return count
}

func countGroupServices(group *ServiceGroupConfig) int {
	var count int
	for _, g := range group.Groups {
		count += countGroupServices(g)
	}
	count += len(group.Services)
	return count
}
