package instance

import (
	"github.com/pkg/errors"
	"github.com/yext/edward/home"
	"github.com/yext/edward/services"
	"github.com/yext/edward/tracker"
	"github.com/yext/edward/worker"
)

// Launch launches this service
func Launch(dirConfig *home.EdwardConfiguration, c *services.ServiceConfig, cfg services.OperationConfig, overrides services.ContextOverride, task tracker.Task, pool *worker.Pool) error {
	if cfg.IsExcluded(c) {
		return nil
	}

	instance, err := Load(dirConfig, c, overrides)
	if err != nil {
		return errors.WithStack(err)
	}

	err = pool.Enqueue(func() error {
		return errors.WithStack(instance.StartAsync(cfg, task))
	})
	return errors.WithStack(err)
}
