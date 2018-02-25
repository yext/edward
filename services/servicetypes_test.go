package services_test

import (
	"encoding/json"
	"testing"

	"github.com/yext/edward/services"
)

type loaderProto struct {
	new     func() services.ConfigType
	handles func(c services.ConfigType) bool
	builder func(s *services.ServiceConfig) (services.Builder, error)
	runner  func(s *services.ServiceConfig) (services.Runner, error)
}

func (l *loaderProto) New() services.ConfigType           { return l.new() }
func (l *loaderProto) Handles(c services.ConfigType) bool { return l.handles(c) }
func (l *loaderProto) Builder(s *services.ServiceConfig) (services.Builder, error) {
	return l.builder(s)
}
func (l *loaderProto) Runner(s *services.ServiceConfig) (services.Runner, error) { return l.runner(s) }

type configTest struct {
	Field string `json:"field"`
}

func (c *configTest) HasBuildStep() bool {
	return false
}

func (c *configTest) HasLaunchStep() bool {
	return false
}

func TestTypeLoading(t *testing.T) {
	serviceType := services.Type("testTypeLoading")
	loader := &loaderProto{
		new: func() services.ConfigType {
			return &configTest{}
		},
		handles: func(c services.ConfigType) bool {
			_, matches := c.(*configTest)
			return matches
		},
	}

	services.RegisterServiceType(serviceType, loader)

	configJson := `{
		"name": "testService",
		"type": "testTypeLoading",
		"field": "value"
	}`

	var out *services.ServiceConfig = &services.ServiceConfig{}
	err := json.Unmarshal([]byte(configJson), out)
	if err != nil {
		t.Error(err)
		return
	}
}
