package instance

import (
	"github.com/pkg/errors"
	"github.com/yext/edward/home"
	"github.com/yext/edward/services"
)

// HasRunning returns true iff the specified service has a currently running instance
func HasRunning(dirConfig *home.EdwardConfiguration, service *services.ServiceConfig) (bool, error) {
	command, err := Load(dirConfig, service, services.ContextOverride{})
	if err != nil {
		return false, errors.WithStack(err)
	}
	if command.Pid == 0 {
		return false, nil
	}
	return true, nil
}
