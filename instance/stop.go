package instance

import (
	"github.com/pkg/errors"
	"github.com/yext/edward/home"
	"github.com/yext/edward/instance/processes"
	"github.com/yext/edward/services"
	"github.com/yext/edward/tracker"
	"github.com/yext/edward/worker"
)

// Stop stops this service
func Stop(dirConfig *home.EdwardConfiguration, c *services.ServiceConfig, cfg services.OperationConfig, overrides services.ContextOverride, task tracker.Task, pool *worker.Pool) error {
	instance, err := Load(dirConfig, &processes.Processes{}, c, overrides)
	if err != nil {
		return errors.WithStack(err)
	}
	if instance.Pid == 0 {
		instance.clearState()
		return nil
	}
	err = pool.Enqueue(func() error {
		return errors.WithStack(instance.StopSync(cfg, overrides, task))
	})
	return errors.WithStack(err)
}
