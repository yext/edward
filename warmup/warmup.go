package warmup

import (
	"fmt"
	"net/http"

	"github.com/yext/edward/tracker"
)

var jobs chan *job
var finished chan struct{}

const workers = 3

func init() {
	jobs = make(chan *job)
	finished = make(chan struct{})

	for w := 1; w <= workers; w++ {
		go worker(w, jobs, finished)
	}
}

func worker(id int, jobs <-chan *job, finished chan<- struct{}) {
	for j := range jobs {
		if j.task != nil && j.task.URL != "" {
			j.tracker.State("Running")
			_, err := http.Get(j.task.URL)
			if err != nil {
				j.tracker.Warning(err.Error())
				continue
			}
			j.tracker.Success("Done")
		}
	}
	finished <- struct{}{}
}

type job struct {
	task    *Warmup
	tracker tracker.Job
}

// Warmup defines an action to take to "warm up" a service after launch
type Warmup struct {
	// A URL that can be used to warm up this service, will result in a GET request
	URL string `json:"URL,omitempty"`
}

// Run executes a warmup operation for a service
func Run(service string, w *Warmup, tracker tracker.Operation) {
	if w == nil {
		return
	}
	t := tracker.GetJob(fmt.Sprintf("%v warmup", service))
	jobs <- &job{
		task:    w,
		tracker: t,
	}
}

// Wait blocks until all Warmup operations are complete
func Wait() {
	close(jobs)
	for a := 1; a <= workers; a++ {
		<-finished
	}
}
