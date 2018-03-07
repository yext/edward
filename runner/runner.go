package runner

import (
	"os"
	"os/signal"
	"sync"
	"time"

	"github.com/kylelemons/godebug/pretty"
	"github.com/pkg/errors"
	"github.com/yext/edward/home"
	"github.com/yext/edward/instance"
	"github.com/yext/edward/services"
)

// Runner provides state and functions for running a given service
type Runner struct {
	Service       *services.ServiceConfig
	backendRunner services.Runner
	DirConfig     *home.EdwardConfiguration

	logFile    *os.File
	messageLog *Log

	commandWait sync.WaitGroup
	NoWatch     bool
	WorkingDir  string

	Logger Logger

	status instance.Status

	instanceId string

	standardLog *Log
	errorLog    *Log

	shutdownChan chan struct{}
}

func NewRunner(
	service *services.ServiceConfig,
	dirConfig *home.EdwardConfiguration,
	noWatch bool,
	workingDir string,
	logger Logger,
) (*Runner, error) {
	r := &Runner{
		Service:    service,
		DirConfig:  dirConfig,
		NoWatch:    noWatch,
		WorkingDir: workingDir,
		Logger:     logger,
	}
	var err error
	r.backendRunner, err = services.GetRunner(service)
	if err != nil {
		return nil, errors.WithStack(err)
	}
	return r, nil
}

func (r *Runner) Messagef(format string, a ...interface{}) {
	if r.messageLog != nil {
		r.messageLog.Printf(format, a...)
	}
	if r.Logger != nil {
		r.Logger.Printf(format, a...)
	}
}

func (r *Runner) Run(args []string) error {
	r.updateServiceState(instance.StateStarting)

	// Allow shutdown through signals
	r.configureSignals()

	r.Messagef("Signals configured")

	r.shutdownChan = make(chan struct{})

	r.status = instance.Status{
		StartTime: time.Now(),
	}

	if r.WorkingDir != "" {
		err := os.Chdir(r.WorkingDir)
		if err != nil {
			r.updateServiceState(instance.StateDied)
			return errors.WithStack(err)
		}
	}

	r.Logger.Printf("Service config: %s", pretty.Sprint(r.Service))

	// Set the instance id
	command, err := instance.Load(r.DirConfig, r.Service, services.ContextOverride{})
	if err != nil {
		r.Messagef("Could not get service command: %v\n", err)
	}
	r.instanceId = command.InstanceId

	err = r.configureLogs()
	if err != nil {
		r.updateServiceState(instance.StateDied)
		return errors.WithStack(err)
	}

	statusTick := time.NewTicker(10 * time.Second)
	defer func() {
		if statusTick != nil {
			statusTick.Stop()
		}
	}()
	go func() {
		for _ = range statusTick.C {
			r.updateStatusDetail()
		}
	}()

	r.commandWait.Add(1)

	err = r.startService()
	if err != nil {
		return errors.WithStack(err)
	}

	r.updateServiceState(instance.StateRunning)

	closeWatchers := r.configureWatch()
	if closeWatchers != nil {
		defer closeWatchers()
	}

	r.commandWait.Wait()

	// Wait for shutdown.
	// If the service stopped and an interrupt was not sent, do not set the "DIED" state.
	select {
	case <-r.shutdownChan:
		r.updateServiceState(instance.StateStopped)
		r.Messagef("Service stopped\n")
		return nil
	default:
		r.updateServiceState(instance.StateDied)
		statusTick.Stop()
		statusTick = nil
	}

	return nil
}

func (r *Runner) updateServiceState(newState instance.State) {
	r.status.State = newState
	err := instance.SaveStatusForService(r.Service, r.instanceId, r.status, r.DirConfig.StateDir)
	if err != nil {
		r.Messagef("could not save state: %v", err)
	}
}

func (r *Runner) updateStatusDetail() {
	r.status.StdoutLines = r.standardLog.Len()
	r.status.StderrLines = r.errorLog.Len()

	backendStatus, err := r.backendRunner.Status()
	if err != nil {
		r.Messagef("could not save state: %v", err)
		return
	}
	r.status.Ports = backendStatus.Ports
	r.status.MemoryInfo = backendStatus.MemoryInfo

	dir := r.DirConfig.StateDir
	err = instance.SaveStatusForService(r.Service, r.instanceId, r.status, dir)
	if err != nil {
		r.Messagef("could not save state: %v", err)
	}
}

func (r *Runner) configureLogs() error {
	logLocation := r.Service.GetRunLog(r.DirConfig.LogDir)
	os.Remove(logLocation)

	var err error
	r.logFile, err = os.Create(logLocation)
	if err != nil {
		return errors.WithStack(err)
	}

	r.messageLog = &Log{
		file:   r.logFile,
		name:   r.Service.Name,
		stream: "messages",
	}
	return nil
}

func (r *Runner) configureSignals() {
	signalChan := make(chan os.Signal, 1)
	signal.Notify(signalChan, os.Interrupt)
	go func() {
		for range signalChan {
			r.Messagef("Received interrupt\n")
			err := r.stopService()
			if err != nil {
				r.Messagef("Could not stop service: %v", err)
			}
			close(r.shutdownChan)
		}
	}()
}

func (r *Runner) configureWatch() func() {
	if !r.NoWatch {
		closeWatchers, err := BeginWatch(r.DirConfig, r.Service, r.restartService, r.messageLog)
		if err != nil {
			r.Messagef("Could not enable auto-restart: %v\n", err)
			return nil
		}
		if closeWatchers != nil {
			r.Messagef("Auto-restart enabled. This service will restart when files in its watch directories are edited.\nThis can be disabled using the --no-watch flag.\n")
		}
		return closeWatchers
	}
	return nil
}

func (r *Runner) restartService() error {
	r.Messagef("Restarting service\n")

	// Increment the counter to prevent exiting unexpectedly
	r.commandWait.Add(1)

	err := r.stopService()
	if err != nil {
		return errors.WithStack(err)
	}
	err = r.startService()
	if err != nil {
		return errors.WithStack(err)
	}
	return nil
}

func (r *Runner) stopService() error {
	wd, err := os.Getwd()
	if err != nil {
		return errors.WithStack(err)
	}

	var scriptErr error
	var scriptOutput []byte

	// TODO: Get the right env function from the client
	scriptOutput, scriptErr = r.backendRunner.Stop(wd, nil)
	if scriptErr != nil {
		r.Messagef("Stop failed:\n%v\n", string(scriptOutput))
		return errors.WithStack(err)
	}
	return nil
}

func (r *Runner) startService() error {
	r.Messagef("Service starting\n")

	r.standardLog = &Log{
		file:   r.logFile,
		name:   r.Service.Name,
		stream: "stdout",
	}
	r.errorLog = &Log{
		file:   r.logFile,
		name:   r.Service.Name,
		stream: "stderr",
	}

	err := r.backendRunner.Start(r.standardLog, r.errorLog)
	if err != nil {
		errors.WithStack(err)
	}
	go func() {
		r.backendRunner.Wait()
		r.commandWait.Done()
	}()
	return nil
}
