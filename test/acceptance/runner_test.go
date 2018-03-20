package acceptance

import (
	"testing"
)

func TestRunFailure(t *testing.T) {
	var tests = []struct {
		name         string
		dataDir      string
		startArgs    []string
		expectedURLs []string
	}{
		{
			name:      "launch failure",
			dataDir:   "testdata/launchfailure",
			startArgs: []string{"run", "broken"},
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
