package commandline

import (
	"github.com/pkg/errors"
	"github.com/yext/edward/services"
)

func init() {
	services.RegisterServiceType(services.TypeCommandLine, &CommandLineLoader{})
}

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

type CommandLineLoader struct {
}

func (l *CommandLineLoader) New() services.ConfigType {
	return &ConfigCommandLine{}
}

func (l *CommandLineLoader) Handles(c services.ConfigType) bool {
	_, ok := c.(*ConfigCommandLine)
	return ok
}

func (l *CommandLineLoader) Builder(s *services.ServiceConfig) (services.Builder, error) {
	return nil, nil
}

func (l *CommandLineLoader) Runner(s *services.ServiceConfig) (services.Runner, error) {
	return nil, nil
}
