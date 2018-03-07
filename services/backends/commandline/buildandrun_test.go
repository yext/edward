package commandline_test

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"path"
	"strings"
	"testing"

	"github.com/pkg/errors"

	"github.com/yext/edward/services"
	"github.com/yext/edward/services/backends/commandline"
)

func TestMain(m *testing.M) {
	// Register necessary backends
	services.RegisterDefaultBackend(&commandline.Loader{})

	os.Exit(m.Run())
}

func TestStartService(t *testing.T) {
	getPath := func(dir string) *string {
		path := path.Join("testdata", dir)
		return &path
	}

	var tests = []struct {
		name     string
		service  *services.ServiceConfig
		expected string
	}{
		{
			name:     "default launch check",
			expected: "Hello",
			service: &services.ServiceConfig{
				Path: getPath("service"),
				Backends: []*services.BackendConfig{
					{
						Name: "backend1",
						Type: "commandline",
						Config: &commandline.Backend{
							Commands: commandline.ServiceConfigCommands{
								Launch: "go run main.go",
							},
						},
					},
				},
			},
		},
		{
			name:     "log launch check",
			expected: "Hello",
			service: &services.ServiceConfig{
				Path: getPath("service"),
				Backends: []*services.BackendConfig{
					{
						Name: "backend1",
						Type: "commandline",
						Config: &commandline.Backend{
							Commands: commandline.ServiceConfigCommands{
								Launch: "go run main.go",
							},
							LaunchChecks: &commandline.LaunchChecks{
								LogText: "Started",
							},
						},
					},
				},
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			runner, err := services.GetRunner(test.service)
			if err != nil {
				t.Error(err)
				return
			}

			err = runner.Start(os.Stdout, os.Stderr)
			if err != nil {
				t.Error(err)
				return
			}

			status, err := runner.Status()
			if err != nil {
				t.Error(err)
				return
			}
			if len(status.Ports) == 0 {
				t.Error("Expected at least one port")
				return
			}

			url := fmt.Sprintf("http://127.0.0.1:%s/", status.Ports[0])

			resp, err := http.Get(url)
			if err != nil {
				t.Error(err)
				return
			}
			defer resp.Body.Close()
			body, err := ioutil.ReadAll(resp.Body)
			if !strings.Contains(string(body), test.expected) {
				t.Errorf("Response incorrect. Expected '%s', got '%s'", test.expected, string(body))
			}

			_, err = runner.Stop(".", nil)
			if err != nil {
				t.Error(err)
				return
			}

			resp, err = http.Get(url)
			if err == nil {
				t.Error("Expected request to fail after service stopped")
			}
		})
	}
}

func TestStartServiceFailure(t *testing.T) {
	getPath := func(dir string) *string {
		path := path.Join("testdata", dir)
		return &path
	}

	var tests = []struct {
		name     string
		service  *services.ServiceConfig
		expected error
	}{
		{
			name:     "service panic",
			expected: errors.New("process exited"),
			service: &services.ServiceConfig{
				Path: getPath("launchfailure"),
				Backends: []*services.BackendConfig{
					{
						Name: "backend1",
						Type: "commandline",
						Config: &commandline.Backend{
							Commands: commandline.ServiceConfigCommands{
								Launch: "go run main.go",
							},
						},
					},
				},
			},
		},
		{
			name:     "default launch check",
			expected: errors.New("process exited"),
			service: &services.ServiceConfig{
				Path: getPath("service"),
				Backends: []*services.BackendConfig{
					{
						Name: "backend1",
						Type: "commandline",
						Config: &commandline.Backend{
							Commands: commandline.ServiceConfigCommands{
								Launch: "go run missing.go",
							},
						},
					},
				},
			},
		},
		{
			name:     "log launch check",
			expected: errors.New("process exited"),
			service: &services.ServiceConfig{
				Path: getPath("service"),
				Backends: []*services.BackendConfig{
					{
						Name: "backend1",
						Type: "commandline",
						Config: &commandline.Backend{
							Commands: commandline.ServiceConfigCommands{
								Launch: "go run missing.go",
							},
							LaunchChecks: &commandline.LaunchChecks{
								LogText: "Started",
							},
						},
					},
				},
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			runner, err := services.GetRunner(test.service)
			if err != nil {
				t.Error(err)
				return
			}

			err = runner.Start(os.Stdout, os.Stderr)
			if err == nil {
				t.Error("expected an error on start, got nil")
				return
			}
			if fmt.Sprint(err) != fmt.Sprint(test.expected) {
				t.Errorf("expected error '%s', got '%s'", test.expected, err)
			}
		})
	}
}
