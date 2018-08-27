package processes

import (
	"strings"
	"syscall"

	"github.com/pkg/errors"
	"github.com/theothertomelliott/gopsutil-nocgo/process"
)

type Processes struct {
}

func (p *Processes) SendSignal(pid int, signal syscall.Signal) error {
	pr, err := process.NewProcess(int32(pid))
	if err != nil {
		return errors.WithStack(err)
	}
	err = pr.SendSignal(signal)
	return errors.WithStack(err)
}

func (p *Processes) PidExists(pid int) (bool, error) {
	exists, err := process.PidExists(int32(pid))
	return exists, errors.WithStack(err)

}

func (p *Processes) PidCommandMatches(pid int, value string) (bool, error) {
	if exists, err := process.PidExists(int32(pid)); !exists || err != nil {
		return false, errors.WithStack(err)
	}
	proc, err := process.NewProcess(int32(pid))
	if err != nil {
		return false, errors.WithStack(err)
	}
	cmdline, err := proc.Cmdline()
	if err != nil {
		return false, errors.WithStack(err)
	}
	return strings.Contains(cmdline, value), nil
}
