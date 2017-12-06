package edward

import (
	"sort"

	"github.com/pkg/errors"
	"github.com/yext/edward/services"
	"github.com/yext/edward/tracker"
	"github.com/yext/edward/worker"
)

func (c *Client) Restart(names []string, force bool, skipBuild bool, tail bool, noWatch bool, exclude []string) error {

	if len(names) == 0 {
		// Prompt user to confirm the restart
		if !force && !c.askForConfirmation("Are you sure you want to restart all services?") {
			return nil
		}
		c.restartAll(skipBuild, tail, noWatch, exclude)
	} else {
		err := c.restartOneOrMoreServices(names, skipBuild, tail, noWatch, exclude)
		if err != nil {
			return errors.WithStack(err)
		}
	}

	if tail {
		return errors.WithStack(c.tailFromFlag(names))
	}
	return nil
}

func (c *Client) restartAll(skipBuild bool, tail bool, noWatch bool, exclude []string) error {
	var as []*services.ServiceConfig
	for _, service := range c.serviceMap {
		s, err := service.Status()
		if err != nil {
			return errors.WithStack(err)
		}
		for _, status := range s {
			if status.Status != services.StatusStopped {
				as = append(as, service)
			}
		}
	}

	sort.Sort(serviceConfigByPID(as))
	var serviceNames []string
	for _, service := range as {
		serviceNames = append(serviceNames, service.Name)
	}

	return errors.WithStack(c.restartOneOrMoreServices(serviceNames, skipBuild, tail, noWatch, exclude))
}

func (c *Client) restartOneOrMoreServices(serviceNames []string, skipBuild bool, tail bool, noWatch bool, exclude []string) error {
	sgs, err := c.getServicesOrGroups(serviceNames)
	if err != nil {
		return errors.WithStack(err)
	}
	if c.ServiceChecks != nil {
		if err = c.ServiceChecks(sgs); err != nil {
			return errors.WithStack(err)
		}
	}

	cfg := services.OperationConfig{
		WorkingDir:       c.WorkingDir,
		EdwardExecutable: c.EdwardExecutable,
		Exclusions:       exclude,
		SkipBuild:        skipBuild,
		NoWatch:          noWatch,
		Tags:             c.Tags,
		LogFile:          c.LogFile,
	}

	task := tracker.NewTask(c.Follower.Handle)
	defer c.Follower.Done()

	poolSize := 1
	if c.DisableConcurrentPhases {
		poolSize = 0
	}

	launchPool := worker.NewPool(poolSize)
	launchPool.Start()
	defer func() {
		launchPool.Stop()
		_ = <-launchPool.Complete()
	}()
	for _, s := range sgs {
		err = s.Restart(cfg, services.ContextOverride{}, task, launchPool)
		if err != nil {
			return errors.WithStack(err)
		}
	}
	return nil
}
