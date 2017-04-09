package tracker

import (
	"bytes"
	"fmt"
	"time"

	"github.com/fatih/color"
)

// Job provides an interface to follow a job within an operation
type Job interface {
	State(state string)

	Success(state string)
	Warning(message string)
	Fail(message string, extra ...string)

	Render() string
	Done() bool
	Failed() bool
}

type jobState int

const jobStateInProgress jobState = 0
const jobStateSuccess jobState = 1
const jobStateWarning jobState = 2
const jobStateFailed jobState = 3

type simpleJob struct {
	update chan struct{}

	name string

	stateMessage string
	state        jobState
	extra        []string

	startTime time.Time
	endTime   time.Time

	// Simplified rendering for test
	testRender bool
}

func newJob(name string, update chan struct{}) Job {
	return &simpleJob{
		name:      name,
		update:    update,
		state:     jobStateInProgress,
		startTime: time.Now(),
	}
}

func (s *simpleJob) State(state string) {
	s.state = jobStateInProgress
	s.setState(state)
}

func (s *simpleJob) setState(state string) {
	s.stateMessage = state
	s.update <- struct{}{}
}

func (s *simpleJob) Success(state string) {
	s.state = jobStateSuccess
	s.endTime = time.Now()
	s.setState(state)
}

func (s *simpleJob) Warning(message string) {
	s.state = jobStateWarning
	s.endTime = time.Now()
	s.setState(message)
}

func (s *simpleJob) Fail(message string, extra ...string) {
	s.state = jobStateFailed
	s.extra = extra
	s.endTime = time.Now()
	s.setState(message)
}

func (s *simpleJob) Render() string {
	if s.testRender {
		return fmt.Sprintf("%v: [%v]", s.name, s.stateMessage)
	}

	var b bytes.Buffer
	tmpOutput := color.Output
	defer func() {
		color.Output = tmpOutput
	}()
	color.Output = &b
	fmt.Fprintf(&b, "%-30s", s.name+"...")
	fmt.Fprint(&b, "[")
	s.setStateColor()
	fmt.Fprint(&b, s.stateMessage)
	color.Unset()
	fmt.Fprintf(&b, "]")
	if s.Done() {
		fmt.Fprintf(&b, " (%v)", autoRoundTime(s.endTime.Sub(s.startTime)))
	}
	fmt.Fprintf(&b, "%-10s", " ")
	return string(b.Bytes())
}

func (s *simpleJob) setStateColor() {
	switch s.state {
	case jobStateFailed:
		color.Set(color.FgRed)
	case jobStateWarning:
		color.Set(color.FgYellow)
	case jobStateSuccess:
		color.Set(color.FgGreen)
	default:
		color.Set(color.FgWhite)
	}
}

func (s *simpleJob) Done() bool {
	return s.state != jobStateInProgress
}
func (s *simpleJob) Failed() bool {
	return s.state == jobStateFailed
}

func autoRoundTime(d time.Duration) time.Duration {
	if d > time.Hour {
		return roundTime(d, time.Second)
	}
	if d > time.Minute {
		return roundTime(d, time.Second)
	}
	if d > time.Second {
		return roundTime(d, time.Millisecond)
	}
	if d > time.Millisecond {
		return roundTime(d, time.Microsecond)
	}
	return d
}

// Based on the example at https://play.golang.org/p/QHocTHl8iR
func roundTime(d, r time.Duration) time.Duration {
	if r <= 0 {
		return d
	}
	neg := d < 0
	if neg {
		d = -d
	}
	if m := d % r; m+m < r {
		d = d - m
	} else {
		d = d + r - m
	}
	if neg {
		return -d
	}
	return d
}
