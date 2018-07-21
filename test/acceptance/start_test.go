package acceptance

import (
	"net/http"
	"strings"
	"testing"
	"time"
)

func TestStartSuccess(t *testing.T) {
	var tests = []struct {
		name         string
		dataDir      string
		startArgs    []string
		stopArgs     []string
		expectedURLs map[string]string
	}{
		{
			name:      "single",
			dataDir:   "testdata/single",
			startArgs: []string{"start", "service"},
			stopArgs:  []string{"stop", "service"},
			expectedURLs: map[string]string{
				"http://127.0.0.1:51234/": "Hello",
			},
		},
		{
			name:      "alternate config",
			dataDir:   "testdata/single",
			startArgs: []string{"-c", "alternate.json", "start", "alternate"},
			stopArgs:  []string{"-c", "alternate.json", "stop", "alternate"},
			expectedURLs: map[string]string{
				"http://127.0.0.1:51234/": "Hello",
			},
		},
		{
			name:      "group",
			dataDir:   "testdata/group",
			startArgs: []string{"start", "group"},
			stopArgs:  []string{"stop", "group"},
			expectedURLs: map[string]string{
				"http://127.0.0.1:51936/": "Hello",
				"http://127.0.0.1:51937/": "Hello",
				"http://127.0.0.1:51938/": "Hello",
			},
		},
		{
			name:      "group alias",
			dataDir:   "testdata/group",
			startArgs: []string{"start", "groupalias"},
			stopArgs:  []string{"stop", "groupalias"},
			expectedURLs: map[string]string{
				"http://127.0.0.1:51936/": "Hello",
				"http://127.0.0.1:51937/": "Hello",
				"http://127.0.0.1:51938/": "Hello",
			},
		},
		{
			name:      "multiple",
			dataDir:   "testdata/group",
			startArgs: []string{"start", "service1", "service2", "service3"},
			stopArgs:  []string{"stop", "service1", "service2", "service3"},
			expectedURLs: map[string]string{
				"http://127.0.0.1:51936/": "Hello",
				"http://127.0.0.1:51937/": "Hello",
				"http://127.0.0.1:51938/": "Hello",
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			workingDir, cleanup, err := createWorkingDir("testStart", test.dataDir)
			defer cleanup()
			if err != nil {
				t.Fatal(err)
			}
			executeCommand(t, workingDir, edwardExecutable, test.startArgs...)
			for url, content := range test.expectedURLs {
				expectFromURL(t, content, url)
			}
			executeCommand(t, workingDir, edwardExecutable, test.stopArgs...)
			for url := range test.expectedURLs {
				expectErrorFromURL(t, url)
			}
		})
	}
}

func TestStartFailure(t *testing.T) {
	var tests = []struct {
		name         string
		dataDir      string
		startArgs    []string
		expectedURLs []string
	}{
		{
			name:      "launch failure",
			dataDir:   "testdata/launchfailure",
			startArgs: []string{"start", "broken"},
			expectedURLs: []string{
				"http://127.0.0.1:51234/",
			},
		},
		{
			name:      "launch failure stops subsequent",
			dataDir:   "testdata/launchfailure",
			startArgs: []string{"start", "broken", "working"},
			expectedURLs: []string{
				"http://127.0.0.1:51234/",
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			workingDir, cleanup, err := createWorkingDir("testStart", test.dataDir)
			defer cleanup()
			if err != nil {
				t.Fatal(err)
			}
			executeCommandExpectFailure(t, workingDir, edwardExecutable, test.startArgs...)
			for _, url := range test.expectedURLs {
				expectErrorFromURL(t, url)
			}
		})
	}
}

func TestStartMultipleBackends(t *testing.T) {
	workingDir, cleanup, err := createWorkingDir("testStartMultipleBackends", "testdata/multiple_backends")
	defer cleanup()
	if err != nil {
		t.Fatal(err)
	}
	// Expect that the broken backend fails
	executeCommandExpectFailure(t, workingDir, edwardExecutable, "start", "-b", "service:broken_build", "service")
	executeCommandExpectFailure(t, workingDir, edwardExecutable, "start", "-b", "service:broken_launch", "service")
	// Expect that the default backend succeeds
	executeCommand(t, workingDir, edwardExecutable, "start", "service")
	// Expect that the working backend succeeds when explicitly specified
	executeCommand(t, workingDir, edwardExecutable, "start", "-b", "service:working", "service")
}

func TestDied(t *testing.T) {
	workingDir, cleanup, err := createWorkingDir("testStart", "testdata/single")
	defer cleanup()
	if err != nil {
		t.Fatal("Creating working dir:", err)
	}

	executeCommand(t, workingDir, edwardExecutable, "start", "service")

	status := executeCommandGetOutput(t, workingDir, edwardExecutable, "status", "service")
	if !strings.Contains(status, "RUNNING") {
		t.Errorf("service was not running after start")
	}
	_, err = http.Get("http://127.0.0.1:51234/")
	if err != nil {
		t.Error("failed opening port: ", err)
	}
	// Test died state
	for strings.Contains(status, "RUNNING") {
		time.Sleep(time.Millisecond * 50)
		status = executeCommandGetOutput(t, workingDir, edwardExecutable, "status", "service")
		if strings.Contains(status, "DIED") {
			break
		}
	}
	executeCommand(t, workingDir, edwardExecutable, "stop", "service")
}
