package commandline

import (
	"github.com/pkg/errors"
	"github.com/yext/edward/services"
)

var _ services.Backend = &CommandLineBackend{}

type CommandLineBackend struct {
	// Commands for managing the service
	Commands ServiceConfigCommands `json:"commands"`
}

// ServiceConfigCommands define the commands for building, launching and stopping a service
// All commands are optional
type ServiceConfigCommands struct {
	// Command to build
	Build string `json:"build,omitempty"`
	// Command to launch
	Launch string `json:"launch,omitempty"`
	// Optional command to stop
	Stop string `json:"stop,omitempty"`
}

func (c *CommandLineBackend) HasBuildStep() bool {
	return c.Commands.Build != ""
}

func (c *CommandLineBackend) HasLaunchStep() bool {
	return c.Commands.Launch != ""
}

func (c *CommandLineBackend) HasStopStep() bool {
	return c.Commands.Stop != ""
}

func GetConfigCommandLine(s *services.ServiceConfig) (*CommandLineBackend, error) {
	if cl, ok := s.BackendConfig.(*CommandLineBackend); ok {
		return cl, nil
	}
	return nil, errors.New("service was not a command line service")
}
