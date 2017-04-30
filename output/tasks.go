package output

import (
	"sync"
	"time"

	"github.com/gosuri/uilive"
	"github.com/pkg/errors"
	"github.com/yext/edward/tracker"
)

func FollowTask(f func(task tracker.Task) error) error {
	uilive.RefreshInterval = time.Hour

	var updateWait sync.WaitGroup
	updateWait.Add(1)

	follower, rootTask := newFollowedTask()
	defer follower.done()
	return errors.WithStack(f(rootTask))
}

type follower struct {
	rootTask   tracker.Task
	inProgress *InProgressRenderer
	writer     *uilive.Writer
}

func newFollowedTask() (*follower, tracker.Task) {
	writer := uilive.New()
	writer.Start()
	f := &follower{
		inProgress: NewInProgressRenderer(),
		writer:     writer,
	}
	task := tracker.NewTask(f.handle)
	f.rootTask = task
	return f, task
}

func (f *follower) handle(update tracker.Task) {
	state := update.State()
	if state != tracker.TaskStatePending &&
		state != tracker.TaskStateInProgress {
		renderer := NewCompletionRenderer(update)
		renderer.Render(f.writer, f.rootTask)
		f.writer.Stop()

		f.writer = uilive.New()
		f.writer.Start()
	}

	f.inProgress.Render(f.writer, f.rootTask)
	f.writer.Flush()
}

func (f *follower) done() {
	f.writer.Stop()
}
