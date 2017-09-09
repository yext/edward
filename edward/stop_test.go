package edward_test

import (
	"os"
	"path"
	"syscall"
	"testing"

	"github.com/theothertomelliott/must"
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
		servicesStart    []string
		servicesStop     []string
		skipBuild        bool
		tail             bool
		noWatch          bool
		exclude          []string
		expectedStates   map[string]string
		expectedMessages []string
		expectedServices int
		err              error
	}{
		{
			name:          "single service",
			path:          "testdata/single",
			config:        "edward.json",
			servicesStart: []string{"service"},
			servicesStop:  []string{"service"},
			expectedStates: map[string]string{
				"service":        "Pending", // This isn't technically right
				"service > Stop": "Success",
			},
			expectedServices: 1,
		},
		{
			name:          "group, stop all",
			path:          "testdata/group",
			config:        "edward.json",
			servicesStart: []string{"group"},
			expectedStates: map[string]string{
				"service1":        "Pending",
				"service1 > Stop": "Success",
				"service2":        "Pending",
				"service2 > Stop": "Success",
				"service3":        "Pending",
				"service3 > Stop": "Success",
			},
			expectedServices: 3,
		},
		{
			name:          "graceless shutdown",
			path:          "testdata/graceless_shutdown",
			config:        "edward.json",
			servicesStart: []string{"graceless"},
			servicesStop:  []string{"graceless"},
			expectedStates: map[string]string{
				"graceless":        "Pending", // This isn't technically right
				"graceless > Stop": "Success",
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
			wd, cleanup := createWorkingDir(t, test.name, test.path)
			defer cleanup()

			err = config.LoadSharedConfig(path.Join(wd, test.config), common.EdwardVersion, nil)
			if err != nil {
				t.Fatal(err)
			}

			client := edward.NewClient()

			client.WorkingDir = wd
			client.Config = test.config
			tf := newTestFollower()
			client.Follower = tf

			client.EdwardExecutable = edwardExecutable
			client.DisableConcurrentPhases = true

			err = client.Start(test.servicesStart, test.skipBuild, false, test.noWatch, test.exclude)
			if err != nil {
				t.Fatal(err)
			}

			childProcesses := getRunnerAndServiceProcesses(t)

			// Reset the follower
			tf = newTestFollower()
			client.Follower = tf

			err = client.Stop(test.servicesStop, true, test.exclude)
			must.BeEqualErrors(t, test.err, err)
			must.BeEqual(t, test.expectedStates, tf.states)
			must.BeEqual(t, test.expectedMessages, tf.messages)

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
