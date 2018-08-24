package edward

import (
	"github.com/pkg/errors"
	"github.com/yext/edward/instance"
	"github.com/yext/edward/services"
	"github.com/yext/edward/tracker"
	"github.com/yext/edward/worker"
)

func (c *Client) Stop(names []string, force bool, exclude []string, all bool) error {
	// Prompt user to confirm as needed
	if len(names) == 0 && !force && !c.askForConfirmation("Are you sure you want to stop all services?") {
		return nil
	}

	sgs, err := c.getServiceList(names, all)
	if err != nil {
		return errors.WithStack(err)
	}

	// Perform required checks and actions for services
	if c.ServiceChecks != nil {
		if err = c.ServiceChecks(sgs); err != nil {
			return errors.WithStack(err)
		}
	}

	cfg := services.OperationConfig{
		WorkingDir:       c.WorkingDir,
		EdwardExecutable: c.EdwardExecutable,
		Exclusions:       exclude,
		Tags:             c.Tags,
		LogFile:          c.LogFile,
		Backends:         c.Backends,
	}

	task := tracker.NewTask(c.Follower.Handle)
	defer c.Follower.Done()

	poolSize := 3
	if c.DisableConcurrentPhases {
		poolSize = 0
	}

	p := worker.NewPool(poolSize)
	p.Start()
	defer func() {
		p.Stop()
		_ = <-p.Complete()
	}()

	err = services.DoForServices(sgs, task, func(s *services.ServiceConfig, overrides services.ContextOverride, task tracker.Task) error {
		_ = instance.Stop(c.DirConfig, s, cfg, overrides, task, p)
		return nil
	})
	if err != nil {
		return errors.WithStack(err)
	}
	return nil
}
