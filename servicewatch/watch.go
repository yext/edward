package servicewatch

import (
	"fmt"
	"log"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"github.com/juju/errgo"
	"github.com/yext/edward/services"
	fsnotify "gopkg.in/fsnotify.v1"
)

func Begin(sgs []services.ServiceOrGroup) error {
	if len(sgs) == 0 {
		return errgo.New("no services")
	}

	hasWatch := false

	for _, s := range sgs {
		watches, err := s.Watch()
		if err != nil {
			return err
		}
		for _, watch := range watches {
			watcher, err := startWatch(&watch)
			if err != nil {
				return err
			}
			defer watcher.Close()
			hasWatch = true
		}
	}

	if !hasWatch {
		fmt.Println("No services configured for watching")
		return nil
	}

	sigs := make(chan os.Signal, 1)
	done := make(chan bool, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		_ = <-sigs
		done <- true
	}()

	<-done
	return nil
}

func startWatch(watches *services.ServiceWatch) (*fsnotify.Watcher, error) {
	fmt.Printf("Watching %v paths for service %v\n", len(watches.IncludedPaths), watches.Service.GetName())
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, errgo.Mask(err)
	}

	go func() {
		for {
			select {
			case event := <-watcher.Events:
				if event.Op&fsnotify.Write == fsnotify.Write {
					fmt.Printf("File edited: %v\n", event.Name)

					var wasExcluded bool
					for _, excluded := range watches.ExcludedPaths {
						if strings.HasPrefix(event.Name, excluded) {
							fmt.Println("File is under excluded path:", excluded)
							wasExcluded = true
							break
						}
					}

					if wasExcluded {
						continue
					}
					fmt.Printf("Rebuilding %v\n", watches.Service.GetName())
					err = rebuildService(watches.Service)
					if err != nil {
						fmt.Printf("Could not rebuild %v: %v\n", watches.Service.GetName(), err)
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
			return nil, errgo.Mask(err)
		}
	}
	return watcher, nil
}

func rebuildService(service *services.ServiceConfig) error {
	command, err := service.GetCommand()
	if err != nil {
		return errgo.Mask(err)
	}
	err = command.BuildSync(true)
	if err != nil {
		return errgo.Mask(err)
	}
	err = service.Stop()
	if err != nil {
		return errgo.Mask(err)
	}
	err = service.Start()
	if err != nil {
		return errgo.Mask(err)
	}
	return nil
}
