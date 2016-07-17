package services

var _ ServiceOrGroup = ServiceGroupConfig{}

// ServiceGroupConfig is a group of services that can be managed together
type ServiceGroupConfig struct {
	// A name for this group, used to identify it in commands
	Name string
	// Full services contained within this group
	Services []*ServiceConfig
	// Groups on which this group depends
	Groups []*ServiceGroupConfig
}

func (sg ServiceGroupConfig) GetName() string {
	return sg.Name
}

func (sg ServiceGroupConfig) Build() error {
	println("Building group: ", sg.Name)
	for _, group := range sg.Groups {
		err := group.Build()
		if err != nil {
			return err
		}
	}
	for _, service := range sg.Services {
		err := service.Build()
		if err != nil {
			return err
		}
	}
	return nil
}

func (sg ServiceGroupConfig) Start() error {
	println("Starting group:", sg.Name)
	for _, group := range sg.Groups {
		err := group.Start()
		if err != nil {
			// Always fail if any services in a dependant group failed
			return err
		}
	}
	var outErr error = nil
	for _, service := range sg.Services {
		err := service.Start()
		if err != nil {
			return err
		}
	}
	return outErr
}

func (sg ServiceGroupConfig) Stop() error {
	println("=== Group:", sg.Name, "===")
	// TODO: Do this in reverse
	for _, service := range sg.Services {
		_ = service.Stop()
	}
	for _, group := range sg.Groups {
		_ = group.Stop()
	}
	return nil
}

func (sg ServiceGroupConfig) GetStatus() []ServiceStatus {
	var outStatus []ServiceStatus
	for _, service := range sg.Services {
		outStatus = append(outStatus, service.GetStatus()...)
	}
	for _, group := range sg.Groups {
		outStatus = append(outStatus, group.GetStatus()...)
	}
	return outStatus
}

func (sg ServiceGroupConfig) IsSudo() bool {
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
