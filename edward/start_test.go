package edward_test

import (
	"errors"
	"path"
	"testing"

	"github.com/theothertomelliott/must"
	"github.com/yext/edward/common"
	"github.com/yext/edward/edward"
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
		expectedMessages []string
		expectedServices int
		err              error
	}{
		{
			name:     "single service",
			path:     "testdata/single",
			config:   "edward.json",
			services: []string{"service"},
			expectedStates: map[string]string{
				"service":         "Pending", // This isn't technically right
				"service > Build": "Success",
				"service > Start": "Success",
			},
			expectedServices: 1,
		},
		{
			name:     "two services",
			path:     "testdata/multiple",
			config:   "edward.json",
			services: []string{"service1", "service2"},
			expectedStates: map[string]string{
				"service1":         "Pending", // This isn't technically right
				"service1 > Build": "Success",
				"service1 > Start": "Success",
				"service2":         "Pending", // This isn't technically right
				"service2 > Build": "Success",
				"service2 > Start": "Success",
			},
			expectedServices: 2,
		},
		{
			name:     "group",
			path:     "testdata/group",
			config:   "edward.json",
			services: []string{"group"},
			expectedStates: map[string]string{
				"group":                    "Pending",
				"group > service1":         "Pending",
				"group > service1 > Build": "Success",
				"group > service1 > Start": "Success",
				"group > service2":         "Pending",
				"group > service2 > Build": "Success",
				"group > service2 > Start": "Success",
				"group > service3":         "Pending",
				"group > service3 > Build": "Success",
				"group > service3 > Start": "Success",
			},
			expectedServices: 3,
		},
		{
			name:     "nested group",
			path:     "testdata/subgroup",
			config:   "edward.json",
			services: []string{"parentgroup"},
			expectedStates: map[string]string{
				"parentgroup":                                 "Pending",
				"parentgroup > service1":                      "Pending",
				"parentgroup > service1 > Build":              "Success",
				"parentgroup > service1 > Start":              "Success",
				"parentgroup > childgroup":                    "Pending",
				"parentgroup > childgroup > service2":         "Pending",
				"parentgroup > childgroup > service2 > Build": "Success",
				"parentgroup > childgroup > service2 > Start": "Success",
				"parentgroup > service3":                      "Pending",
				"parentgroup > service3 > Build":              "Success",
				"parentgroup > service3 > Start":              "Success",
			},
			expectedServices: 3,
		},
		{
			name:     "groupalias",
			path:     "testdata/group",
			config:   "edward.json",
			services: []string{"groupalias"},
			expectedStates: map[string]string{
				"group":                    "Pending",
				"group > service1":         "Pending",
				"group > service1 > Build": "Success",
				"group > service1 > Start": "Success",
				"group > service2":         "Pending",
				"group > service2 > Build": "Success",
				"group > service2 > Start": "Success",
				"group > service3":         "Pending",
				"group > service3 > Build": "Success",
				"group > service3 > Start": "Success",
			},
			expectedServices: 3,
		},
		{
			name:     "one service of two",
			path:     "testdata/multiple",
			config:   "edward.json",
			services: []string{"service2"},
			expectedStates: map[string]string{
				"service2":         "Pending", // This isn't technically right
				"service2 > Build": "Success",
				"service2 > Start": "Success",
			},
			expectedServices: 1,
		},
		{
			name:     "service not found",
			path:     "testdata/single",
			config:   "edward.json",
			services: []string{"missing"},
			err:      errors.New("Service or group not found"),
		},
		{
			name:     "warmup",
			path:     "testdata/features",
			config:   "edward.json",
			services: []string{"warmup"},
			expectedStates: map[string]string{
				"warmup":          "Pending", // This isn't technically right
				"warmup > Build":  "Success",
				"warmup > Start":  "Success",
				"warmup > Warmup": "Success",
			},
			expectedServices: 1,
		},
		{
			name:     "environment variables in services",
			path:     "testdata/features",
			config:   "edward.json",
			services: []string{"env"},
			expectedStates: map[string]string{
				"env":         "Pending", // This isn't technically right
				"env > Build": "Success",
				"env > Start": "Success",
			},
			expectedServices: 1,
		},
		{
			name:     "environment variables in groups",
			path:     "testdata/features",
			config:   "edward.json",
			services: []string{"env-group"},
			expectedStates: map[string]string{
				"env-group":                         "Pending",
				"env-group > env-for-group":         "Pending",
				"env-group > env-for-group > Build": "Success",
				"env-group > env-for-group > Start": "Success",
			},
			expectedServices: 1,
		},
		{
			name:     "launch check wait",
			path:     "testdata/features",
			config:   "edward.json",
			services: []string{"wait"},
			expectedStates: map[string]string{
				"wait":         "Pending", // This isn't technically right
				"wait > Build": "Success",
				"wait > Start": "Success",
			},
			expectedServices: 1,
		},
		{
			name:     "launch check log",
			path:     "testdata/features",
			config:   "edward.json",
			services: []string{"logLine"},
			expectedStates: map[string]string{
				"logLine":         "Pending", // This isn't technically right
				"logLine > Build": "Success",
				"logLine > Start": "Success",
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

			client, err := edward.NewClientWithConfig(path.Join(wd, test.config), common.EdwardVersion)
			if err != nil {
				t.Fatal(err)
			}
			client.WorkingDir = wd
			client.EdwardExecutable = edwardExecutable
			client.DisableConcurrentPhases = true

			tf := newTestFollower()
			client.Follower = tf

			err = client.Start(test.services, test.skipBuild, false, test.noWatch, test.exclude)
			must.BeEqual(t, test.expectedStates, tf.states)
			must.BeEqual(t, test.expectedMessages, tf.messages)
			must.BeEqualErrors(t, test.err, err)

			// Verify that the process actually started
			verifyAndStopRunners(t, test.expectedServices)
		})
	}
}

func TestStartOrder(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping test in short mode.")
	}

	var tests = []struct {
		name               string
		path               string
		config             string
		services           []string
		skipBuild          bool
		tail               bool
		noWatch            bool
		exclude            []string
		expectedStateOrder []string
		expectedServices   int
		err                error
	}{
		{
			name:     "group",
			path:     "testdata/group",
			config:   "edward.json",
			services: []string{"group"},
			expectedStateOrder: []string{
				"group",
				"group > service1",
				"group > service1 > Build",
				"group > service1 > Start",
				"group > service2",
				"group > service2 > Build",
				"group > service2 > Start",
				"group > service3",
				"group > service3 > Build",
				"group > service3 > Start",
			},
			expectedServices: 3,
		},
		{
			name:     "nested group",
			path:     "testdata/subgroup",
			config:   "edward.json",
			services: []string{"parentgroup"},
			expectedStateOrder: []string{
				"parentgroup",
				"parentgroup > service1",
				"parentgroup > service1 > Build",
				"parentgroup > service1 > Start",
				"parentgroup > childgroup",
				"parentgroup > childgroup > service2",
				"parentgroup > childgroup > service2 > Build",
				"parentgroup > childgroup > service2 > Start",
				"parentgroup > service3",
				"parentgroup > service3 > Build",
				"parentgroup > service3 > Start",
			},
			expectedServices: 3,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			var err error

			// Copy test content into a temp dir on the GOPATH & defer deletion
			wd, cleanup := createWorkingDir(t, test.name, test.path)
			defer cleanup()

			client, err := edward.NewClientWithConfig(path.Join(wd, test.config), common.EdwardVersion)
			if err != nil {
				t.Fatal(err)
			}
			client.WorkingDir = wd
			tf := newTestFollower()
			client.Follower = tf
			client.EdwardExecutable = edwardExecutable
			client.DisableConcurrentPhases = true

			err = client.Start(test.services, test.skipBuild, false, test.noWatch, test.exclude)
			must.BeEqual(t, test.expectedStateOrder, tf.stateOrder)
			must.BeEqualErrors(t, test.err, err)

			// Verify that the process actually started
			verifyAndStopRunners(t, test.expectedServices)
		})
	}
}
