package services

import "fmt"

type ConfigType interface {
	HasBuildStep() bool
	HasLaunchStep() bool
}

type TypeLoader interface {
	New() ConfigType
	Handles(ConfigType) bool
	Builder(*ServiceConfig) (Builder, error)
	Runner(*ServiceConfig) (Runner, error)
}

type Runner interface {
	Start() error
	Stop() error
}

type Builder interface {
	Build() error
}

var loaders = make(map[Type]TypeLoader)

func RegisterServiceType(name Type, loader TypeLoader) {
	loaders[name] = loader
}

func GetBuilder(s *ServiceConfig) (Builder, error) {
	for _, loader := range loaders {
		if loader.Handles(s.TypeConfig) {
			return loader.Builder(s)
		}
	}
	return nil, fmt.Errorf("builder not found for service '%s'", s.Name)
}

func GetRunner(s *ServiceConfig) (Runner, error) {
	for _, loader := range loaders {
		if loader.Handles(s.TypeConfig) {
			return loader.Runner(s)
		}
	}
	return nil, fmt.Errorf("runner not found for service '%s'", s.Name)
}
