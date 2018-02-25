package commandline

import (
	"github.com/pkg/errors"
	"github.com/yext/edward/services"
)

var _ services.ConfigType = &ConfigCommandLine{}

type ConfigCommandLine struct {
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

func (c *ConfigCommandLine) HasBuildStep() bool {
	return c.Commands.Build != ""
}

func (c *ConfigCommandLine) HasLaunchStep() bool {
	return c.Commands.Launch != ""
}

func GetConfigCommandLine(s *services.ServiceConfig) (*ConfigCommandLine, error) {
	if cl, ok := s.TypeConfig.(*ConfigCommandLine); ok {
		return cl, nil
	}
	return nil, errors.New("service was not a command line service")
}
