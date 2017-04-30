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

	Updates() <-chan Task
	Close()

	Messages() []string

	Child(name string) Task
	Children() []Task
}

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
	children   map[string]Task

	startTime time.Time
	endTime   time.Time

	updates chan Task
	mtx     sync.Mutex
}

func NewTask() Task {
	return &task{
		name:      "",
		children:  make(map[string]Task),
		updates:   make(chan Task, 2),
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

	var states = make(map[TaskState]int)

	for _, n := range t.childNames {
		child := t.children[n]
		states[child.State()]++
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

func (t *task) Updates() <-chan Task {
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
	if state == t.State() {
		return
	}

	t.mtx.Lock()
	defer func() {
		t.mtx.Unlock()
		t.updates <- t
	}()

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
	var added Task
	t.mtx.Lock()
	defer func() {
		t.mtx.Unlock()
		if added != nil {
			t.updates <- added
		}
	}()

	if c, ok := t.children[name]; ok {
		return c
	}

	t.childNames = append(t.childNames, name)
	t.children[name] = &task{
		name:      name,
		children:  make(map[string]Task),
		updates:   t.updates,
		startTime: time.Now(),
	}
	added = t.children[name]
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
