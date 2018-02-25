package services

import (
	"github.com/pkg/errors"
)

// TypeCommandLine identifies a service as being built and launched via the command line
const TypeCommandLine Type = "commandline"

func init() {
	RegisterServiceType(TypeCommandLine, &CommandLineLoader{})
}

var _ ConfigType = &ConfigCommandLine{}

type ConfigCommandLine struct {
	// Commands for managing the service
	Commands ServiceConfigCommands `json:"commands"`
}

func (c *ConfigCommandLine) HasBuildStep() bool {
	return c.Commands.Build != ""
}

func (c *ConfigCommandLine) HasLaunchStep() bool {
	return c.Commands.Launch != ""
}

func GetConfigCommandLine(s *ServiceConfig) (*ConfigCommandLine, error) {
	if cl, ok := s.TypeConfig.(*ConfigCommandLine); ok {
		return cl, nil
	}
	return nil, errors.New("service was not a command line service")
}

type CommandLineLoader struct {
}

func (l *CommandLineLoader) New() ConfigType {
	return &ConfigCommandLine{}
}

func (l *CommandLineLoader) Handles(c ConfigType) bool {
	_, ok := c.(*ConfigCommandLine)
	return ok
}

func (l *CommandLineLoader) Builder(s *ServiceConfig) (Builder, error) {
	return nil, nil
}

func (l *CommandLineLoader) Runner(s *ServiceConfig) (Runner, error) {
	return nil, nil
}
