package commandline

import (
	"github.com/pkg/errors"
	"github.com/yext/edward/commandline"
	"github.com/yext/edward/services"
)

type buildandrun struct {
	Service *services.ServiceConfig
	Backend *CommandLineBackend
}

var _ services.Builder = &buildandrun{}
var _ services.Runner = &buildandrun{}

func (b *buildandrun) Build(workingDir string, getenv func(string) string) ([]byte, error) {
	cmd, err := commandline.ConstructCommand(workingDir, b.Service.Path, b.Backend.Commands.Build, getenv)
	if err != nil {
		return nil, errors.WithStack(err)
	}

	out, err := cmd.CombinedOutput()
	return out, errors.WithStack(err)
}

func (b *buildandrun) Start() error {
	return nil
}

func (b *buildandrun) Stop() error {
	return nil
}
