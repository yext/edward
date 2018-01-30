package instance

import (
	"github.com/pkg/errors"
	"github.com/yext/edward/home"
	"github.com/yext/edward/services"
	"github.com/yext/edward/tracker"
	"github.com/yext/edward/worker"
)

// Stop stops this service
func Stop(dirConfig *home.EdwardConfiguration, c *services.ServiceConfig, cfg services.OperationConfig, overrides services.ContextOverride, task tracker.Task, pool *worker.Pool) error {
	instance, err := Load(dirConfig, c, overrides)
	if err != nil {
		return errors.WithStack(err)
	}
	err = pool.Enqueue(func() error {
		return errors.WithStack(instance.StopSync(cfg, overrides, task))
	})
	return errors.WithStack(err)
}
