package edward_test

import (
	"os"
	"syscall"
	"testing"

	"github.com/theothertomelliott/go-must"
	"github.com/yext/edward/common"
	"github.com/yext/edward/config"
	"github.com/yext/edward/edward"
	"github.com/yext/edward/home"
)

func TestStopAll(t *testing.T) {
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
			name:     "single service",
			path:     "testdata/start1",
			config:   "edward.json",
			services: []string{"service"},
			expectedStates: map[string]string{
				"service":        "Pending", // This isn't technically right
				"service > Stop": "Success",
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
			if err != nil {
				t.Fatal(err)
			}

			childProcesses := getRunnerAndServiceProcesses(t)

			// Reset the follower
			tf = newTestFollower()
			client.Follower = tf

			err = client.Stop(test.services, test.exclude)
			must.BeEqualErrors(t, test.err, err)
			must.BeEqual(t, test.expectedStates, tf.states)

			for _, p := range childProcesses {
				process, err := os.FindProcess(int(p.Pid))
				if err != nil {
					t.Fatal(err)
				}
				if err == nil {
					if process.Signal(syscall.Signal(0)) == nil {
						t.Errorf("process should not still be running: %v", p.Pid)
					}
				}
			}
		})
	}
}
