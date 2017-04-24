package output

import (
	"sync"
	"time"

	"github.com/gosuri/uilive"
	"github.com/pkg/errors"
	"github.com/yext/edward/tracker"
	"github.com/yext/edward/warmup"
)

func FollowTask(task tracker.Task, f func() error) error {
	uilive.RefreshInterval = time.Hour
	writer := uilive.New()
	writer.Start()

	var updateWait sync.WaitGroup
	updateWait.Add(1)

	go func() {
		renderer := tracker.NewAnsiRenderer()
		for _ = range task.Updates() {
			renderer.Render(writer, task)
			writer.Flush()
		}
		warmup.Wait()
		updateWait.Done()
	}()

	defer func() {
		task.Close()
		updateWait.Wait()
		writer.Stop()
	}()

	return errors.WithStack(f())
}
