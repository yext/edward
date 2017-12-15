package services

import (
	"github.com/yext/edward/tracker"
	"github.com/yext/edward/worker"
)

// OperationConfig provides additional configuration for an operation
// on a service or group
type OperationConfig struct {
	WorkingDir       string
	EdwardExecutable string   // Path to the edward executable for launching runners
	Exclusions       []string // Names of services/groups to be excluded from this operation
	NoWatch          bool
	SkipBuild        bool
	Tags             []string // Tags to pass to `edward run`
	LogFile          string
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
	GetDescription() string
	Start(cfg OperationConfig, overrides ContextOverride, task tracker.Task, pool *worker.Pool) error  // Build and Launch this service/group
	Launch(cfg OperationConfig, overrides ContextOverride, task tracker.Task, pool *worker.Pool) error // Launch this service/group without building
	Stop(cfg OperationConfig, overrides ContextOverride, task tracker.Task, pool *worker.Pool) error
	Restart(cfg OperationConfig, overrides ContextOverride, task tracker.Task, pool *worker.Pool) error
	IsSudo(cfg OperationConfig) bool
	Watch() ([]ServiceWatch, error)
}

// ContextOverride defines overrides for service configuration caused by commandline
// flags or group configuration.
type ContextOverride struct {
	// Overrides to environment variables
	Env []string
}

func (c ContextOverride) Merge(m ContextOverride) ContextOverride {
	// TODO: Ensure that environment vars from c take precedence over m
	return ContextOverride{
		Env: append(m.Env, c.Env...),
	}
}

// CountServices returns the total number of services in the slice of services and groups.
func CountServices(sgs []ServiceOrGroup) int {
	return len(Services(sgs))
}

// Services returns a slice of services from a slice of services or groups.
func Services(sgs []ServiceOrGroup) []*ServiceConfig {
	var services []*ServiceConfig
	for _, sg := range sgs {
		switch v := sg.(type) {
		case *ServiceConfig:
			services = append(services, v)
		case *ServiceGroupConfig:
			services = append(services, getGroupServices(v)...)
		}
	}
	return services
}

func getGroupServices(group *ServiceGroupConfig) []*ServiceConfig {
	var services []*ServiceConfig
	for _, g := range group.Groups {
		services = append(services, getGroupServices(g)...)
	}
	services = append(services, group.Services...)
	return services
}
