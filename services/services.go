package services

type ServiceStatus struct {
	Service *ServiceConfig
	Status  string
	Pid     int
}

type ServiceOrGroup interface {
	GetName() string
	Build() error
	Start() error
	Stop() error
	Status() ([]ServiceStatus, error)
	IsSudo() bool
}
