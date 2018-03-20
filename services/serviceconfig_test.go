package services_test

import (
	"encoding/json"
	"testing"

	"github.com/theothertomelliott/must"
	"github.com/yext/edward/services"
)

func TestJsonMarshal(t *testing.T) {
	loader := &loaderProto{
		new: func() services.Backend {
			return &configTest{}
		},
		name: "testTypeLoading",
		handles: func(c services.Backend) bool {
			_, matches := c.(*configTest)
			return matches
		},
	}

	services.RegisterBackend(loader)

	tests := []struct {
		name          string
		serviceConfig *services.ServiceConfig
	}{
		{
			name: "simple command line",
			serviceConfig: &services.ServiceConfig{
				Name: "simple service",
				Backends: []*services.BackendConfig{
					{
						Name:   "backend1",
						Type:   "testTypeLoading",
						Config: &configTest{},
					},
				},
			},
		},
		{
			name: "command line with commands",
			serviceConfig: &services.ServiceConfig{
				Name: "command line service",
				Backends: []*services.BackendConfig{
					{
						Name: "backend1",
						Type: "testTypeLoading",
						Config: &configTest{
							Field: "field_value",
						},
					},
				},
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			jsonData, err := json.Marshal(test.serviceConfig)
			if err != nil {
				t.Error(err)
				return
			}
			var out *services.ServiceConfig = &services.ServiceConfig{}
			t.Log(string(jsonData))
			err = json.Unmarshal(jsonData, out)
			if err != nil {
				t.Error(err)
				return
			}
			must.BeEqual(t, test.serviceConfig, out, "service was not returned as expected")
		})
	}
}
