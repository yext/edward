package docker

import "github.com/yext/edward/services"

var _ services.Backend = &Backend{}

type Backend struct {
	Repository string            `json:"repository"`
	Tag        string            `json:"tag,omitempty"`
	Ports      map[string]string `json:"ports,omitempty"`
}

func (d *Backend) HasBuildStep() bool {
	return false
}

func (d *Backend) HasLaunchStep() bool {
	return true
}
