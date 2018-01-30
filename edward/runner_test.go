package edward_test

import (
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path"
	"testing"
	"time"
)

func TestLogOutput(t *testing.T) {
	runnerTest(t, func(wd string, success chan struct{}) {
		homeDir := path.Join(wd, "edward_home")
		expectedLog := path.Join(homeDir, "edward_logs")
		expectedLog = path.Join(expectedLog, "edward.log")
		fmt.Println("Expecting log file at:", expectedLog)
		for true {
			if _, err := os.Stat(expectedLog); !os.IsNotExist(err) {
				success <- struct{}{}
				return
			}
			time.Sleep(time.Millisecond)
		}
	})
}

func TestStatusFile(t *testing.T) {
	runnerTest(t, func(wd string, success chan struct{}) {
		homeDir := path.Join(wd, "edward_home")
		statusDir := path.Join(homeDir, "stateFiles")
		fmt.Println("Expecting status file under:", statusDir)
		for true {
			files, _ := ioutil.ReadDir(statusDir)
			if len(files) > 0 {
				success <- struct{}{}
				return
			}
			time.Sleep(time.Millisecond)
		}
	})
}

func runnerTest(t *testing.T, f func(string, chan struct{})) {
	wd, cleanup, err := createWorkingDir("TestLogOutput", "testdata/buildless")
	defer cleanup()
	if err != nil {
		t.Fatal(err)
	}
	homeDir := path.Join(wd, "edward_home")

	cmd := exec.Command(edwardExecutable,
		"--edward_home", homeDir,
		"-c", path.Join(wd, "edward.json"),
		"run",
		"service",
	)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	err = cmd.Start()
	if err != nil {
		t.Fatal(err)
	}
	done := make(chan error, 1)
	go func() {
		done <- cmd.Wait()
	}()
	logFound := make(chan struct{})
	go f(wd, logFound)
	select {
	case <-time.After(3 * time.Second):
		if err := cmd.Process.Kill(); err != nil {
			t.Error("failed to kill process: ", err)
		}
		t.Error("timed out before success condition")
	case err := <-done:
		if err != nil {
			t.Errorf("process exited with error = %v", err)
		} else {
			t.Error("process exited before success condition")
		}
	case <-logFound:
		return
	}
}
