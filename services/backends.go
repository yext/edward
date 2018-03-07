package services

import (
	"fmt"
	"io"

	"github.com/theothertomelliott/gopsutil-nocgo/process"
)

type Backend interface {
	HasBuildStep() bool
	HasLaunchStep() bool
}

type BackendLoader interface {
	New() Backend
	Name() string
	Handles(Backend) bool
	Builder(*ServiceConfig) (Builder, error)
	Runner(*ServiceConfig) (Runner, error)
}

type Runner interface {
	Start(standardLog io.Writer, errorLog io.Writer) error
	Stop(workingDir string, getenv func(string) string) ([]byte, error)
	Status() (BackendStatus, error)
	Wait()
}

type BackendStatus struct {
	Ports      []string
	MemoryInfo *process.MemoryInfoStat
}

type Builder interface {
	Build(string, func(string) string) ([]byte, error)
}

var defaultType string
var loaders = make(map[string]BackendLoader)

func RegisterBackend(loader BackendLoader) {
	loaders[loader.Name()] = loader
}

func RegisterDefaultBackend(loader BackendLoader) {
	loaders[loader.Name()] = loader
	defaultType = loader.Name()
}

func GetBuilder(s *ServiceConfig) (Builder, error) {
	for _, backend := range s.Backends {
		if loader, ok := loaders[backend.Type]; !ok {
			return loader.Builder(s)
		}
	}
	return nil, fmt.Errorf("builder not found for service '%s'", s.Name)
}

func GetRunner(s *ServiceConfig) (Runner, error) {
	for _, backend := range s.Backends {
		if loader, ok := loaders[backend.Type]; !ok {
			return loader.Runner(s)
		}
	}
	return nil, fmt.Errorf("runner not found for service '%s'", s.Name)
}
