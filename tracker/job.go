package tracker

import "fmt"

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
}

func newJob(name string, update chan struct{}) Job {
	return &simpleJob{
		name:   name,
		update: update,
		state:  jobStateInProgress,
	}
}

func (s *simpleJob) State(state string) {
	s.stateMessage = state
	s.update <- struct{}{}
}

func (s *simpleJob) Success(state string) {
	s.state = jobStateSuccess
	s.State(state)
}

func (s *simpleJob) Warning(message string) {
	s.state = jobStateWarning
	s.State(message)
}

func (s *simpleJob) Fail(message string, extra ...string) {
	s.state = jobStateFailed
	s.extra = extra
	s.State(message)
}

func (s *simpleJob) Render() string {
	return fmt.Sprintf("%v: [%v]", s.name, s.stateMessage)
}
func (s *simpleJob) Done() bool {
	return s.state != jobStateInProgress
}
func (s *simpleJob) Failed() bool {
	return s.state == jobStateFailed
}
