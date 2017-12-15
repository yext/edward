package services

import (
	"fmt"

	"github.com/pkg/errors"
	"github.com/yext/edward/common"
	"github.com/yext/edward/tracker"
	"github.com/yext/edward/worker"
)

var _ ServiceOrGroup = &ServiceGroupConfig{}

// ServiceGroupConfig is a group of services that can be managed together
type ServiceGroupConfig struct {
	// A name for this group, used to identify it in commands
	Name string
	// Alternative names for this group
	Aliases []string
	// A description
	Description string
	// Full services contained within this group
	Services []*ServiceConfig
	// Groups on which this group depends
	Groups []*ServiceGroupConfig

	// Launch order for children
	ChildOrder []string

	// Environment variables to be passed to all child services
	Env []string

	Logger common.Logger
}

// Matches returns true if the group name or an alias matches the provided name.
func (c *ServiceGroupConfig) Matches(name string) bool {
	if c.Name == name {
		return true
	}
	for _, alias := range c.Aliases {
		if alias == name {
			return true
		}
	}
	return false
}

func (c *ServiceGroupConfig) printf(format string, v ...interface{}) {
	if c.Logger == nil {
		return
	}
	c.Logger.Printf(format, v...)
}

// GetName returns the name for this group
func (c *ServiceGroupConfig) GetName() string {
	return c.Name
}

// GetDescription returns the description for this group
func (c *ServiceGroupConfig) GetDescription() string {
	return c.Description
}

func (c *ServiceGroupConfig) getOverrides(o ContextOverride) ContextOverride {
	override := ContextOverride{
		Env: c.Env,
	}
	return override.Merge(o)
}

func (c *ServiceGroupConfig) getChild(name string) ServiceOrGroup {
	for _, group := range c.Groups {
		if group.Name == name {
			return group
		}
	}
	for _, service := range c.Services {
		if service.Name == name {
			return service
		}
	}
	return nil
}

// Start will build and launch all services within this group
func (c *ServiceGroupConfig) Start(cfg OperationConfig, overrides ContextOverride, task tracker.Task, pool *worker.Pool) error {
	if cfg.IsExcluded(c) {
		return nil
	}
	groupTracker := task.Child(c.GetName())

	for _, childName := range c.ChildOrder {
		child := c.getChild(childName)
		if child == nil {
			return fmt.Errorf("Child not found: %s", childName)
		}
		err := child.Start(cfg, c.getOverrides(overrides), groupTracker, pool)
		if err != nil {
			// Always fail if any services in a dependant group failed
			return errors.WithStack(err)
		}
	}
	return nil
}

// Launch will launch all services within this group
func (c *ServiceGroupConfig) Launch(cfg OperationConfig, overrides ContextOverride, task tracker.Task, pool *worker.Pool) error {
	if cfg.IsExcluded(c) {
		return nil
	}

	groupTracker := task.Child(c.GetName())
	for _, group := range c.Groups {
		err := group.Launch(cfg, c.getOverrides(overrides), groupTracker, pool)
		if err != nil {
			// Always fail if any services in a dependant group failed
			return errors.WithStack(err)
		}
	}
	var outErr error
	for _, service := range c.Services {
		err := service.Launch(cfg, c.getOverrides(overrides), groupTracker, pool)
		if err != nil {
			return errors.WithStack(err)
		}
	}
	return outErr
}

// Stop stops all services within this group
func (c *ServiceGroupConfig) Stop(cfg OperationConfig, overrides ContextOverride, task tracker.Task, pool *worker.Pool) error {
	if cfg.IsExcluded(c) {
		return nil
	}
	groupTracker := task.Child(c.GetName())

	// TODO: Do this in reverse
	for _, service := range c.Services {
		err := service.Stop(cfg, c.getOverrides(overrides), groupTracker, pool)
		if err != nil {
			return errors.WithStack(err)
		}
	}
	for _, group := range c.Groups {
		err := group.Stop(cfg, c.getOverrides(overrides), groupTracker, pool)
		if err != nil {
			return errors.WithStack(err)
		}
	}
	return nil
}

// Restart restarts all services within this group
func (c *ServiceGroupConfig) Restart(cfg OperationConfig, overrides ContextOverride, task tracker.Task, pool *worker.Pool) error {
	if cfg.IsExcluded(c) {
		return nil
	}
	groupTracker := task.Child(c.GetName())

	// TODO: Do this in reverse
	for _, service := range c.Services {
		err := service.Restart(cfg, c.getOverrides(overrides), groupTracker, pool)
		if err != nil {
			return errors.WithStack(err)
		}
	}
	for _, group := range c.Groups {
		err := group.Restart(cfg, c.getOverrides(overrides), groupTracker, pool)
		if err != nil {
			return errors.WithStack(err)
		}
	}
	return nil
}

// IsSudo returns true if any of the services in this group require sudo to run
func (c *ServiceGroupConfig) IsSudo(cfg OperationConfig) bool {
	if cfg.IsExcluded(c) {
		return false
	}
	for _, service := range c.Services {
		if service.IsSudo(cfg) {
			return true
		}
	}
	for _, group := range c.Groups {
		if group.IsSudo(cfg) {
			return true
		}
	}

	return false
}

// Watch returns all service watches configured for this group
func (c *ServiceGroupConfig) Watch() ([]ServiceWatch, error) {
	var watches []ServiceWatch
	for _, service := range c.Services {
		sw, err := service.Watch()
		if err != nil {
			return nil, err
		}
		watches = append(watches, sw...)
	}
	for _, group := range c.Groups {
		gw, err := group.Watch()
		if err != nil {
			return nil, err
		}
		watches = append(watches, gw...)
	}
	return watches, nil
}
