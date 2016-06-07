package main

import (
	"bytes"
	"errors"
	"io"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/davecgh/go-spew/spew"
)

var service1 = ServiceConfig{
	Name:         "service1",
	Path:         stringToStringPointer("."),
	RequiresSudo: true,
	Commands: ServiceConfigCommands{
		Build:  "buildCmd",
		Launch: "launchCmd",
		Stop:   "stopCmd",
	},
	Properties: ServiceConfigProperties{
		Started: "startedProperty",
	},
}

var group1 = ServiceGroupConfig{
	Name:     "group1",
	Services: []*ServiceConfig{&service1},
	Groups:   []*ServiceGroupConfig{},
}

var service2 = ServiceConfig{
	Name: "service2",
	Path: stringToStringPointer("service2/path"),
	Commands: ServiceConfigCommands{
		Build:  "buildCmd2",
		Launch: "launchCmd2",
		Stop:   "stopCmd2",
	},
}

var group2 = ServiceGroupConfig{
	Name:     "group2",
	Services: []*ServiceConfig{&service2},
	Groups:   []*ServiceGroupConfig{},
}

var service3 = ServiceConfig{
	Name:         "service3",
	Path:         stringToStringPointer("."),
	RequiresSudo: true,
	Commands: ServiceConfigCommands{
		Build:  "buildCmd",
		Launch: "launchCmd",
		Stop:   "stopCmd",
	},
	Properties: ServiceConfigProperties{
		Started: "startedProperty",
	},
}

var group3 = ServiceGroupConfig{
	Name:     "group3",
	Services: []*ServiceConfig{&service3},
	Groups:   []*ServiceGroupConfig{},
}

func stringToStringPointer(s string) *string {
	return &s
}

var basicTests = []struct {
	name          string
	inJson        string
	outServiceMap map[string]*ServiceConfig
	outGroupMap   map[string]*ServiceGroupConfig
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
		outServiceMap: make(map[string]*ServiceConfig),
		outGroupMap:   make(map[string]*ServiceGroupConfig),
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
		outServiceMap: map[string]*ServiceConfig{
			"service1": &service1,
		},
		outGroupMap: map[string]*ServiceGroupConfig{
			"group1": &group1,
		},
		outErr: nil,
	},
}

func TestLoadConfigBasic(t *testing.T) {
	for _, test := range basicTests {
		cfg, err := LoadConfig(bytes.NewBufferString(test.inJson))
		validateTestResults(cfg, err, test.outServiceMap, test.outGroupMap, test.outErr, test.name, t)
	}
}

var fileBasedTests = []struct {
	name          string
	inFile        string
	outServiceMap map[string]*ServiceConfig
	outGroupMap   map[string]*ServiceGroupConfig
	outErr        error
}{
	{
		name:   "Config with imports",
		inFile: "testdata/test1.json",
		outServiceMap: map[string]*ServiceConfig{
			"service1": &service1,
			"service2": &service2,
			"service3": &service3,
		},
		outGroupMap: map[string]*ServiceGroupConfig{
			"group1": &group1,
			"group2": &group2,
			"group3": &group3,
		},
		outErr: nil,
	},
	{
		name:          "Config missing imports",
		inFile:        "testdata/test2.json",
		outServiceMap: nil,
		outGroupMap:   nil,
		outErr:        errors.New("open testdata/imports2/import2.json: no such file or directory"),
	},
	{
		name:          "Duplicated import",
		inFile:        "testdata/test3.json",
		outServiceMap: nil,
		outGroupMap:   nil,
		outErr:        errors.New("Service name already exists: service2"),
	},
	{
		name:          "Duplicated service",
		inFile:        "testdata/test4.json",
		outServiceMap: nil,
		outGroupMap:   nil,
		outErr:        errors.New("Service name already exists: service1"),
	},
}

func TestLoadConfigWithImports(t *testing.T) {
	for _, test := range fileBasedTests {
		f, err := os.Open(test.inFile)
		if err != nil {
			t.Errorf("%v: Could not open input file", test.name)
			return
		}
		cfg, err := LoadConfigWithDir(f, filepath.Dir(test.inFile))
		validateTestResults(cfg, err, test.outServiceMap, test.outGroupMap, test.outErr, test.name, t)
	}
}

func validateTestResults(cfg Config, err error, expectedServices map[string]*ServiceConfig, expectedGroups map[string]*ServiceGroupConfig, expectedErr error, name string, t *testing.T) {
	if !reflect.DeepEqual(cfg.ServiceMap, expectedServices) {
		t.Errorf("%v: Service maps did not match.\nExpected:\n%v\nGot:%v", name, spew.Sdump(expectedServices), spew.Sdump(cfg.ServiceMap))
	}

	if !reflect.DeepEqual(cfg.GroupMap, expectedGroups) {
		t.Errorf("%v: Group maps did not match. Expected %v, got %v", name, spew.Sdump(expectedGroups), spew.Sdump(cfg.GroupMap))
	}

	if err != nil && expectedErr != nil {
		if !reflect.DeepEqual(err.Error(), expectedErr.Error()) {
			t.Errorf("%v: Error did not match. Expected %v, got %v", name, expectedErr, err)
		}
	} else if expectedErr != nil {
		t.Errorf("%v: expected error, %v", name, expectedErr)
	} else if err != nil {
		t.Errorf("%v: unexpected error", name, err)
	}
}
