package warmup

import (
	"net/http"

	"github.com/yext/edward/tracker"
)

// Warmup defines an action to take to "warm up" a service after launch
type Warmup struct {
	// A URL that can be used to warm up this service, will result in a GET request
	URL string `json:"URL,omitempty"`
}

// Run executes a warmup operation for a service
func Run(service string, w *Warmup, tr tracker.Task) {
	if w == nil {
		return
	}
	t := tr.Child(service)
	t = t.Child("Warmup")
	t.SetState(tracker.TaskStateInProgress)
	_, err := http.Get(w.URL)
	if err != nil {
		t.SetState(tracker.TaskStateWarning, err.Error())
		return
	}
	t.SetState(tracker.TaskStateSuccess)
}
