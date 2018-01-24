package edward

import (
	"sort"

	"github.com/pkg/errors"
	"github.com/yext/edward/builder"
	"github.com/yext/edward/instance"
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
	var instances []*instance.Instance
	for _, service := range c.serviceMap {
		running, err := instance.HasRunning(c.DirConfig, service)
		if err != nil {
			return errors.WithStack(err)
		}
		if running {
			i, err := instance.Load(c.DirConfig, service, services.ContextOverride{})
			if err != nil {
				return errors.WithStack(err)
			}
			instances = append(instances, i)
		}
	}

	sort.Slice(instances, func(i, j int) bool {
		return instances[i].Pid < instances[j].Pid
	})

	var serviceNames []string
	for _, instance := range instances {
		serviceNames = append(serviceNames, instance.Service.Name)
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
	services.DoForServices(sgs, task, func(service *services.ServiceConfig, overrides services.ContextOverride, task tracker.Task) error {
		var err error
		i, err := instance.Load(c.DirConfig, service, overrides)
		if err != nil {
			return errors.WithStack(err)
		}
		overrides = i.Overrides.Merge(overrides)

		err = i.StopSync(cfg, overrides, task)
		if err != nil {
			return errors.WithStack(err)
		}

		if !cfg.SkipBuild {
			b := builder.New(cfg, overrides)
			err := b.Build(c.DirConfig, task, service)
			if err != nil {
				return errors.WithStack(err)
			}
		}
		err = i.StartAsync(cfg, task)
		if err != nil {
			return errors.WithStack(err)
		}

		return nil
	})
	return nil
}
