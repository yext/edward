package edward_test

import (
	"fmt"
	"os"
	"strings"
	"testing"

	"github.com/pkg/errors"
	"github.com/shirou/gopsutil/process"
	"github.com/yext/edward/tracker"
)

type testFollower struct {
	states map[string]string
}

func newTestFollower() *testFollower {
	return &testFollower{
		states: make(map[string]string),
	}
}

func (f *testFollower) Handle(update tracker.Task) {
	var names []string
	for _, task := range update.Lineage() {
		if task.Name() != "" {
			names = append(names, task.Name())
		}
	}

	fullName := strings.Join(names, " > ")
	f.states[fullName] = update.State().String()

	fmt.Printf("%v - %v\n", fullName, update.State())
}
func (f *testFollower) Done() {}

// verifyAndStopRunners expects that there will be the specified number of runners in progress,
// and that the runners are behaving as expected (exactly one child service, etc).
// Once verified, it will kill the runners and their child services.
func verifyAndStopRunners(t *testing.T, serviceCount int) {
	testProcess, err := process.NewProcess(int32(os.Getpid()))
	if err != nil {
		t.Fatal(err)
	}
	children, err := testProcess.Children()
	if err != nil {
		t.Fatal(err)
	}
	if len(children) != serviceCount {
		// We can't know which test or operation this would be for, so don't try to stop anything
		t.Fatalf("Expected 1 child, got %d", len(children))
	}
	for _, child := range children {
		err = verifyAndStopRunner(t, child)
		if err != nil {
			t.Fatal(err)
		}
	}
}

// verifyAndStopRunner will check that a runner process has exactly one child service,
// and then kill the service, expecting the runner to die.
func verifyAndStopRunner(t *testing.T, runner *process.Process) error {
	defer func() {
		if running, _ := runner.IsRunning(); running {
			return
		}
		t.Error("Expected stopping children to kill runner process")
		err := runner.Kill()
		if err != nil {
			t.Fatal("Could not kill runner:", err)
		}
	}()

	cmdline, err := runner.CmdlineSlice()
	if err != nil {
		return errors.WithStack(err)
	}
	if cmdline[0] == "edward" || cmdline[1] == "run" {
		services, err := runner.Children()
		if err != nil {
			return errors.WithStack(err)
		}
		if len(services) != 1 {
			t.Errorf("Expected 1 child, got %v", len(services))
		}
		for _, service := range services {
			err = service.Kill()
			if err != nil {
				return errors.WithStack(err)
			}
		}
	} else {
		t.Errorf("Expected an edward run command, got: %v", cmdline)
	}
	return nil
}
