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

	Messages() []string

	Child(name string) Task
	Children() []Task
}

type UpdateHandler func(updatedTask Task)

type TaskState int

const (
	TaskStatePending TaskState = iota
	TaskStateInProgress
	TaskStateSuccess
	TaskStateWarning
	TaskStateFailed
)

type task struct {
	name     string
	messages []string
	state    TaskState

	childNames []string
	children   map[string]*task

	startTime time.Time
	endTime   time.Time

	updateHandler UpdateHandler

	mtx *sync.Mutex
}

func NewTask(updateHandler UpdateHandler) Task {
	return &task{
		name:          "",
		children:      make(map[string]*task),
		startTime:     time.Now(),
		mtx:           &sync.Mutex{},
		updateHandler: updateHandler,
	}
}

func (t *task) Name() string {
	return t.name
}

func (t *task) State() TaskState {
	t.mtx.Lock()
	defer t.mtx.Unlock()
	return t.getState()
}

func (t *task) getState() TaskState {
	if len(t.children) == 0 {
		return t.state
	}

	var states = make(map[TaskState]int)

	for _, n := range t.childNames {
		child := t.children[n]
		states[child.getState()]++
	}

	if count, ok := states[TaskStateFailed]; ok && count > 0 {
		return TaskStateFailed
	}
	if count, ok := states[TaskStatePending]; ok {
		if count == len(t.childNames) {
			return TaskStatePending
		}
		return TaskStateInProgress
	}
	if count, ok := states[TaskStateInProgress]; ok && count > 0 {
		return TaskStateInProgress
	}
	if count, ok := states[TaskStateWarning]; ok && count > 0 {
		return TaskStateWarning
	}
	if count, ok := states[TaskStateSuccess]; ok && count == len(t.childNames) {
		return TaskStateSuccess
	}
	return TaskStateInProgress
}

func (t *task) Duration() time.Duration {
	t.mtx.Lock()
	defer t.mtx.Unlock()
	if t.state == TaskStateInProgress || t.state == TaskStatePending {
		return time.Since(t.startTime)
	}
	return t.endTime.Sub(t.startTime)
}

func (t *task) SetState(state TaskState, messages ...string) {
	t.applyState(state, messages...)
	if t.updateHandler != nil {
		t.updateHandler(t)
	}
}

func (t *task) applyState(state TaskState, messages ...string) {
	t.mtx.Lock()
	defer t.mtx.Unlock()
	t.state = state
	t.messages = messages

	if state != TaskStateInProgress && state != TaskStatePending {
		t.endTime = time.Now()
	}
}

func (t *task) Messages() []string {
	t.mtx.Lock()
	defer t.mtx.Unlock()
	return t.messages
}

func (t *task) Child(name string) Task {
	t.mtx.Lock()
	if c, ok := t.children[name]; ok {
		t.mtx.Unlock()
		return c
	}

	t.childNames = append(t.childNames, name)
	t.children[name] = &task{
		name:          name,
		children:      make(map[string]*task),
		updateHandler: t.updateHandler,
		startTime:     time.Now(),
		mtx:           t.mtx,
	}
	t.mtx.Unlock()
	if t.updateHandler != nil {
		t.updateHandler(t.children[name])
	}
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
