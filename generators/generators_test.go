package generators

import (
	"testing"

	must "github.com/theothertomelliott/go-must"
	"github.com/yext/edward/common"
	"github.com/yext/edward/services"
)

func TestGoGenerator(t *testing.T) {
	var goTests = []struct {
		name        string
		path        string
		targets     []string
		outServices []*services.ServiceConfig
		outErr      error
	}{

		{
			name: "Go Simple",
			path: "testdata/go/simple/",
			outServices: []*services.ServiceConfig{
				{
					Name:      "simple",
					Path:      common.StringToStringPointer("gocode/src/yext/simple"),
					Env:       []string{},
					WatchJson: []byte("{\"include\":[\"gocode/src/yext/simple\"]}"),
					Commands: services.ServiceConfigCommands{
						Build:  "go install",
						Launch: "simple",
					},
				},
			},
			outErr: nil,
		},
		{
			name: "Go Multiple unfiltered",
			path: "testdata/go/multiple/",
			outServices: []*services.ServiceConfig{
				{
					Name:      "service1",
					Path:      common.StringToStringPointer("service1"),
					Env:       []string{},
					WatchJson: []byte("{\"include\":[\"service1\"]}"),
					Commands: services.ServiceConfigCommands{
						Build:  "go install",
						Launch: "service1",
					},
				},
				{
					Name:      "service2",
					Path:      common.StringToStringPointer("service2"),
					Env:       []string{},
					WatchJson: []byte("{\"include\":[\"service2\"]}"),
					Commands: services.ServiceConfigCommands{
						Build:  "go install",
						Launch: "service2",
					},
				},
			},
			outErr: nil,
		},
		{
			name:    "Go Multiple filtered",
			path:    "testdata/go/multiple/",
			targets: []string{"service1"},
			outServices: []*services.ServiceConfig{
				{
					Name:      "service1",
					Path:      common.StringToStringPointer("service1"),
					Env:       []string{},
					WatchJson: []byte("{\"include\":[\"service1\"]}"),
					Commands: services.ServiceConfigCommands{
						Build:  "go install",
						Launch: "service1",
					},
				},
			},
			outErr: nil,
		},
	}
	for _, test := range goTests {
		t.Run(test.name, func(t *testing.T) {
			gc := &GeneratorCollection{
				Generators: []Generator{&GoGenerator{}},
				Path:       test.path,
				Targets:    test.targets,
			}
			err := gc.Generate()
			services := gc.Services()
			must.BeEqual(t, test.outServices, services, "services did not match.")
			must.BeEqualErrors(t, test.outErr, err, "errors did not match.")
		})
	}
}

func TestDockerGenerator(t *testing.T) {
	var tests = []struct {
		name        string
		path        string
		targets     []string
		outServices []*services.ServiceConfig
		outErr      error
	}{

		{
			name: "Docker Simple",
			path: "testdata/docker/single/",
			outServices: []*services.ServiceConfig{
				{
					Name: "service",
					Path: common.StringToStringPointer("service"),
					Env:  []string{},
					Commands: services.ServiceConfigCommands{
						Build:  "docker build -t service:edward .",
						Launch: "docker run -p 80:80 service:edward",
					},
					LaunchChecks: &services.LaunchChecks{
						Ports: []int{80},
					},
				},
			},
			outErr: nil,
		},
	}
	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			gc := &GeneratorCollection{
				Generators: []Generator{&DockerGenerator{}},
				Path:       test.path,
				Targets:    test.targets,
			}
			err := gc.Generate()
			services := gc.Services()
			must.BeEqual(t, test.outServices, services, "services did not match.")
			must.BeEqualErrors(t, test.outErr, err, "errors did not match.")
		})
	}
}
