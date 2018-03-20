package edward_test

import (
	"testing"

	"github.com/theothertomelliott/must"
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
			client, wd, cleanup, err := createClient(test.config, test.name, test.path)
			defer cleanup()

			tf := newTestFollower()
			client.Follower = tf

			defer showLogsIfFailed(t, test.name, wd, client)

			err = client.Start(test.services, test.skipBuild, test.noWatch, test.exclude)
			must.BeEqual(t, test.expectedStates, tf.states)
			must.BeEqual(t, test.expectedMessages, tf.messages)
			must.BeEqualErrors(t, test.err, err)

			// Verify that the process actually started
			verifyAndStopRunners(t, client, test.expectedServices)
		})
	}
}
