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

	var updateWait sync.WaitGroup
	updateWait.Add(1)

	go func() {
		writer := uilive.New()
		writer.Start()

		inProgress := NewInProgressRenderer()

		//renderer := tracker.NewAnsiRenderer()
		for updatedTask := range task.Updates() {
			state := updatedTask.State()
			if state != tracker.TaskStatePending &&
				state != tracker.TaskStateInProgress {
				renderer := NewCompletionRenderer(updatedTask)
				renderer.Render(writer, task)
				writer.Stop()

				writer = uilive.New()
				writer.Start()
			}

			inProgress.Render(writer, task)
			writer.Flush()
		}

		warmup.Wait()
		updateWait.Done()
		writer.Stop()
	}()

	defer func() {
		task.Close()
		updateWait.Wait()
	}()

	return errors.WithStack(f())
}
