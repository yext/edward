package commandline

import (
	"github.com/pkg/errors"
	"github.com/yext/edward/services"
)

type CommandLineLoader struct {
}

func (l *CommandLineLoader) New() services.Backend {
	return &CommandLineBackend{}
}

func (l *CommandLineLoader) Name() string {
	return "commandline"
}

func (l *CommandLineLoader) Handles(c services.Backend) bool {
	_, ok := c.(*CommandLineBackend)
	return ok
}

func (l *CommandLineLoader) Builder(s *services.ServiceConfig) (services.Builder, error) {
	return l.buildandrun(s)
}

func (l *CommandLineLoader) Runner(s *services.ServiceConfig) (services.Runner, error) {
	return l.buildandrun(s)
}

func (l *CommandLineLoader) buildandrun(s *services.ServiceConfig) (*buildandrun, error) {
	if config, ok := s.BackendConfig.(*CommandLineBackend); ok {
		return &buildandrun{
			Service: s,
			Backend: config,
		}, nil
	}
	return nil, errors.New("config was not of expected type")
}
