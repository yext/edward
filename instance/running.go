package instance

import (
	"github.com/pkg/errors"
	"github.com/yext/edward/services"
)

// HasRunning returns true iff the specified service has a currently running instance
func HasRunning(service *services.ServiceConfig) (bool, error) {
	command, err := service.GetCommand(services.ContextOverride{})
	if err != nil {
		return false, errors.WithStack(err)
	}
	if command.Pid == 0 {
		return false, nil
	}
	return true, nil
}
