package edward_test

import (
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"strings"
	"testing"

	"github.com/theothertomelliott/gopsutil-nocgo/process"
	"github.com/yext/edward/edward"
	"github.com/yext/edward/home"
	"github.com/yext/edward/services"
	"github.com/yext/edward/services/backends/commandline"
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

	// Register necessary backends
	services.RegisterLegacyMarshaler(&commandline.LegacyUnmarshaler{})
	services.RegisterBackend(&commandline.Loader{})

	os.Exit(m.Run())
}

type testFollower struct {
	states     map[string]string
	stateOrder []string
	messages   []string
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
	if len(f.stateOrder) == 0 || f.stateOrder[len(f.stateOrder)-1] != fullName {
		f.stateOrder = append(f.stateOrder, fullName)
	}
	f.messages = append(f.messages, update.Messages()...)
}
func (f *testFollower) Done() {}

// getRunnerAndServiceProcesses returns all processes and children spawned by this test
func getRunnerAndServiceProcesses(t *testing.T) []*process.Process {
	var processes []*process.Process
	testProcess, err := process.NewProcess(int32(os.Getpid()))
	if err != nil {
		t.Error(err)
		return nil
	}
	runners, err := testProcess.Children()
	if err != nil {
		t.Errorf("No processes found: %v", err)
		return nil
	}
	processes = append(processes, runners...)
	for _, runner := range runners {
		services, err := runner.Children()
		if err != nil {
			t.Errorf("No processes found: %v", err)
			return nil
		}
		processes = append(processes, services...)
	}
	return processes
}

// verifyAndStopRunners expects that there will be the specified number of runners in progress,
// and that the runners are behaving as expected (exactly one child service, etc).
// Once verified, it will kill the runners and their child services.
func verifyAndStopRunners(t *testing.T, client *edward.Client, serviceCount int) {
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
	var verifiedChildren []*process.Process
	for _, child := range children {
		verified, err := verifyAndStopRunner(t, client, child)
		if err != nil {
			t.Fatal(err)
		}
		if verified {
			verifiedChildren = append(verifiedChildren, child)
		}
	}
	if len(verifiedChildren) != serviceCount {
		var cmdLines []string
		for _, p := range verifiedChildren {
			cmd, err := p.Cmdline()
			if err != nil {
				cmd = err.Error()
			}
			cmdLines = append(cmdLines, cmd)
		}
		t.Errorf("Expected %d tagged runners, got [%s]", serviceCount, strings.Join(cmdLines, ", "))
	}
}

// verifyAndStopRunner will check that a runner process has exactly one child service,
// and then kill the service, expecting the runner to die.
func verifyAndStopRunner(t *testing.T, client *edward.Client, runner *process.Process) (bool, error) {
	cmdline, err := runner.CmdlineSlice()
	if err != nil {
		t.Logf("error getting command line, ignoring: %v", err)
		return false, nil
	}
	if len(cmdline) > 2 && strings.HasSuffix(cmdline[0], "edward") && cmdline[1] == "run" {
		fullCmd := strings.Join(cmdline, " ")
		for _, tag := range client.Tags {
			if !strings.Contains(fullCmd, fmt.Sprintf("--tag %s", tag)) {
				t.Logf("Missing tag: %v", tag)
				return false, nil
			}
		}
		err := runner.Kill()
		if err != nil {
			t.Fatal("Could not kill runner:", err)
		}
		return true, nil
	}
	return false, nil
}

func showLogsIfFailed(t *testing.T, name string, wd string, client *edward.Client) {
	if !t.Failed() {
		return
	}
	dirConfig := &home.EdwardConfiguration{}
	err := dirConfig.InitializeWithDir(path.Join(wd, "edwardHome"))
	if err != nil {
		t.Fatal(err)
	}
	logFile := filepath.Join(dirConfig.EdwardLogDir, "edward.log")
	b, err := ioutil.ReadFile(logFile)
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("=== Log (%s) <%s> ===\n%s=== /Log ===\n", name, logFile, string(b))
}
