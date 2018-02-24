package services

import "github.com/pkg/errors"

type ConfigType interface {
	HasBuildStep() bool
	HasLaunchStep() bool
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
