package tracker

import (
	"sync"
	"time"
)

type Task interface {
	Name() string

	State() TaskState
	SetState(TaskState, ...string)

	Duration() time.Duration

	Updates() <-chan struct{}
	Close()

	Messages() []string

	Child(name string) Task
	Children() []Task
}

type TaskState int

const (
	TaskStateInProgress TaskState = iota
	TaskStateSuccess
	TaskStateWarning
	TaskStateFailed
)

type task struct {
	name     string
	messages []string
	state    TaskState

	childNames []string
	children   map[string]Task

	startTime time.Time
	endTime   time.Time

	updates chan struct{}
	mtx     sync.Mutex
}

func NewTask() Task {
	return &task{
		name:      "",
		children:  make(map[string]Task),
		updates:   make(chan struct{}, 2),
		startTime: time.Now(),
	}
}

func (t *task) Name() string {
	t.mtx.Lock()
	defer t.mtx.Unlock()
	return t.name
}

func (t *task) State() TaskState {
	t.mtx.Lock()
	defer t.mtx.Unlock()

	if len(t.children) == 0 {
		return t.state
	}

	var state = TaskStateSuccess
	for _, n := range t.childNames {
		child := t.children[n]
		switch child.State() {
		case TaskStateFailed:
			return TaskStateFailed
		case TaskStateInProgress:
			state = TaskStateInProgress
		case TaskStateWarning:
			if state != TaskStateInProgress {
				state = TaskStateWarning
			}
		}
	}
	return state
}

func (t *task) Duration() time.Duration {
	t.mtx.Lock()
	defer t.mtx.Unlock()
	if t.state == TaskStateInProgress {
		return time.Since(t.startTime)
	}
	return t.endTime.Sub(t.startTime)
}

func (t *task) Updates() <-chan struct{} {
	t.mtx.Lock()
	defer t.mtx.Unlock()
	return t.updates
}

func (t *task) Close() {
	t.mtx.Lock()
	defer t.mtx.Unlock()
	close(t.updates)
}

func (t *task) SetState(state TaskState, messages ...string) {
	t.mtx.Lock()
	defer func() {
		t.mtx.Unlock()
		t.updates <- struct{}{}
	}()

	t.state = state
	t.messages = messages

	if state != TaskStateInProgress {
		t.endTime = time.Now()
	}
}

func (t *task) Messages() []string {
	t.mtx.Lock()
	defer t.mtx.Unlock()

	return t.messages
}

func (t *task) Child(name string) Task {
	var added bool
	t.mtx.Lock()
	defer func() {
		t.mtx.Unlock()
		if added {
			t.updates <- struct{}{}
		}
	}()

	if c, ok := t.children[name]; ok {
		return c
	}

	t.childNames = append(t.childNames, name)
	t.children[name] = &task{
		name:     name,
		children: make(map[string]Task),
		updates:  t.updates,
	}
	added = true
	return t.children[name]
}

func (t *task) Children() []Task {
	t.mtx.Lock()
	defer t.mtx.Unlock()

	var children []Task
	for _, c := range t.childNames {
		children = append(children, t.children[c])
	}
	return children
}
