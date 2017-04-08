package services

import (
	"github.com/pkg/errors"
	"github.com/yext/edward/common"
)

var _ ServiceOrGroup = &ServiceGroupConfig{}

// ServiceGroupConfig is a group of services that can be managed together
type ServiceGroupConfig struct {
	// A name for this group, used to identify it in commands
	Name string
	// Full services contained within this group
	Services []*ServiceConfig
	// Groups on which this group depends
	Groups []*ServiceGroupConfig

	Logger common.Logger
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

// Build builds all services within this group
func (c *ServiceGroupConfig) Build(cfg OperationConfig) error {
	if cfg.IsExcluded(c) {
		return nil
	}

	println("Building group: ", c.Name)
	for _, group := range c.Groups {
		err := group.Build(cfg)
		if err != nil {
			return errors.WithStack(err)
		}
	}
	for _, service := range c.Services {
		err := service.Build(cfg)
		if err != nil {
			return errors.WithStack(err)
		}
	}
	return nil
}

// Start will build and launch all services within this group
func (c *ServiceGroupConfig) Start(cfg OperationConfig) error {
	if cfg.IsExcluded(c) {
		return nil
	}

	for _, group := range c.Groups {
		err := group.Start(cfg)
		if err != nil {
			// Always fail if any services in a dependant group failed
			return errors.WithStack(err)
		}
	}
	var outErr error
	for _, service := range c.Services {
		err := service.Start(cfg)
		if err != nil {
			return errors.WithStack(err)
		}
	}
	return outErr
}

// Launch will launch all services within this group
func (c *ServiceGroupConfig) Launch(cfg OperationConfig) error {
	if cfg.IsExcluded(c) {
		return nil
	}

	println("Launching group:", c.Name)
	for _, group := range c.Groups {
		err := group.Launch(cfg)
		if err != nil {
			// Always fail if any services in a dependant group failed
			return errors.WithStack(err)
		}
	}
	var outErr error
	for _, service := range c.Services {
		err := service.Launch(cfg)
		if err != nil {
			return errors.WithStack(err)
		}
	}
	return outErr
}

// Stop stops all services within this group
func (c *ServiceGroupConfig) Stop(cfg OperationConfig) error {
	if cfg.IsExcluded(c) {
		return nil
	}

	// TODO: Do this in reverse
	for _, service := range c.Services {
		err := service.Stop(cfg)
		if err != nil {
			return errors.WithStack(err)
		}
	}
	for _, group := range c.Groups {
		err := group.Stop(cfg)
		if err != nil {
			return errors.WithStack(err)
		}
	}
	return nil
}

// Status returns the status for all services within this group
func (c *ServiceGroupConfig) Status() ([]ServiceStatus, error) {
	var outStatus []ServiceStatus
	for _, service := range c.Services {
		statuses, err := service.Status()
		if err != nil {
			return outStatus, errors.WithStack(err)
		}
		outStatus = append(outStatus, statuses...)
	}
	for _, group := range c.Groups {
		statuses, err := group.Status()
		if err != nil {
			return outStatus, errors.WithStack(err)
		}
		outStatus = append(outStatus, statuses...)
	}
	return outStatus, nil
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
