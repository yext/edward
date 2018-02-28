package config

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	must "github.com/theothertomelliott/must"

	"github.com/yext/edward/common"
	"github.com/yext/edward/services"
	"github.com/yext/edward/services/backends/commandline"
)

func TestMain(m *testing.M) {
	// Register necessary backends
	services.RegisterDefaultBackend(&commandline.Loader{})

	os.Exit(m.Run())
}

var service1 = services.ServiceConfig{
	Name:         "service1",
	Description:  "My Service 1 is magic",
	Path:         common.StringToStringPointer("."),
	RequiresSudo: true,
	BackendConfig: &commandline.Backend{
		Commands: commandline.ServiceConfigCommands{
			Build:  "buildCmd",
			Launch: "launchCmd",
			Stop:   "stopCmd",
		},
		LaunchChecks: &commandline.LaunchChecks{
			LogText: "startedProperty",
		},
	},
	Logger: common.NullLogger{},
}

var service1alias = services.ServiceConfig{
	Name:         "service1",
	Aliases:      []string{"service2"},
	Path:         common.StringToStringPointer("."),
	RequiresSudo: true,
	BackendConfig: &commandline.Backend{
		Commands: commandline.ServiceConfigCommands{
			Build:  "buildCmd",
			Launch: "launchCmd",
			Stop:   "stopCmd",
		},
		LaunchChecks: &commandline.LaunchChecks{
			LogText: "startedProperty",
		},
	},
	Logger: common.NullLogger{},
}

var group1 = services.ServiceGroupConfig{
	Name:        "group1",
	Description: "My wonderfull group 1",
	Services:    []*services.ServiceConfig{&service1},
	Groups:      []*services.ServiceGroupConfig{},
	ChildOrder:  []string{"service1"},
	Logger:      common.NullLogger{},
}

var group1alias = services.ServiceGroupConfig{
	Name:       "group1",
	Aliases:    []string{"group2"},
	Services:   []*services.ServiceConfig{&service1alias},
	Groups:     []*services.ServiceGroupConfig{},
	ChildOrder: []string{"service1"},
	Logger:     common.NullLogger{},
}

var service2 = services.ServiceConfig{
	Name: "service2",
	Path: common.StringToStringPointer("service2/path"),
	BackendConfig: &commandline.Backend{
		Commands: commandline.ServiceConfigCommands{
			Build:  "buildCmd2",
			Launch: "launchCmd2",
			Stop:   "stopCmd2",
		},
	},
	Logger: common.NullLogger{},
}

var group2 = services.ServiceGroupConfig{
	Name:       "group2",
	Services:   []*services.ServiceConfig{&service2},
	Groups:     []*services.ServiceGroupConfig{},
	Logger:     common.NullLogger{},
	ChildOrder: []string{"service2"},
}

var service3 = services.ServiceConfig{
	Name:         "service3",
	Path:         common.StringToStringPointer("."),
	RequiresSudo: true,
	BackendConfig: &commandline.Backend{
		Commands: commandline.ServiceConfigCommands{
			Build:  "buildCmd",
			Launch: "launchCmd",
			Stop:   "stopCmd",
		},
		LaunchChecks: &commandline.LaunchChecks{
			LogText: "startedProperty",
		},
	},
	Logger: common.NullLogger{},
}

var group3 = services.ServiceGroupConfig{
	Name:       "group3",
	Services:   []*services.ServiceConfig{&service3},
	Groups:     []*services.ServiceGroupConfig{},
	Logger:     common.NullLogger{},
	ChildOrder: []string{"service3"},
}

var fileBasedTests = []struct {
	name          string
	inFile        string
	outServiceMap map[string]*services.ServiceConfig
	outGroupMap   map[string]*services.ServiceGroupConfig
	outErr        error
}{
	{
		name:   "Config with imports",
		inFile: "test1.json",
		outServiceMap: map[string]*services.ServiceConfig{
			"service1": &service1,
			"service2": &service2,
			"service3": &service3,
		},
		outGroupMap: map[string]*services.ServiceGroupConfig{
			"group1": &group1,
			"group2": &group2,
			"group3": &group3,
		},
		outErr: nil,
	},
	{
		name:   "Config missing imports",
		inFile: "test2.json",
		outErr: errors.New("open imports2/import2.json: no such file or directory"),
	},
	{
		name:   "Duplicated import",
		inFile: "test3.json",
		outErr: errors.New("Duplicate name or alias: service2"),
	},
	{
		name:   "Duplicated service",
		inFile: "test4.json",
		outErr: errors.New("Duplicate name or alias: service1"),
	},
	{
		name:   "Duplicated group",
		inFile: "test5.json",
		outErr: errors.New("Duplicate name or alias: group"),
	},
	{
		name:   "Group and service clash",
		inFile: "test6.json",
		outErr: errors.New("Duplicate name or alias: group"),
	},
	{
		name:   "Service alias clash",
		inFile: "test7.json",
		outErr: errors.New("Duplicate name or alias: service1"),
	},
	{
		name:   "Group alias clashes",
		inFile: "test8.json",
		outErr: errors.New("Duplicate name or alias: service1, service3"),
	},
	{
		name:   "Valid aliases",
		inFile: "test9.json",
		outServiceMap: map[string]*services.ServiceConfig{
			"service1": &service1alias,
		},
		outGroupMap: map[string]*services.ServiceGroupConfig{
			"group1": &group1alias,
		},
	},
	{
		name:   "Invalid json",
		inFile: "bad.json",
		outErr: errors.New("could not parse config file (line 7, char 9): invalid character ':' after array element"),
	},
}

func TestLoadConfigWithImports(t *testing.T) {
	err := os.Chdir("testdata")
	if err != nil {
		t.Errorf("%v", err)
		return
	}
	for _, test := range fileBasedTests {
		cfg, err := LoadConfig(test.inFile, "", nil)
		validateTestResults(cfg, err, test.inFile, test.outServiceMap, test.outGroupMap, test.outErr, test.name, t)
	}
}

func validateTestResults(cfg Config, err error, file string, expectedServices map[string]*services.ServiceConfig, expectedGroups map[string]*services.ServiceGroupConfig, expectedErr error, name string, t *testing.T) {
	for _, s := range expectedServices {
		s.ConfigFile, _ = filepath.Abs(file)
	}
	must.BeEqual(t, expectedServices, cfg.ServiceMap, name+": services did not match.")
	must.BeEqual(t, expectedGroups, cfg.GroupMap, name+": groups did not match.")

	must.BeEqualErrors(t, expectedErr, err, name+": Errors did not match.")
}
