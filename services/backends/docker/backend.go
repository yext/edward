package docker

import (
	"github.com/yext/edward/services"
)

var _ services.Backend = &Backend{}

type Backend struct {
	Image           string     `json:"image"`
	Persistent      bool       `json:"persistent,omitempty"`
	ContainerConfig Config     `json:"containerConfig,omitempty"`
	HostConfig      HostConfig `json:"hostConfig,omitempty"`
}

func (d *Backend) HasBuildStep() bool {
	return false
}

func (d *Backend) HasLaunchStep() bool {
	return true
}
