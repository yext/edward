package warmup

import (
	"fmt"
	"net/http"
)

var jobs chan *Warmup
var finished chan struct{}

const workers = 3

func init() {
	jobs = make(chan *Warmup)
	finished = make(chan struct{})

	for w := 1; w <= workers; w++ {
		go worker(w, jobs, finished)
	}
}

func worker(id int, jobs <-chan *Warmup, finished chan<- struct{}) {
	for w := range jobs {
		if w.URL != "" {
			_, err := http.Get(w.URL)
			if err != nil {
				fmt.Println("[", id, "] Warmup error: ", err)
			}
		}
	}
	finished <- struct{}{}
}

// Warmup defines an action to take to "warm up" a service after launch
type Warmup struct {
	// A URL that can be used to warm up this service, will result in a GET request
	URL string `json:"URL,omitempty"`
}

// Run executes a warmup operation for a service
func Run(service string, w *Warmup) {
	if w == nil {
		return
	}
	if w.URL != "" {
		fmt.Printf("Warming up %v: %v\n", service, w.URL)
	}
	jobs <- w
}

// Wait blocks until all Warmup operations are complete
func Wait() {
	close(jobs)
	for a := 1; a <= workers; a++ {
		<-finished
	}
}
