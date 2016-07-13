package generators

import (
	"reflect"
	"testing"

	"github.com/davecgh/go-spew/spew"
	"github.com/yext/edward/common"
	"github.com/yext/edward/services"
)

var goTests = []struct {
	name        string
	path        string
	outServices []*services.ServiceConfig
	outErr      error
}{

	{
		name: "Go Simple",
		path: "testdata/go_simple/",
		outServices: []*services.ServiceConfig{
			&services.ServiceConfig{
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
}

func TestGoGenerator(t *testing.T) {
	for _, test := range goTests {
		services, err := GenerateServices(test.path)
		if !reflect.DeepEqual(test.outServices, services) {
			t.Errorf("%v: Services did not match.\nExpected:\n%v\nGot:%v", test.name, spew.Sdump(test.outServices), spew.Sdump(services))
		}
		if err != nil && test.outErr != nil {
			if err.Error() != test.outErr.Error() {
				t.Errorf("%v: Error did not match. Expected %v, got %v", test.name, test.outErr, err)
			}
		} else if err != test.outErr {
			t.Errorf("%v: Errors did not match. Expected: %v, got: %v", test.name, test.outErr, err)
		}
	}
}
