package fake

import (
	"io"

	"github.com/yext/edward/services"
)

type Loader struct {
}

func (l *Loader) New() services.Backend {
	return &Backend{}
}

func (l *Loader) Name() string {
	return "fake"
}

func (l *Loader) Handles(c services.Backend) bool {
	_, ok := c.(*Backend)
	return ok
}

func (l *Loader) Builder(s *services.ServiceConfig, b services.Backend) (services.Builder, error) {
	return &buildAndRun{}, nil
}

func (l *Loader) Runner(s *services.ServiceConfig, b services.Backend) (services.Runner, error) {
	return &buildAndRun{}, nil
}

type buildAndRun struct {
}

func (b *buildAndRun) Build(string, func(string) string, io.Writer) error {
	return nil
}

func (b *buildAndRun) Start(standardLog io.Writer, errorLog io.Writer) error {
	return nil
}

func (b *buildAndRun) Stop(workingDir string, getenv func(string) string) ([]byte, error) {
	return nil, nil
}

func (b *buildAndRun) Status() (services.BackendStatus, error) {
	return services.BackendStatus{}, nil
}

func (b *buildAndRun) Wait() {

}
