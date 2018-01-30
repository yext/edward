package edward_test

import (
	"testing"
	"time"

	"github.com/theothertomelliott/must"
)

func TestLog(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping test in short mode.")
	}
	var tests = []struct {
		name             string
		path             string
		config           string
		services         []string
		skipBuild        bool
		err              error
		expectedServices int
	}{
		{
			name:             "single service",
			path:             "testdata/single",
			config:           "edward.json",
			services:         []string{"service"},
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

			err = client.Start(test.services, test.skipBuild, false, nil)
			if err != nil {
				t.Fatal(err)
			}

			// TODO: redirect output for tailing and verify

			var finishChan = make(chan struct{})
			close(finishChan)
			var logDone = make(chan struct{})
			go func() {
				err = client.Log(test.services, finishChan)
				must.BeEqualErrors(t, test.err, err)
				close(logDone)
			}()

			select {
			case _ = <-time.After(time.Second):
				t.Error("log exit timed out")
			case _ = <-logDone:
				t.Log("log command finished")
			}

			// Verify that the process actually started
			verifyAndStopRunners(t, client, test.expectedServices)
		})
	}
}
