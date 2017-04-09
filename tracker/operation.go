package tracker

import (
	"fmt"
	"strings"
	"sync"
)

// Operation provides a means to track the progress of operations on a set of services
type Operation interface {
	GetJob(name string) Job
	GetOperation(name string) Operation
	StateUpdate() <-chan struct{}
	Render() string
	Failed() bool
	Done() bool
	Close()
}

type op struct {
	jobNames   []string
	jobs       map[string]Job
	operations map[string]Operation
	updates    chan struct{}
	mtx        sync.Mutex
}

func NewOperation() Operation {
	return &op{
		jobs:       make(map[string]Job),
		operations: make(map[string]Operation),
		updates:    make(chan struct{}, 2),
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

func (o *op) GetOperation(name string) Operation {
	if o == nil {
		return nil
	}

	o.mtx.Lock()
	defer o.mtx.Unlock()

	if operation, ok := o.operations[name]; ok {
		return operation
	}

	o.jobNames = append(o.jobNames, name)
	o.operations[name] = &op{
		jobs:       make(map[string]Job),
		operations: make(map[string]Operation),
		updates:    o.updates,
	}
	return o.operations[name]
}

func (o *op) StateUpdate() <-chan struct{} {
	return o.updates
}

func (o *op) Render() string {
	o.mtx.Lock()
	defer o.mtx.Unlock()

	var lines []string
	for _, name := range o.jobNames {
		if job, ok := o.jobs[name]; ok {
			lines = append(lines, job.Render())
			if job.Failed() {
				break
			}
		}
		if operation, ok := o.operations[name]; ok {
			lines = append(lines, fmt.Sprintf("%v:", name))
			for _, jobLine := range strings.Split(operation.Render(), "\n") {
				lines = append(lines, fmt.Sprintf("  %v", jobLine))
			}
			if operation.Failed() {
				break
			}
		}
	}
	return strings.Join(lines, "\n")
}

func (o *op) Done() bool {
	o.mtx.Lock()
	defer o.mtx.Unlock()

	allDone := true
	for _, name := range o.jobNames {
		if job, ok := o.jobs[name]; ok {
			if job.Failed() {
				return true
			}
			if !job.Done() {
				allDone = false
			}
		}
		if operation, ok := o.operations[name]; ok {
			if operation.Failed() {
				return true
			}
			if !operation.Done() {
				allDone = false
			}
		}
	}
	return allDone
}

func (o *op) Failed() bool {
	o.mtx.Lock()
	defer o.mtx.Unlock()

	for _, name := range o.jobNames {
		if job, ok := o.jobs[name]; ok {
			if job.Failed() {
				return true
			}
		}
		if operation, ok := o.operations[name]; ok {
			if operation.Failed() {
				return true
			}
		}
	}
	return false
}

func (o *op) Close() {
	o.mtx.Lock()
	defer o.mtx.Unlock()
	close(o.updates)
}
