package acceptance

import "testing"

func TestStop(t *testing.T) {
	var tests = []struct {
		name         string
		dataDir      string
		startArgs    []string
		stopArgs     []string
		expectedURLs map[string]string
	}{
		{
			name:      "graceless shutdown",
			dataDir:   "testdata/graceless_shutdown",
			startArgs: []string{"start", "graceless"},
			stopArgs:  []string{"stop", "graceless"},
			expectedURLs: map[string]string{
				"http://127.0.0.1:51234/": "Hello",
			},
		},
		{
			name:      "graceless shutdown with timeout",
			dataDir:   "testdata/graceless_shutdown",
			startArgs: []string{"start", "gracelessWithTimeout"},
			stopArgs:  []string{"stop", "gracelessWithTimeout"},
			expectedURLs: map[string]string{
				"http://127.0.0.1:51234/": "Hello",
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
