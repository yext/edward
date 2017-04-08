package tracker

import (
	"strings"
	"sync"
)

// Operation provides a means to track the progress of operations on a set of services
type Operation interface {
	GetJob(name string) Job
	StateUpdate() <-chan struct{}
	RenderState() string
	Done() bool
	Close()
}

type op struct {
	jobNames []string
	jobs     map[string]Job
	updates  chan struct{}
	mtx      sync.Mutex
}

func NewOperation() Operation {
	return &op{
		jobs:    make(map[string]Job),
		updates: make(chan struct{}, 2),
	}
}

func (o *op) GetJob(name string) Job {
	o.mtx.Lock()
	defer o.mtx.Unlock()

	if job, ok := o.jobs[name]; ok {
		return job
	}

	o.jobNames = append(o.jobNames, name)
	o.jobs[name] = newJob(name, o.updates)
	return o.jobs[name]
}

func (o *op) StateUpdate() <-chan struct{} {
	return o.updates
}

func (o *op) RenderState() string {
	o.mtx.Lock()
	defer o.mtx.Unlock()

	var lines []string
	for _, name := range o.jobNames {
		lines = append(lines, o.jobs[name].Render())
		if o.jobs[name].Failed() {
			break
		}
	}
	return strings.Join(lines, "\n")
}

func (o *op) Done() bool {
	o.mtx.Lock()
	defer o.mtx.Unlock()

	allDone := true
	for _, name := range o.jobNames {
		if o.jobs[name].Failed() {
			return true
		}
		if !o.jobs[name].Done() {
			allDone = false
		}
	}
	return allDone
}

func (o *op) Close() {
	o.mtx.Lock()
	defer o.mtx.Unlock()
	close(o.updates)
}
