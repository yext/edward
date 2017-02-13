package runner

import (
	"fmt"
	"log"
	"strings"

	"github.com/pkg/errors"
	"github.com/yext/edward/services"
	fsnotify "gopkg.in/fsnotify.v1"
)

func BeginWatch(service services.ServiceOrGroup, restart func() error, logger Logger) (func(), error) {
	watches, err := service.Watch()
	if err != nil {
		return nil, errors.WithStack(err)
	}
	var watchers []*fsnotify.Watcher
	for _, watch := range watches {
		watcher, err := startWatch(&watch, restart, logger)
		if err != nil {
			return nil, errors.WithStack(err)
		}
		watchers = append(watchers, watcher)
	}

	closeAll := func() {
		for _, watch := range watchers {
			watch.Close()
		}
	}
	return closeAll, nil
}

func startWatch(watches *services.ServiceWatch, restart func() error, logger Logger) (*fsnotify.Watcher, error) {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, errors.WithStack(err)
	}

	go func() {
		for {
			select {
			case event := <-watcher.Events:
				if event.Op&fsnotify.Write == fsnotify.Write {
					logger.Printf("File edited: %v\n", event.Name)

					var wasExcluded bool
					for _, excluded := range watches.ExcludedPaths {
						if strings.HasPrefix(event.Name, excluded) {
							logger.Printf("File is under excluded path: %v\n", excluded)
							wasExcluded = true
							break
						}
					}

					if wasExcluded {
						continue
					}
					fmt.Printf("Rebuilding %v\n", watches.Service.GetName())
					err = rebuildService(watches.Service, restart, logger)
					if err != nil {
						logger.Printf("Could not rebuild %v: %v\n", watches.Service.GetName(), err)
					}
				}

			case err := <-watcher.Errors:
				if err != nil {
					log.Println("error:", err)
				}
			}
		}
	}()

	for _, dir := range watches.IncludedPaths {
		err = watcher.Add(dir)
		if err != nil {
			watcher.Close()
			return nil, errors.WithStack(err)
		}
	}
	return watcher, nil
}

type rebuildTracker struct {
	logger Logger
}

func (t *rebuildTracker) Start() {
	t.logger.Printf("Build starting\n")
}
func (t *rebuildTracker) Success() {
	t.logger.Printf("Build complete\n")
}
func (t *rebuildTracker) SoftFail(err error) {
	t.logger.Printf("Build skipped: %v\n", err)
}
func (t *rebuildTracker) Fail(err error) {
	t.logger.Printf("Build failed: %v\n", err)
}
func (t *rebuildTracker) FailWithOutput(err error, output string) {
	t.logger.Printf("%v\n", output)
	t.logger.Printf("Build failed: %v\n", err)
}

func rebuildService(service *services.ServiceConfig, restart func() error, logger Logger) error {
	command, err := service.GetCommand()
	if err != nil {
		return errors.WithStack(err)
	}
	err = command.BuildWithTracker(true, &rebuildTracker{
		logger: logger,
	})
	if err != nil {
		return errors.New("build failed")
	}
	err = restart()
	if err != nil {
		return errors.WithStack(err)
	}
	return nil
}
