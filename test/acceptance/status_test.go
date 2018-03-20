package acceptance

import (
	"strings"
	"testing"
)

func TestStatus(t *testing.T) {
	var tests = []struct {
		name         string
		dataDir      string
		startArgs    []string
		statusArgs   []string
		stopArgs     []string
		expectedText []string
	}{
		{
			name:         "single",
			dataDir:      "testdata/single",
			startArgs:    []string{"start", "service"},
			statusArgs:   []string{"status", "service"},
			stopArgs:     []string{"stop", "service"},
			expectedText: []string{"service", "51234"},
		},
		{
			name:       "group",
			dataDir:    "testdata/group",
			startArgs:  []string{"start", "group"},
			statusArgs: []string{"status", "group"},
			stopArgs:   []string{"stop", "group"},
			expectedText: []string{
				"service1", "51936",
				"service2", "51937",
				"service3", "51938",
			},
		},
		{
			name:       "group - status for all",
			dataDir:    "testdata/group",
			startArgs:  []string{"start", "group"},
			statusArgs: []string{"status", "group"},
			stopArgs:   []string{"stop", "group"},
			expectedText: []string{
				"service1", "51936",
				"service2", "51937",
				"service3", "51938",
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
			out := executeCommandGetOutput(t, workingDir, edwardExecutable, test.statusArgs...)
			for _, text := range test.expectedText {
				if !strings.Contains(out, text) {
					t.Errorf("text missing from status output: %s", text)
				}
			}
			executeCommand(t, workingDir, edwardExecutable, test.stopArgs...)
		})
	}
}
