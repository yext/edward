package services_test

import (
	"encoding/json"
	"testing"

	"github.com/theothertomelliott/must"
	"github.com/yext/edward/services"
)

func TestJsonMarshal(t *testing.T) {
	tests := []struct {
		name          string
		serviceConfig *services.ServiceConfig
	}{
		{
			name: "simple command line",
			serviceConfig: &services.ServiceConfig{
				Name:       "simple service",
				TypeConfig: &services.ConfigCommandLine{},
			},
		},
		{
			name: "command line with commands",
			serviceConfig: &services.ServiceConfig{
				Name: "command line service",
				TypeConfig: &services.ConfigCommandLine{
					Commands: services.ServiceConfigCommands{
						Build:  "build",
						Launch: "launch",
						Stop:   "stop",
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
			err = json.Unmarshal(jsonData, out)
			if err != nil {
				t.Error(err)
				return
			}
			must.BeEqual(t, test.serviceConfig, out, "service was not returned as expected")
		})
	}
}
