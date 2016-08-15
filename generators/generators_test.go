package generators

import (
	"testing"

	must "github.com/theothertomelliott/go-must"
	"github.com/yext/edward/common"
	"github.com/yext/edward/services"
)

var goTests = []struct {
	name        string
	path        string
	targets     []string
	outServices []*services.ServiceConfig
	outErr      error
}{

	{
		name: "Go Simple",
		path: "testdata/go_simple/",
		outServices: []*services.ServiceConfig{
			{
				Name: "simple",
				Path: common.StringToStringPointer("gocode/src/yext/simple"),
				Env:  []string{},
				Commands: services.ServiceConfigCommands{
					Build:  "go install",
					Launch: "simple",
				},
				Properties: services.ServiceConfigProperties{
					Started: "Listening",
				},
			},
		},
		outErr: nil,
	},
	{
		name: "Go Multiple unfiltered",
		path: "testdata/go_multiple/",
		outServices: []*services.ServiceConfig{
			{
				Name: "service1",
				Path: common.StringToStringPointer("service1"),
				Env:  []string{},
				Commands: services.ServiceConfigCommands{
					Build:  "go install",
					Launch: "service1",
				},
				Properties: services.ServiceConfigProperties{
					Started: "Listening",
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
				Properties: services.ServiceConfigProperties{
					Started: "Listening",
				},
			},
		},
		outErr: nil,
	},
	{
		name:    "Go Multiple filtered",
		path:    "testdata/go_multiple/",
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
				Properties: services.ServiceConfigProperties{
					Started: "Listening",
				},
			},
		},
		outErr: nil,
	},
}

func TestGoGenerator(t *testing.T) {
	for _, test := range goTests {
		services, err := GenerateServices(test.path, test.targets)
		must.BeEqual(t, test.outServices, services, test.name+": services did not match.")
		must.BeEqualErrors(t, test.outErr, err, test.name+": errors did not match.")
	}
}
