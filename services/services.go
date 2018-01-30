package services

import (
	"github.com/pkg/errors"
	"github.com/yext/edward/tracker"
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

// DoForServices performs a taks for a set of services
func DoForServices(sgs []ServiceOrGroup, task tracker.Task, f func(service *ServiceConfig, overrides ContextOverride, task tracker.Task) error) error {
	for _, sg := range sgs {
		switch v := sg.(type) {
		case *ServiceConfig:
			err := f(v, ContextOverride{}, task)
			if err != nil {
				return errors.WithStack(err)
			}
		case *ServiceGroupConfig:
			t := task.Child(v.Name)
			err := DoForServices(v.Children(), t, func(service *ServiceConfig, overrides ContextOverride, task tracker.Task) error {
				return errors.WithStack(f(service, v.getOverrides(overrides), task))
			})
			if err != nil {
				return errors.WithStack(err)
			}
		}
	}
	return nil
}

func getGroupServices(group *ServiceGroupConfig) []*ServiceConfig {
	var services []*ServiceConfig
	for _, g := range group.Groups {
		services = append(services, getGroupServices(g)...)
	}
	services = append(services, group.Services...)
	return services
}
