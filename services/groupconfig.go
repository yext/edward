package services

import (
	"github.com/juju/errgo"
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

func (sg *ServiceGroupConfig) GetName() string {
	return sg.Name
}

func (sg *ServiceGroupConfig) Build(cfg OperationConfig) error {
	if cfg.IsExcluded(sg) {
		return nil
	}

	println("Building group: ", sg.Name)
	for _, group := range sg.Groups {
		err := group.Build(cfg)
		if err != nil {
			return err
		}
	}
	for _, service := range sg.Services {
		err := service.Build(cfg)
		if err != nil {
			return err
		}
	}
	return nil
}

func (sg *ServiceGroupConfig) Start(cfg OperationConfig) error {
	if cfg.IsExcluded(sg) {
		return nil
	}

	println("Starting group:", sg.Name)
	for _, group := range sg.Groups {
		err := group.Start(cfg)
		if err != nil {
			// Always fail if any services in a dependant group failed
			return err
		}
	}
	var outErr error = nil
	for _, service := range sg.Services {
		err := service.Start(cfg)
		if err != nil {
			return err
		}
	}
	return outErr
}

func (sg *ServiceGroupConfig) Launch(cfg OperationConfig) error {
	if cfg.IsExcluded(sg) {
		return nil
	}

	println("Launching group:", sg.Name)
	for _, group := range sg.Groups {
		err := group.Launch(cfg)
		if err != nil {
			// Always fail if any services in a dependant group failed
			return err
		}
	}
	var outErr error = nil
	for _, service := range sg.Services {
		err := service.Launch(cfg)
		if err != nil {
			return err
		}
	}
	return outErr
}

func (sg *ServiceGroupConfig) Stop(cfg OperationConfig) error {
	if cfg.IsExcluded(sg) {
		return nil
	}

	println("=== Group:", sg.Name, "===")
	// TODO: Do this in reverse
	for _, service := range sg.Services {
		err := service.Stop(cfg)
		if err != nil {
			return errgo.Mask(err)
		}
	}
	for _, group := range sg.Groups {
		err := group.Stop(cfg)
		if err != nil {
			return errgo.Mask(err)
		}
	}
	return nil
}

func (sg *ServiceGroupConfig) Status() ([]ServiceStatus, error) {
	var outStatus []ServiceStatus
	for _, service := range sg.Services {
		statuses, err := service.Status()
		if err != nil {
			return outStatus, errgo.Mask(err)
		}
		outStatus = append(outStatus, statuses...)
	}
	for _, group := range sg.Groups {
		statuses, err := group.Status()
		if err != nil {
			return outStatus, errgo.Mask(err)
		}
		outStatus = append(outStatus, statuses...)
	}
	return outStatus, nil
}

func (sg *ServiceGroupConfig) IsSudo() bool {
	for _, service := range sg.Services {
		if service.IsSudo() {
			return true
		}
	}
	for _, group := range sg.Groups {
		if group.IsSudo() {
			return true
		}
	}

	return false
}

func (sg *ServiceGroupConfig) Watch() ([]ServiceWatch, error) {
	var watches []ServiceWatch
	for _, service := range sg.Services {
		sw, err := service.Watch()
		if err != nil {
			return nil, err
		}
		watches = append(watches, sw...)
	}
	for _, group := range sg.Groups {
		gw, err := group.Watch()
		if err != nil {
			return nil, err
		}
		watches = append(watches, gw...)
	}
	return watches, nil
}
