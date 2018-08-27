package instance

import "syscall"

// Processes provides functions for working with processes
type Processes interface {
	// PidExists returns true iff the process with the provided PID exists.
	PidExists(pid int) (bool, error)

	// PidCommandMatches returns true iff the process with the provided PID exists,
	// and its command contains the provided string.
	PidCommandMatches(pid int, value string) (bool, error)

	// SendSignal issues the specified signal to the process running with the provided PID.
	// If the PID does not exist, no error will be returned.
	SendSignal(pid int, signal syscall.Signal) error
}
