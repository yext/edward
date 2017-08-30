package worker

import (
	"sync"

	"github.com/pkg/errors"
)

// Pool provides a set of workers for executing functions
type Pool struct {
	jobs       chan func() error
	finished   chan struct{}
	completion chan struct{}
	workers    int

	err error
	mtx sync.Mutex
}

// NewPool creates a pool with the specified number of workers.
// The pool will not begin accepting jobs until Start() is called.
func NewPool(workers int) *Pool {
	return &Pool{
		workers: workers,
	}
}

func (p *Pool) Err() error {
	p.mtx.Lock()
	defer p.mtx.Unlock()

	return p.err
}

func (p *Pool) setErr(err error) {
	p.mtx.Lock()
	defer p.mtx.Unlock()
	p.err = err
}

// Start initializes the Pool to begin accepting and running jobs.
func (p *Pool) Start() {
	p.jobs = make(chan func() error)
	p.finished = make(chan struct{})
	p.completion = make(chan struct{})

	for w := 1; w <= p.workers; w++ {
		go p.worker(w)
	}

	go func() {
		for a := 1; a <= p.workers; a++ {
			<-p.finished
		}
		close(p.finished)
		close(p.completion)
	}()
}

// Enqueue adds a job to the pool.
func (p *Pool) Enqueue(job func() error) error {
	// Execute the job synchronously if no workers are provided
	if p.workers == 0 {
		return errors.WithStack(job())
	}

	if p.Err() != nil {
		return errors.WithStack(p.Err())
	}

	p.jobs <- job
	return nil
}

// Stop prevents this pool from accepting new jobs.
func (p *Pool) Stop() {
	close(p.jobs)
}

// Complete returns a channel that will be closed when all workers have finished
func (p *Pool) Complete() <-chan struct{} {
	return p.completion
}

func (p *Pool) worker(index int) {
	for j := range p.jobs {
		err := j()
		if err != nil {
			p.setErr(err)
		}
	}
	p.finished <- struct{}{}
}
