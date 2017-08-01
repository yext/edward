package edward_test

import (
	"testing"

	"github.com/theothertomelliott/go-must"
	"github.com/yext/edward/common"
	"github.com/yext/edward/config"
	"github.com/yext/edward/edward"
	"github.com/yext/edward/home"
)

func TestStart(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping test in short mode.")
	}

	var tests = []struct {
		name             string
		path             string
		config           string
		services         []string
		skipBuild        bool
		tail             bool
		noWatch          bool
		exclude          []string
		expectedStates   map[string]string
		expectedServices int
		err              error
	}{
		{
			name:     "successful start",
			path:     "testdata/start1",
			config:   "edward.json",
			services: []string{"service"},
			expectedStates: map[string]string{
				"service":         "Pending", // This isn't technically right
				"service > Build": "Success",
				"service > Start": "Success",
			},
			expectedServices: 1,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			// Set up edward home directory
			if err := home.EdwardConfig.Initialize(); err != nil {
				t.Fatal(err)
			}

			var err error

			// Copy test content into a temp dir on the GOPATH & defer deletion
			cleanup := createWorkingDir(t, test.name, test.path)
			defer cleanup()

			err = config.LoadSharedConfig(test.config, common.EdwardVersion, nil)
			if err != nil {
				t.Fatal(err)
			}

			client := edward.NewClient()
			client.Config = test.config
			tf := newTestFollower()
			client.Follower = tf

			client.EdwardExecutable = edwardExecutable

			err = client.Start(test.services, test.skipBuild, false, test.noWatch, test.exclude)
			must.BeEqualErrors(t, test.err, err)
			must.BeEqual(t, test.expectedStates, tf.states)

			// Verify that the process actually started
			verifyAndStopRunners(t, test.expectedServices)
		})
	}
}
