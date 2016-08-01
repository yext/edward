package config

import (
	"bytes"
	"errors"
	"io"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/davecgh/go-spew/spew"
	"github.com/shazow/go-diff"
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
}

func TestLoadConfigBasic(t *testing.T) {
	for _, test := range basicTests {
		cfg, err := LoadConfig(bytes.NewBufferString(test.inJson), nil)
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
		cfg, err := LoadConfigWithDir(f, filepath.Dir(test.inFile), nil)
		validateTestResults(cfg, err, test.outServiceMap, test.outGroupMap, test.outErr, test.name, t)
	}
}

func validateTestResults(cfg Config, err error, expectedServices map[string]*services.ServiceConfig, expectedGroups map[string]*services.ServiceGroupConfig, expectedErr error, name string, t *testing.T) {
	if !reflect.DeepEqual(cfg.ServiceMap, expectedServices) {
		t.Errorf("%v: Service maps did not match.\nExpected:\n%v\nGot:%v\nDiff:\n%v", name, spew.Sdump(expectedServices), spew.Sdump(cfg.ServiceMap), getDiffOrErrorText(expectedServices, cfg.ServiceMap))

	}

	if !reflect.DeepEqual(cfg.GroupMap, expectedGroups) {
		t.Errorf("%v: Group maps did not match.\nExpected:\n%v\nGot:%v\nDiff:\n%v", name, spew.Sdump(expectedGroups), spew.Sdump(cfg.GroupMap), getDiffOrErrorText(expectedGroups, cfg.GroupMap))
	}

	if err != nil && expectedErr != nil {
		if !reflect.DeepEqual(err.Error(), expectedErr.Error()) {
			t.Errorf("%v: Error did not match. Expected %v, got %v", name, expectedErr, err)
		}
	} else if expectedErr != nil {
		t.Errorf("%v: expected error, %v", name, expectedErr)
	} else if err != nil {
		t.Errorf("%v: unexpected error, %v", name, err)
	}
}

func getDiffOrErrorText(expected interface{}, got interface{}) string {
	d, err := getDiff(expected, got)
	if err != nil {
		return err.Error()
	}
	return d
}

func getDiff(expected interface{}, got interface{}) (string, error) {
	differ := diff.DefaultDiffer()
	readerA := strings.NewReader(spew.Sdump(expected))
	readerB := strings.NewReader(spew.Sdump(got))

	diffBuffer := new(bytes.Buffer)

	err := differ.Diff(diffBuffer, readerA, readerB)
	if err != nil {
		return "", err
	}

	return diffBuffer.String(), nil
}
