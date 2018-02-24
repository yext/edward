package services

import "github.com/pkg/errors"

type ConfigCommandLine struct {
	// Commands for managing the service
	Commands ServiceConfigCommands `json:"commands"`
}

func GetConfigCommandLine(s *ServiceConfig) (*ConfigCommandLine, error) {
	if cl, ok := s.TypeConfig.(*ConfigCommandLine); ok {
		return cl, nil
	}
	return nil, errors.New("service was not a command line service")
}
