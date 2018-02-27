package services

import (
	"fmt"
	"io"

	"github.com/theothertomelliott/gopsutil-nocgo/process"
)

// BackendName identifies the manner in which this service is built and launched.
type BackendName string

type Backend interface {
	HasBuildStep() bool
	HasLaunchStep() bool
	HasStopStep() bool
}

type BackendLoader interface {
	New() Backend
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

var defaultType BackendName
var loaders = make(map[BackendName]BackendLoader)

func RegisterBackend(name BackendName, loader BackendLoader) {
	loaders[name] = loader
}

func SetDefaultBackend(name BackendName) {
	defaultType = name
}

func GetBuilder(s *ServiceConfig) (Builder, error) {
	for _, loader := range loaders {
		if loader.Handles(s.BackendConfig) {
			return loader.Builder(s)
		}
	}
	return nil, fmt.Errorf("builder not found for service '%s'", s.Name)
}

func GetRunner(s *ServiceConfig) (Runner, error) {
	for _, loader := range loaders {
		if loader.Handles(s.BackendConfig) {
			return loader.Runner(s)
		}
	}
	return nil, fmt.Errorf("runner not found for service '%s'", s.Name)
}
