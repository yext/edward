package services

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

// Children returns a slice of all children of this group in the configured order
func (c *ServiceGroupConfig) Children() []ServiceOrGroup {
	var children []ServiceOrGroup
	for _, name := range c.ChildOrder {
		children = append(children, c.getChild(name))
	}
	return children
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
