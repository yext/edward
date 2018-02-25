package services_test

import (
	"encoding/json"
	"testing"

	"github.com/yext/edward/services"
)

type loaderProto struct {
	new     func() services.Backend
	handles func(c services.Backend) bool
	builder func(s *services.ServiceConfig) (services.Builder, error)
	runner  func(s *services.ServiceConfig) (services.Runner, error)
}

func (l *loaderProto) New() services.Backend           { return l.new() }
func (l *loaderProto) Handles(c services.Backend) bool { return l.handles(c) }
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

func testBackendName(t *testing.T) {
	serviceType := services.BackendName("testBackendName")
	loader := &loaderProto{
		new: func() services.Backend {
			return &configTest{}
		},
		handles: func(c services.Backend) bool {
			_, matches := c.(*configTest)
			return matches
		},
	}

	services.RegisterBackend(serviceType, loader)

	configJson := `{
		"name": "testService",
		"backend": "testBackendName",
		"field": "value"
	}`

	var out *services.ServiceConfig = &services.ServiceConfig{}
	err := json.Unmarshal([]byte(configJson), out)
	if err != nil {
		t.Error(err)
		return
	}
}
