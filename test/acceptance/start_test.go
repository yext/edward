package acceptance

import (
	"testing"
)

func TestStart(t *testing.T) {
	workingDir, cleanup, err := createWorkingDir("testStart", "testdata/single")
	defer cleanup()
	if err != nil {
		t.Fatal(err)
	}
	executeCommand(t, workingDir, edwardExecutable, "start", "service")
	expectFromURL(t, "Hello", "http://127.0.0.1:51234/")

	executeCommand(t, workingDir, edwardExecutable, "stop", "service")
	expectErrorFromURL(t, "http://127.0.0.1:51234/")
}

func TestStartAlternateConfig(t *testing.T) {
	workingDir, cleanup, err := createWorkingDir("testStart", "testdata/single")
	defer cleanup()
	if err != nil {
		t.Fatal(err)
	}
	executeCommand(t, workingDir, edwardExecutable, "-c", "alternate.json", "start", "alternate")
	expectFromURL(t, "Hello", "http://127.0.0.1:51234/")

	executeCommand(t, workingDir, edwardExecutable, "-c", "alternate.json", "stop", "alternate")
	expectErrorFromURL(t, "http://127.0.0.1:51234/")
}
