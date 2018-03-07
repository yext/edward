package commandline

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/pkg/errors"
	"github.com/theothertomelliott/gopsutil-nocgo/net"
	"github.com/theothertomelliott/gopsutil-nocgo/process"
	"github.com/yext/edward/commandline"
	"github.com/yext/edward/services"
)

type buildandrun struct {
	Service *services.ServiceConfig
	Backend *Backend
	done    chan struct{}
	cmd     *exec.Cmd

	launchConditionMet chan struct{}

	mtx sync.Mutex
}

var _ services.Builder = &buildandrun{}
var _ services.Runner = &buildandrun{}

func (b *buildandrun) Build(workingDir string, getenv func(string) string) ([]byte, error) {
	cmd, err := commandline.ConstructCommand(workingDir, b.Service.Path, b.Backend.Commands.Build, getenv)
	if err != nil {
		return nil, errors.WithStack(err)
	}
	out, err := cmd.CombinedOutput()
	return out, errors.WithStack(err)
}

func (b *buildandrun) Start(standardLog io.Writer, errorLog io.Writer) error {
	b.mtx.Lock()
	defer b.mtx.Unlock()

	if b.cmd != nil {
		return errors.New("service already started")
	}

	b.done = make(chan struct{})
	b.launchConditionMet = make(chan struct{})

	command, cmdArgs, err := commandline.ParseCommand(os.ExpandEnv(b.Backend.Commands.Launch))
	if err != nil {
		return errors.WithStack(err)
	}
	b.cmd = exec.Command(command, cmdArgs...)
	if b.Service.Path != nil {
		b.cmd.Dir = os.ExpandEnv(*b.Service.Path)
	}
	b.cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	b.cmd.Stdout = b.newWatchingWriter(standardLog)
	b.cmd.Stderr = b.newWatchingWriter(errorLog)

	var started = make(chan struct{})
	var failure error
	go func() {
		err := b.cmd.Start()
		close(started)
		defer close(b.done)
		if err != nil {
			failure = err
			return
		}
		err = b.cmd.Wait()
		if err != nil {
			failure = err
		}
	}()
	<-started

	if failure != nil {
		fmt.Println("Returning failure:", failure)
		return errors.WithStack(failure)
	}

	var live = make(chan error)
	go func() {
		err = b.waitUntilLive()
		if err != nil {
			live <- errors.WithStack(err)
		}
		close(live)
	}()

	select {
	case <-b.done:
		return errors.New("process exited")
	case e := <-live:
		return errors.WithStack(e)
	}
}

func (b *buildandrun) Status() (services.BackendStatus, error) {
	var status services.BackendStatus
	pid := b.cmd.Process.Pid
	if pid != 0 {
		proc, err := process.NewProcess(int32(pid))
		if err != nil {
			return services.BackendStatus{}, errors.WithStack(err)
		}
		status.MemoryInfo, err = proc.MemoryInfo()
		if err != nil {
			return services.BackendStatus{}, errors.WithMessage(err, "retrieving memory info")
		}
		ports, err := doGetPorts(proc)
		if err != nil {
			return services.BackendStatus{}, errors.WithMessage(err, "retrieving port info")
		}
		status.Ports = ports
	}
	return status, nil
}

func doGetPorts(proc *process.Process) ([]string, error) {
	connections, err := net.Connections("all")
	if err != nil {
		return nil, errors.WithStack(err)
	}

	var ports []string
	for _, connection := range connections {
		if connection.Status == "LISTEN" && connection.Pid == proc.Pid {
			ports = append(ports, strconv.Itoa(int(connection.Laddr.Port)))
		}
	}

	children, err := proc.Children()
	// This will error out if the process has finished or has no children
	if err != nil {
		return ports, nil
	}
	for _, child := range children {
		childPorts, err := doGetPorts(child)
		if err == nil {
			ports = append(ports, childPorts...)
		}
	}
	return ports, nil
}

// Wait blocks until this service has stopped.
func (b *buildandrun) Wait() {
	<-b.done
}

func (b *buildandrun) waitForCompletionWithTimeout(timeout time.Duration) bool {
	var completed = make(chan struct{})
	defer close(completed)
	timer := time.NewTimer(timeout)
	defer timer.Stop()

	go func() {
		b.Wait()
		completed <- struct{}{}
	}()

	select {
	case <-completed:
		return true
	case <-timer.C:
		return false
	}
}

func (b *buildandrun) Stop(workingDir string, getenv func(string) string) ([]byte, error) {
	b.mtx.Lock()
	defer b.mtx.Unlock()

	var out []byte
	if b.Backend.Commands.Stop != "" {
		cmd, err := commandline.ConstructCommand(workingDir, b.Service.Path, b.Backend.Commands.Stop, getenv)
		if err != nil {
			return nil, errors.WithStack(err)
		}

		out, err = cmd.CombinedOutput()
		if err != nil {
			return out, errors.WithStack(err)
		}

		if b.waitForCompletionWithTimeout(1 * time.Second) {
			return out, nil
		}
	}

	err := InterruptGroup(b.cmd.Process.Pid, b.Service)
	if err != nil {
		return out, errors.WithStack(err)
	}

	if b.waitForCompletionWithTimeout(2 * time.Second) {
		return out, nil
	}

	err = KillGroup(b.cmd.Process.Pid, b.Service)
	if err != nil {
		return out, errors.WithMessage(err, "Kill failed")
	}

	if b.waitForCompletionWithTimeout(2 * time.Second) {
		return out, nil
	}
	return nil, errors.New("kill did not stop service")

	b.cmd = nil
	return out, nil
}

// InterruptGroup sends an interrupt signal to a process group.
// Will use sudo if required by this service.
func InterruptGroup(pgid int, service *services.ServiceConfig) error {
	return errors.WithStack(signalGroup(pgid, service, "-2"))
}

// KillGroup sends a kill signal to a process group.
// Will use sudo priviledges if required by this service.
func KillGroup(pgid int, service *services.ServiceConfig) error {
	return errors.WithStack(signalGroup(pgid, service, "-9"))
}

func signalGroup(pgid int, service *services.ServiceConfig, flag string) error {
	cmdName := "kill"
	cmdArgs := []string{}
	if service.IsSudo(services.OperationConfig{}) {
		cmdName = "sudo"
		cmdArgs = append(cmdArgs, "kill")
	}
	cmdArgs = append(cmdArgs, flag, fmt.Sprintf("-%v", pgid))
	cmd := exec.Command(cmdName, cmdArgs...)
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	err := cmd.Run()
	return errors.WithMessage(err, "signalGroup:")
}

func (b *buildandrun) newWatchingWriter(w io.Writer) io.Writer {
	return &logWatchingWriter{
		out:          w,
		launchChecks: b.Backend.LaunchChecks,
		found:        b.launchConditionMet,
	}
}

type logWatchingWriter struct {
	out          io.Writer
	launchChecks *LaunchChecks

	received []byte

	found chan struct{}

	mtx sync.Mutex
}

func (pw *logWatchingWriter) Write(p []byte) (n int, err error) {
	pw.mtx.Lock()
	defer pw.mtx.Unlock()

	select {
	case <-pw.found:
		// Don't do anything if the log message has already been found
	default:
		if pw.launchChecks != nil && pw.launchChecks.LogText != "" {
			pw.received = append(pw.received, p...)
			if strings.Contains(string(pw.received), pw.launchChecks.LogText) {
				close(pw.found)
			}
		}
	}

	return pw.out.Write(p)
}
