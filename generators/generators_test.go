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
				Name:  "simple",
				Path:  common.StringToStringPointer("gocode/src/yext/simple"),
				Env:   []string{},
				Watch: common.StringToStringPointer("gocode/src/yext/simple"),
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
		path: "testdata/go_multiple/",
		outServices: []*services.ServiceConfig{
			{
				Name:  "service1",
				Path:  common.StringToStringPointer("service1"),
				Env:   []string{},
				Watch: common.StringToStringPointer("service1"),
				Commands: services.ServiceConfigCommands{
					Build:  "go install",
					Launch: "service1",
				},
			},
			{
				Name:  "service2",
				Path:  common.StringToStringPointer("service2"),
				Env:   []string{},
				Watch: common.StringToStringPointer("service2"),
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
		path:    "testdata/go_multiple/",
		targets: []string{"service1"},
		outServices: []*services.ServiceConfig{
			{
				Name:  "service1",
				Path:  common.StringToStringPointer("service1"),
				Env:   []string{},
				Watch: common.StringToStringPointer("service1"),
				Commands: services.ServiceConfigCommands{
					Build:  "go install",
					Launch: "service1",
				},
			},
		},
		outErr: nil,
	},
}

func TestGoGenerator(t *testing.T) {
	for _, test := range goTests {
		services, _, err := GenerateServices(test.path, test.targets)
		must.BeEqual(t, test.outServices, services, test.name+": services did not match.")
		must.BeEqualErrors(t, test.outErr, err, test.name+": errors did not match.")
	}
}
