package commandline

import (
	"github.com/pkg/errors"
	"github.com/yext/edward/services"
)

type Loader struct {
}

func (l *Loader) New() services.Backend {
	return &Backend{}
}

func (l *Loader) Name() string {
	return "commandline"
}

func (l *Loader) Handles(c services.Backend) bool {
	_, ok := c.(*Backend)
	return ok
}

func (l *Loader) Builder(s *services.ServiceConfig, b services.Backend) (services.Builder, error) {
	return l.buildandrun(s, b)
}

func (l *Loader) Runner(s *services.ServiceConfig, b services.Backend) (services.Runner, error) {
	return l.buildandrun(s, b)
}

func (l *Loader) buildandrun(s *services.ServiceConfig, b services.Backend) (*buildandrun, error) {
	if config, ok := b.(*Backend); ok {
		return &buildandrun{
			Service: s,
			Backend: config,
		}, nil
	}
	return nil, errors.New("config was not of expected type")
}
