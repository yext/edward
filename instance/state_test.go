package instance_test

import (
	"io/ioutil"
	"log"
	"os"
	"testing"

	"github.com/theothertomelliott/must"
	"github.com/yext/edward/instance"
	"github.com/yext/edward/services"
)

func TestSerialization(t *testing.T) {
	var tests = []struct {
		name             string
		service          *services.ServiceConfig
		instanceStatuses map[string]instance.Status
	}{
		{
			name: "single status",
			service: &services.ServiceConfig{
				Name:       "testService",
				ConfigFile: "/path/to/config/file",
			},
			instanceStatuses: map[string]instance.Status{
				"test": instance.Status{
					State: instance.StateStarting,
				},
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			testDir, err := ioutil.TempDir("", test.name)
			if err != nil {
				log.Fatal(err)
			}
			defer os.RemoveAll(testDir)

			for identifier, status := range test.instanceStatuses {
				err := instance.SaveStatusForService(test.service, identifier, status, testDir)
				if err != nil {
					t.Errorf("error saving status: %v", err)
				}
			}

			got, err := instance.LoadStatusForService(test.service, testDir)
			if err != nil {
				t.Errorf("error loading status: %v", err)
			}
			must.BeEqual(t, test.instanceStatuses, got)
		})
	}
}

func TestDelete(t *testing.T) {
	var tests = []struct {
		name             string
		service          *services.ServiceConfig
		instanceStatuses map[string]instance.Status
	}{
		{
			name: "single status",
			service: &services.ServiceConfig{
				Name:       "testService",
				ConfigFile: "/path/to/config/file",
			},
			instanceStatuses: map[string]instance.Status{
				"test": instance.Status{
					State: instance.StateStarting,
				},
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			testDir, err := ioutil.TempDir("", test.name)
			if err != nil {
				log.Fatal(err)
			}
			defer os.RemoveAll(testDir)

			for identifier, status := range test.instanceStatuses {
				err := instance.SaveStatusForService(test.service, identifier, status, testDir)
				if err != nil {
					t.Errorf("error saving status: %v", err)
				}
				err = instance.DeleteStatusForService(test.service, identifier, testDir)
				if err != nil {
					t.Errorf("error deleting status: %v", err)
				}
			}

			got, err := instance.LoadStatusForService(test.service, testDir)
			if err != nil {
				t.Errorf("error loading status: %v", err)
			}
			must.BeEqual(t, 0, len(got))
		})
	}
}
