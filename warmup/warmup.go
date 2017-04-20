package warmup

import (
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
			tr := j.tracker.Child("Warmup")
			_, err := http.Get(j.task.URL)
			if err != nil {
				tr.SetState(tracker.TaskStateWarning, err.Error())
				continue
			}
			tr.SetState(tracker.TaskStateSuccess)
		}
	}
	finished <- struct{}{}
}

type job struct {
	task    *Warmup
	tracker tracker.Task
}

// Warmup defines an action to take to "warm up" a service after launch
type Warmup struct {
	// A URL that can be used to warm up this service, will result in a GET request
	URL string `json:"URL,omitempty"`
}

// Run executes a warmup operation for a service
func Run(service string, w *Warmup, tracker tracker.Task) {
	if w == nil {
		return
	}
	t := tracker.Child(service)
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
