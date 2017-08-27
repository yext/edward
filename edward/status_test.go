package edward_test

import (
	"strings"
	"testing"

	"github.com/theothertomelliott/must"
	"github.com/yext/edward/common"
	"github.com/yext/edward/config"
	"github.com/yext/edward/edward"
	"github.com/yext/edward/home"
)

func TestStatus(t *testing.T) {
	var tests = []struct {
		name      string
		path      string
		config    string
		services  []string
		skipBuild bool
		tail      bool
		noWatch   bool
		exclude   []string
		err       error
	}{
		{
			name:     "single service",
			path:     "testdata/single",
			config:   "edward.json",
			services: []string{"service"},
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

			// TODO: Support specifying services
			output, err := client.Status([]string{})
			for _, service := range test.services {
				if !strings.Contains(output, service) {
					t.Error("No status entry found for: ", service)
				}
			}
			must.BeEqualErrors(t, test.err, err)

			err = client.Stop(test.services, true, test.exclude)
			if err != nil {
				t.Fatal(err)
			}
		})
	}
}
