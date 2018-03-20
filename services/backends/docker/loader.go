package docker

import (
	"github.com/docker/docker/client"
	"github.com/pkg/errors"
	"github.com/yext/edward/services"
)

type Loader struct {
}

func (l *Loader) New() services.Backend {
	return &Backend{}
}

func (l *Loader) Name() string {
	return "docker"
}

func (l *Loader) Handles(c services.Backend) bool {
	_, ok := c.(*Backend)
	return ok
}

func (l *Loader) Builder(s *services.ServiceConfig, backend services.Backend) (services.Builder, error) {
	return l.buildandrun(s, backend)
}

func (l *Loader) Runner(s *services.ServiceConfig, backend services.Backend) (services.Runner, error) {
	return l.buildandrun(s, backend)
}

func (l *Loader) buildandrun(s *services.ServiceConfig, backend services.Backend) (*buildandrun, error) {
	client, err := client.NewEnvClient()
	if err != nil {
		return nil, errors.WithMessage(err, "initializing client")
	}
	if config, ok := backend.(*Backend); ok {
		return &buildandrun{
			Service: s,
			Backend: config,
			client:  client,
		}, nil
	}
	return nil, errors.New("config was not of expected type")
}
