package edward_test

import (
	"fmt"
	"io/ioutil"
	"path"
	"testing"

	"github.com/theothertomelliott/must"
)

func TestTelemetryStart(t *testing.T) {
	// if testing.Short() {
	// 	t.Skip("skipping test in short mode.")
	// }

	var tests = []struct {
		name             string
		path             string
		config           string
		services         []string
		expectedFile     string
		expectedContent  string
		expectedServices int
		err              error
	}{
		{
			name:             "start telemetry",
			path:             "testdata/telemetry",
			config:           "edward.json",
			services:         []string{"service"},
			expectedFile:     "events.txt",
			expectedContent:  "start service\n",
			expectedServices: 1,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			var err error

			// Copy test content into a temp dir on the GOPATH & defer deletion
			client, wd, cleanup, err := createClient(test.config, test.name, test.path)
			defer cleanup()

			tf := newTestFollower()
			client.Follower = tf

			defer showLogsIfFailed(t, test.name, wd, client)

			err = client.Start(test.services, false, false, nil)
			must.BeEqualErrors(t, test.err, err)

			// Check file content is as expected
			expectedFilePath := path.Join(wd, test.expectedFile)
			content, err := ioutil.ReadFile(expectedFilePath)
			if err != nil {
				t.Errorf("could not read %q: %v", expectedFilePath, err)
			}
			must.BeEqual(t, test.expectedContent, string(content), fmt.Sprintf("content of file %q", expectedFilePath))

			// Verify that the process actually started
			verifyAndStopRunners(t, client, test.expectedServices)
		})
	}
}
