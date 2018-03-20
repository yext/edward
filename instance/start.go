package instance

import (
	"fmt"
	"os"

	"github.com/pkg/errors"
	"github.com/yext/edward/home"
	"github.com/yext/edward/services"
	"github.com/yext/edward/tracker"
	"github.com/yext/edward/warmup"
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

// StartAsync starts the service in the background
// Will block until the service is known to have started successfully.
// If the service fails to launch, an error will be returned.
func (c *Instance) StartAsync(cfg services.OperationConfig, task tracker.Task) error {
	if !c.Service.Backend().HasLaunchStep() {
		return nil
	}

	startTask := task.Child(c.Service.GetName()).Child("Start")
	startTask.SetState(tracker.TaskStateInProgress)

	if c.Pid != 0 {
		startTask.SetState(tracker.TaskStateWarning, "Already running")
		return nil
	}

	// Clear previously existing statuses to avoid premature STOPPED state.
	err := DeleteAllStatusesForService(c.Service, c.dirConfig.StateDir)
	if err != nil {
		return errors.WithStack(err)
	}

	os.Remove(c.Service.GetRunLog(c.dirConfig.LogDir))

	cmd, err := c.getLaunchCommand(cfg)
	if err != nil {
		startTask.SetState(tracker.TaskStateFailed, err.Error())
		return errors.WithStack(err)
	}
	cmd.Env = append(os.Environ(), c.Overrides.Env...)
	cmd.Env = append(cmd.Env, c.Service.Env...)

	err = cmd.Start()
	if err != nil {
		startTask.SetState(tracker.TaskStateFailed)
		return errors.WithStack(err)
	}

	c.Pid = cmd.Process.Pid

	c.printf("%v has PID: %d.\n", c.Service.Name, c.Pid)

	err = c.save()
	if err != nil {
		startTask.SetState(tracker.TaskStateFailed)
		return errors.WithStack(err)
	}

	err = WaitUntilRunning(c.dirConfig, cmd, c.Service)
	if err == nil {
		startTask.SetState(tracker.TaskStateSuccess)
		warmup.Run(c.Service.Name, c.Service.Warmup, task)
		return nil
	}
	c.printf("%v failed to start: %s", c.Service.Name, err)

	log, readingErr := logToStringSlice(c.Service.GetRunLog(c.dirConfig.LogDir))
	if readingErr != nil {
		startTask.SetState(tracker.TaskStateFailed, "Could not read log", readingErr.Error(), fmt.Sprint("Original error: ", err.Error()))
	} else {
		log = append(log, err.Error())
		startTask.SetState(tracker.TaskStateFailed, log...)
	}
	stopErr := c.StopSync(cfg, c.Overrides, task.Child("Cleanup"))
	if stopErr != nil {
		return errors.WithStack(stopErr)
	}
	return errors.WithStack(err)
}
