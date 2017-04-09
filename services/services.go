package services

import (
	"time"

	"github.com/yext/edward/tracker"
)

// StatusRunning is the status string for a running service
const StatusRunning = "RUNNING"

// StatusStopped is the status string for a stopped service
const StatusStopped = "STOPPED"

// ServiceStatus contains the status for a service at a given point in time
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

// IsExcluded returns true if the given service/group is excluded by this OperationConfig.
// No operations should be performed on excluded services.
func (o *OperationConfig) IsExcluded(sg ServiceOrGroup) bool {
	name := sg.GetName()
	for _, e := range o.Exclusions {
		if name == e {
			return true
		}
	}
	return false
}

// ServiceOrGroup provides a common interface to services and groups
type ServiceOrGroup interface {
	GetName() string
	Build(cfg OperationConfig, tracker tracker.Operation) error  // Build this service/group from source
	Start(cfg OperationConfig, tracker tracker.Operation) error  // Build and Launch this service/group
	Launch(cfg OperationConfig, tracker tracker.Operation) error // Launch this service/group without building
	Stop(cfg OperationConfig, tracker tracker.Operation) error
	Status() ([]ServiceStatus, error)
	IsSudo(cfg OperationConfig) bool
	Watch() ([]ServiceWatch, error)
}

// CountServices returns the total number of services in the slice of services and groups.
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
