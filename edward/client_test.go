package edward_test

import (
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"path"
	"strings"
	"testing"

	"github.com/pkg/errors"
	"github.com/theothertomelliott/gopsutil-nocgo/process"
	"github.com/yext/edward/tracker"
)

// Path to the Edward executable as built
var edwardExecutable string

func TestMain(m *testing.M) {
	buildDir, err := ioutil.TempDir("", "edwardTest")
	if err != nil {
		log.Fatal(err)
	}
	defer os.RemoveAll(buildDir)

	edwardExecutable = path.Join(buildDir, "edward")

	cmd := exec.Command("go", "build", "-o", edwardExecutable, "github.com/yext/edward")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	err = cmd.Run()
	if err != nil {
		log.Fatal(err)
	}

	os.Exit(m.Run())
}

type testFollower struct {
	states   map[string]string
	messages []string
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
	f.messages = append(f.messages, update.Messages()...)
}
func (f *testFollower) Done() {}

// getRunnerAndServiceProcesses returns all processes and children spawned by this test
func getRunnerAndServiceProcesses(t *testing.T) []*process.Process {
	var processes []*process.Process
	testProcess, err := process.NewProcess(int32(os.Getpid()))
	if err != nil {
		t.Fatal(err)
	}
	runners, err := testProcess.Children()
	if err != nil {
		t.Fatalf("No processes found")
	}
	processes = append(processes, runners...)
	for _, runner := range runners {
		services, err := runner.Children()
		if err != nil {
			t.Fatalf("No processes found")
		}
		processes = append(processes, services...)
	}
	return processes
}

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
		if serviceCount != 0 {
			t.Fatalf("No processes found, expected %d", serviceCount)
		}
	}
	if len(children) != serviceCount {
		// We can't know which test or operation this would be for, so don't try to stop anything
		var childNames []string
		for _, child := range children {
			cmdline, err := child.Cmdline()
			if err != nil {
				t.Errorf("Error getting cmdline: %v", err)
			}
			childNames = append(childNames, cmdline)
		}
		t.Fatalf("Expected %d children, got %s", serviceCount, strings.Join(childNames, ", "))
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
