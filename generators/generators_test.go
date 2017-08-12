package generators

import (
	"errors"
	"testing"

	must "github.com/theothertomelliott/must"
	"github.com/yext/edward/common"
	"github.com/yext/edward/services"
)

func TestInvalidPaths(t *testing.T) {
	var goTests = []struct {
		name        string
		path        string
		targets     []string
		outServices []*services.ServiceConfig
		outErr      error
	}{
		{
			name:   "Invalid path",
			path:   "invalid_path",
			outErr: errors.New("stat invalid_path: no such file or directory"),
		},
		{
			name:   "Not directory",
			path:   "testdata/go/multiple/service1/main.go",
			outErr: errors.New("testdata/go/multiple/service1/main.go is not a directory"),
		},
	}
	for _, test := range goTests {
		t.Run(test.name, func(t *testing.T) {
			gc := &GeneratorCollection{
				Generators: []Generator{},
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

func TestEdwardGenerator(t *testing.T) {
	var goTests = []struct {
		name        string
		path        string
		targets     []string
		outServices []*services.ServiceConfig
		outGroups   []*services.ServiceGroupConfig
		outImports  []string
		outErr      error
	}{
		{
			name: "Edward Simple",
			path: "testdata/edward/simple/",
			outImports: []string{
				"project1/edward.json",
			},
		},
		{
			name: "Edward With Go",
			path: "testdata/edward/with_go",
			outImports: []string{
				"project1/edward.json",
			},
			outServices: []*services.ServiceConfig{
				{
					Name: "goproject",
					Path: common.StringToStringPointer("goproject"),
					Env:  []string{},
					Commands: services.ServiceConfigCommands{
						Build:  "go install",
						Launch: "goproject",
					},
				},
			},
		},
	}
	for _, test := range goTests {
		t.Run(test.name, func(t *testing.T) {
			gc := &GeneratorCollection{
				Generators: []Generator{
					&EdwardGenerator{},
					&GoGenerator{},
				},
				Path:    test.path,
				Targets: test.targets,
			}
			err := gc.Generate()
			services := gc.Services()
			groups := gc.Groups()
			imports := gc.Imports()
			must.BeEqual(t, test.outServices, services, "services did not match.")
			must.BeEqual(t, test.outGroups, groups, "groups did not match.")
			must.BeEqual(t, test.outImports, imports, "imports did not match.")
			must.BeEqualErrors(t, test.outErr, err, "errors did not match.")
		})
	}
}

func TestGoGenerator(t *testing.T) {
	var goTests = []struct {
		name        string
		path        string
		targets     []string
		outServices []*services.ServiceConfig
		outGroups   []*services.ServiceGroupConfig
		outImports  []string
		outErr      error
	}{

		{
			name: "Go Simple",
			path: "testdata/go/simple/",
			outServices: []*services.ServiceConfig{
				{
					Name: "simple",
					Path: common.StringToStringPointer("gocode/src/yext/simple"),
					Env:  []string{},
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
					Name: "service1",
					Path: common.StringToStringPointer("service1"),
					Env:  []string{},
					Commands: services.ServiceConfigCommands{
						Build:  "go install",
						Launch: "service1",
					},
				},
				{
					Name: "service2",
					Path: common.StringToStringPointer("service2"),
					Env:  []string{},
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
					Name: "service1",
					Path: common.StringToStringPointer("service1"),
					Env:  []string{},
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
			groups := gc.Groups()
			imports := gc.Imports()
			must.BeEqual(t, test.outServices, services, "services did not match.")
			must.BeEqual(t, test.outGroups, groups, "groups did not match.")
			must.BeEqual(t, test.outImports, imports, "imports did not match.")
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
		outGroups   []*services.ServiceGroupConfig
		outImports  []string
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
			groups := gc.Groups()
			imports := gc.Imports()
			must.BeEqual(t, test.outServices, services, "services did not match.")
			must.BeEqual(t, test.outGroups, groups, "groups did not match.")
			must.BeEqual(t, test.outImports, imports, "imports did not match.")
			must.BeEqualErrors(t, test.outErr, err, "errors did not match.")
		})
	}
}

func TestProcfileGenerator(t *testing.T) {
	var tests = []struct {
		name        string
		path        string
		targets     []string
		outServices []*services.ServiceConfig
		outGroups   []*services.ServiceGroupConfig
		outImports  []string
		outErr      error
	}{
		{
			name: "Procfile Simple",
			path: "testdata/procfiles/simple/",
			outServices: []*services.ServiceConfig{
				{
					Name: "service-database",
					Path: common.StringToStringPointer("service"),
					Env:  []string{},
					Commands: services.ServiceConfigCommands{
						Launch: "db launch command",
					},
				},
				{
					Name: "service-web",
					Path: common.StringToStringPointer("service"),
					Env:  []string{},
					Commands: services.ServiceConfigCommands{
						Launch: "web launch command",
					},
				},
			},
			outGroups: []*services.ServiceGroupConfig{
				{
					Name: "service",
					Services: []*services.ServiceConfig{
						{
							Name: "service-web",
							Path: common.StringToStringPointer("service"),
							Env:  []string{},
							Commands: services.ServiceConfigCommands{
								Launch: "web launch command",
							},
						},
						{
							Name: "service-database",
							Path: common.StringToStringPointer("service"),
							Env:  []string{},
							Commands: services.ServiceConfigCommands{
								Launch: "db launch command",
							},
						},
					},
				},
			},
		},
	}

	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			gc := &GeneratorCollection{
				Generators: []Generator{&ProcfileGenerator{}},
				Path:       test.path,
				Targets:    test.targets,
			}
			err := gc.Generate()
			services := gc.Services()
			groups := gc.Groups()
			imports := gc.Imports()
			must.BeEqual(t, test.outServices, services, "services did not match.")
			must.BeEqual(t, test.outGroups, groups, "groups did not match.")
			must.BeEqual(t, test.outImports, imports, "imports did not match.")
			must.BeEqualErrors(t, test.outErr, err, "errors did not match.")
		})
	}
}
