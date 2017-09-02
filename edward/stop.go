package edward

import (
	"github.com/pkg/errors"
	"github.com/yext/edward/config"
	"github.com/yext/edward/services"
	"github.com/yext/edward/tracker"
	"github.com/yext/edward/worker"
)

func (c *Client) Stop(names []string, force bool, exclude []string) error {
	var sgs []services.ServiceOrGroup
	var err error
	if len(names) == 0 {
		// Prompt user to confirm
		if !force && !c.askForConfirmation("Are you sure you want to stop all services?") {
			return nil
		}
		allSrv := config.GetAllServicesSorted()
		for _, service := range allSrv {
			var s []services.ServiceStatus
			s, err = service.Status()
			if err != nil {
				return errors.WithStack(err)
			}
			for _, status := range s {
				if status.Status != services.StatusStopped {
					sgs = append(sgs, service)
				}
			}
		}
	} else {
		sgs, err = config.GetServicesOrGroups(names)
		if err != nil {
			return errors.WithStack(err)
		}
	}

	// Perform required checks and actions for services
	if c.ServiceChecks != nil {
		if err = c.ServiceChecks(sgs); err != nil {
			return errors.WithStack(err)
		}
	}

	cfg := services.OperationConfig{
		EdwardExecutable: c.EdwardExecutable,
		Exclusions:       exclude,
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
	for _, s := range sgs {
		_ = s.Stop(cfg, services.ContextOverride{}, task, p)
	}
	return nil
}
