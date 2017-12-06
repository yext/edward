package edward_test

import (
	"fmt"
	"log"
	"path"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/theothertomelliott/must"
	"github.com/yext/edward/common"
	"github.com/yext/edward/edward"
	lumberjack "gopkg.in/natefinch/lumberjack.v2"
)

func TestStatus(t *testing.T) {
	var tests = []struct {
		name             string
		path             string
		config           string
		runningServices  []string
		inServices       []string
		expectedServices []string
		err              error
	}{
		{
			name:             "single service",
			path:             "testdata/single",
			config:           "edward.json",
			runningServices:  []string{"service"},
			expectedServices: []string{"service"},
		},
		{
			name:             "multiple services",
			path:             "testdata/multiple",
			config:           "edward.json",
			runningServices:  []string{"service1", "service2"},
			expectedServices: []string{"service1", "service2"},
		},
		{
			name:             "multiple services - one specified",
			path:             "testdata/multiple",
			config:           "edward.json",
			runningServices:  []string{"service1", "service2"},
			inServices:       []string{"service2"},
			expectedServices: []string{"service2"},
		},
		{
			name:             "full group",
			path:             "testdata/group",
			config:           "edward.json",
			runningServices:  []string{"group"},
			inServices:       []string{"group"},
			expectedServices: []string{"service1", "service2", "service3"},
		},
		{
			name:             "partial group",
			path:             "testdata/group",
			config:           "edward.json",
			runningServices:  []string{"service2", "service3"},
			inServices:       []string{"group"},
			expectedServices: []string{"service2", "service3"},
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
			client.WorkingDir = wd
			tf := newTestFollower()
			client.Follower = tf

			client.EdwardExecutable = edwardExecutable
			client.DisableConcurrentPhases = true
			client.Tags = []string{fmt.Sprintf("test.status.%d", time.Now().UnixNano())}

			err = client.Start(test.runningServices, false, false, false, nil)
			if err != nil {
				t.Fatal(err)
			}

			output, err := client.Status(test.inServices, false)
			for _, service := range test.expectedServices {
				if !strings.Contains(output, service) {
					t.Error("No status entry found for: ", service)
				}
			}
			must.BeEqualErrors(t, test.err, err)

			err = client.Stop(test.runningServices, true, nil, false)
			if err != nil {
				t.Fatal(err)
			}
		})
	}
}
