package worker

// Pool provides a set of workers for executing functions
type Pool struct {
	jobs       chan func() error
	finished   chan struct{}
	completion chan struct{}
	workers    int
}

// NewPool creates a pool with the specified number of workers.
// The pool will not begin accepting jobs until Start() is called.
func NewPool(workers int) *Pool {
	return &Pool{
		workers: workers,
	}
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
		j()
	}
	p.finished <- struct{}{}
}
