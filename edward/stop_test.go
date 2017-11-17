package edward_test

import (
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path"
	"path/filepath"
	"strings"
	"syscall"
	"testing"
	"time"

	"github.com/theothertomelliott/must"
	"github.com/yext/edward/common"
	"github.com/yext/edward/edward"
	lumberjack "gopkg.in/natefinch/lumberjack.v2"
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
			var err error

			// Copy test content into a temp dir on the GOPATH & defer deletion
			wd, cleanup := createWorkingDir(t, test.name, test.path)
			defer cleanup()

			client, err := edward.NewClientWithConfig(
				path.Join(wd, test.config),
				common.EdwardVersion,
				log.New(&lumberjack.Logger{
					Filename:   filepath.Join(wd, "edward.log"),
					MaxSize:    50, // megabytes
					MaxBackups: 30,
					MaxAge:     1, //days
				}, "", log.Ldate|log.Ltime|log.Lmicroseconds|log.Lshortfile),
			)
			if err != nil {
				t.Fatal(err)
			}
			// TODO: Configure the client to set the runner's log file to the same as Logger above.
			client.WorkingDir = wd
			tf := newTestFollower()
			client.Follower = tf

			client.EdwardExecutable = edwardExecutable
			client.DisableConcurrentPhases = true
			client.Tags = []string{fmt.Sprintf("test.stop.%d", time.Now().UnixNano())}

			err = client.Start(test.servicesStart, test.skipBuild, false, test.noWatch, test.exclude)
			if err != nil {
				b, err := ioutil.ReadFile(filepath.Join(wd, "edward.log"))
				if err != nil {
					t.Fatal(err)
				}
				fmt.Print("=== Log ===\n", string(b), "=== /Log ===\n")
				fmt.Println("=== Messages ===\n", strings.Join(tf.messages, "\n"), "\n=== /Messages ===")
				t.Fatal(err)
			}

			childProcesses := getRunnerAndServiceProcesses(t)

			// Reset the follower
			tf = newTestFollower()
			client.Follower = tf

			err = client.Stop(test.servicesStop, true, test.exclude, false)
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
