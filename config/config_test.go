package config

import (
	"bytes"
	"errors"
	"io"
	"os"
	"path/filepath"
	"testing"

	must "github.com/theothertomelliott/go-must"

	"github.com/yext/edward/common"
	"github.com/yext/edward/services"
)

var service1 = services.ServiceConfig{
	Name:         "service1",
	Path:         common.StringToStringPointer("."),
	RequiresSudo: true,
	Commands: services.ServiceConfigCommands{
		Build:  "buildCmd",
		Launch: "launchCmd",
		Stop:   "stopCmd",
	},
	Properties: services.ServiceConfigProperties{
		Started: "startedProperty",
	},
	Logger: common.NullLogger{},
}

var group1 = services.ServiceGroupConfig{
	Name:     "group1",
	Services: []*services.ServiceConfig{&service1},
	Groups:   []*services.ServiceGroupConfig{},
	Logger:   common.NullLogger{},
}

var service2 = services.ServiceConfig{
	Name: "service2",
	Path: common.StringToStringPointer("service2/path"),
	Commands: services.ServiceConfigCommands{
		Build:  "buildCmd2",
		Launch: "launchCmd2",
		Stop:   "stopCmd2",
	},
	Logger: common.NullLogger{},
}

var group2 = services.ServiceGroupConfig{
	Name:     "group2",
	Services: []*services.ServiceConfig{&service2},
	Groups:   []*services.ServiceGroupConfig{},
	Logger:   common.NullLogger{},
}

var service3 = services.ServiceConfig{
	Name:         "service3",
	Path:         common.StringToStringPointer("."),
	RequiresSudo: true,
	Commands: services.ServiceConfigCommands{
		Build:  "buildCmd",
		Launch: "launchCmd",
		Stop:   "stopCmd",
	},
	Properties: services.ServiceConfigProperties{
		Started: "startedProperty",
	},
	Logger: common.NullLogger{},
}

var group3 = services.ServiceGroupConfig{
	Name:     "group3",
	Services: []*services.ServiceConfig{&service3},
	Groups:   []*services.ServiceGroupConfig{},
	Logger:   common.NullLogger{},
}

var basicTests = []struct {
	name          string
	inJson        string
	outServiceMap map[string]*services.ServiceConfig
	outGroupMap   map[string]*services.ServiceGroupConfig
	outErr        error
}{
	{
		name:          "Invalid, blank",
		inJson:        "",
		outServiceMap: nil,
		outGroupMap:   nil,
		outErr:        io.EOF,
	},
	{
		name:          "Valid, empty",
		inJson:        "{}",
		outServiceMap: make(map[string]*services.ServiceConfig),
		outGroupMap:   make(map[string]*services.ServiceGroupConfig),
		outErr:        nil,
	},
	{
		name: "Valid, services and groups",
		inJson: `
		{
			"services": [
				{
					"name": "service1",
					"path": ".",
					"requiresSudo": true,
					"commands": {
						"build": "buildCmd",
						"launch": "launchCmd",
						"stop": "stopCmd"
					},
					"log_properties": {
						"started": "startedProperty"
					}
				}
			],
			"groups": [
				{
					"name": "group1",
					"children": ["service1"]
				}
			]
		}
		`,
		outServiceMap: map[string]*services.ServiceConfig{
			"service1": &service1,
		},
		outGroupMap: map[string]*services.ServiceGroupConfig{
			"group1": &group1,
		},
		outErr: nil,
	},
	{
		name: "Invalid, self-referencing group",
		inJson: `
		{
			"groups": [
				{
					"name": "group1",
					"children": ["group1"]
				}
			]
		}
		`,
		outServiceMap: nil,
		outGroupMap:   nil,
		outErr:        errors.New("group cycle: group1"),
	},
	{
		name: "Invalid, group cycle",
		inJson: `
		{
			"groups": [
				{
					"name": "group1",
					"children": ["group2"]
				},
				{
					"name": "group2",
					"children": ["group1"]
				}
			]
		}
		`,
		outServiceMap: nil,
		outGroupMap:   nil,
		outErr:        errors.New("group cycle: group2"),
	},
	{
		name: "Invalid, 3 group cycle",
		inJson: `
		{
			"groups": [
				{
					"name": "group1",
					"children": ["group2"]
				},
				{
					"name": "group2",
					"children": ["group3"]
				},
				{
					"name": "group3",
					"children": ["group1"]
				}
			]
		}
		`,
		outServiceMap: nil,
		outGroupMap:   nil,
		outErr:        errors.New("group cycle: group3"),
	},
	{
		name: "Platform checks, no matches",
		inJson: `
		{
			"services": [
				{
					"name": "service1",
					"platform": "never matches"
				}
			]
		}
		`,
		outServiceMap: map[string]*services.ServiceConfig{},
		outGroupMap:   map[string]*services.ServiceGroupConfig{},
	},
	{
		name: "Platform check, should skip group",
		inJson: `
		{
			"services": [
				{
					"name": "service1",
					"platform": "never matches"
				}
			],
			"groups": [
				{
					"name": "group1",
					"children": ["service1"]
				}
			]
		}
		`,
		outServiceMap: map[string]*services.ServiceConfig{},
		outGroupMap: map[string]*services.ServiceGroupConfig{
			"group1": &services.ServiceGroupConfig{
				Name:   "group1",
				Logger: common.NullLogger{},
			},
		},
	},
	{
		name: "Platform check, replace with matching name",
		inJson: `
		{
			"services": [
				{
					"name": "service1",
					"platform": "never matches"
				},
				{
					"name": "service1"
				}
			],
			"groups": [
				{
					"name": "group1",
					"children": ["service1"]
				}
			]
		}
		`,
		outServiceMap: map[string]*services.ServiceConfig{
			"service1": &services.ServiceConfig{
				Name:   "service1",
				Logger: common.NullLogger{},
			},
		},
		outGroupMap: map[string]*services.ServiceGroupConfig{
			"group1": &services.ServiceGroupConfig{
				Name: "group1",
				Services: []*services.ServiceConfig{
					&services.ServiceConfig{
						Name:   "service1",
						Logger: common.NullLogger{},
					},
				},
				Logger: common.NullLogger{},
			},
		},
	},
	{
		name: "Platform check, replace with matching name, matching service first",
		inJson: `
		{
			"services": [
				{
					"name": "service1"
				},
				{
					"name": "service1",
					"platform": "never matches"
				}
			],
			"groups": [
				{
					"name": "group1",
					"children": ["service1"]
				}
			]
		}
		`,
		outServiceMap: map[string]*services.ServiceConfig{
			"service1": &services.ServiceConfig{
				Name:   "service1",
				Logger: common.NullLogger{},
			},
		},
		outGroupMap: map[string]*services.ServiceGroupConfig{
			"group1": &services.ServiceGroupConfig{
				Name: "group1",
				Services: []*services.ServiceConfig{
					&services.ServiceConfig{
						Name:   "service1",
						Logger: common.NullLogger{},
					},
				},
				Logger: common.NullLogger{},
			},
		},
	},
}

func TestLoadConfigBasic(t *testing.T) {
	for _, test := range basicTests {
		cfg, err := LoadConfig(bytes.NewBufferString(test.inJson), "", nil)
		validateTestResults(cfg, err, test.outServiceMap, test.outGroupMap, test.outErr, test.name, t)
	}
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
		name:          "Config missing imports",
		inFile:        "test2.json",
		outServiceMap: nil,
		outGroupMap:   nil,
		outErr:        errors.New("open imports2/import2.json: no such file or directory"),
	},
	{
		name:          "Duplicated import",
		inFile:        "test3.json",
		outServiceMap: nil,
		outGroupMap:   nil,
		outErr:        errors.New("Service name already exists: service2"),
	},
	{
		name:          "Duplicated service",
		inFile:        "test4.json",
		outServiceMap: nil,
		outGroupMap:   nil,
		outErr:        errors.New("Service name already exists: service1"),
	},
	{
		name:          "Duplicated group",
		inFile:        "test5.json",
		outServiceMap: nil,
		outGroupMap:   nil,
		outErr:        errors.New("Group name already exists: group"),
	},
}

func TestLoadConfigWithImports(t *testing.T) {
	err := os.Chdir("testdata")
	if err != nil {
		t.Errorf("%v", err)
		return
	}
	for _, test := range fileBasedTests {
		f, err := os.Open(test.inFile)
		if err != nil {
			t.Errorf("%v: Could not open input file", test.name)
			return
		}
		cfg, err := LoadConfigWithDir(f, filepath.Dir(test.inFile), "", nil)
		validateTestResults(cfg, err, test.outServiceMap, test.outGroupMap, test.outErr, test.name, t)
	}
}

func validateTestResults(cfg Config, err error, expectedServices map[string]*services.ServiceConfig, expectedGroups map[string]*services.ServiceGroupConfig, expectedErr error, name string, t *testing.T) {

	must.BeEqual(t, expectedServices, cfg.ServiceMap, name+": services did not match.")
	must.BeEqual(t, expectedGroups, cfg.GroupMap, name+": groups did not match.")

	must.BeEqualErrors(t, expectedErr, err, name+": Errors did not match.")
}
