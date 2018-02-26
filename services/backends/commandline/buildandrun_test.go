package commandline_test

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"path"
	"strings"
	"testing"

	"github.com/yext/edward/home"

	"github.com/yext/edward/services"
	"github.com/yext/edward/services/backends/commandline"
)

func TestMain(m *testing.M) {
	// Register necessary backends
	services.RegisterBackend(commandline.TypeCommandLine, &commandline.CommandLineLoader{})
	services.SetDefaultBackend(commandline.TypeCommandLine)

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
			name:     "service",
			expected: "Hello",
			service: &services.ServiceConfig{
				Path: getPath("service"),
				BackendConfig: &commandline.CommandLineBackend{
					Commands: commandline.ServiceConfigCommands{
						Launch: "go run main.go",
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

			err = runner.Start(&home.EdwardConfiguration{}, os.Stdout, os.Stderr)
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
