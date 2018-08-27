package processes

import (
	"fmt"
	"os/exec"
	"strconv"
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

func (p *Processes) KillGroup(pid int, sudo bool) error {
	pgid, err := syscall.Getpgid(pid)
	if err != nil {
		return errors.WithMessage(err, fmt.Sprintf("Could not kill pid %d", pid))
	}

	if pgid == 0 || pgid == 1 {
		return errors.WithStack(errors.New("suspect pgid: " + strconv.Itoa(pgid)))
	}

	flag := "-9"

	cmdName := "kill"
	cmdArgs := []string{}
	if sudo {
		cmdName = "sudo"
		cmdArgs = append(cmdArgs, "kill")
	}
	cmdArgs = append(cmdArgs, flag, fmt.Sprintf("-%v", pgid))
	cmd := exec.Command(cmdName, cmdArgs...)
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	err = cmd.Run()
	if err != nil {
		return errors.WithMessage(err, "signalGroup:")
	}
	return nil
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
