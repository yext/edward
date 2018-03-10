package docker

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	"github.com/pkg/errors"
	"github.com/yext/edward/services"
)

var _ services.Backend = &Backend{}

type Backend struct {
	Image           string           `json:"image"`
	Persistent      bool             `json:"persistent,omitempty"`
	Ports           []*PortMapping   `json:"ports,omitempty"`
	ContainerConfig Config           `json:"containerConfig,omitempty"`
	HostConfig      HostConfig       `json:"hostConfig,omitempty"`
	NetworkConfig   NetworkingConfig `json:"networkConfig,omitempty"`
}

func (d *Backend) MarshalJSON() ([]byte, error) {
	if d == nil {
		return nil, nil
	}
	copy := *d
	type Alias Backend
	aux := &struct {
		*Alias
	}{
		Alias: (*Alias)(&copy),
	}

	for _, port := range d.Ports {
		if d.ContainerConfig.ExposedPorts != nil {
			if _, exists := d.ContainerConfig.ExposedPorts[port.ContainerPort]; exists {
				delete(d.ContainerConfig.ExposedPorts, port.ContainerPort)
			}
		}
		if d.HostConfig.PortBindings != nil {
			if _, exists := d.HostConfig.PortBindings[port.ContainerPort]; exists {
				delete(d.HostConfig.PortBindings, port.ContainerPort)
			}
		}
	}

	return json.Marshal(aux)
}

func (d *Backend) UnmarshalJSON(data []byte) error {
	type Alias Backend
	aux := &struct {
		*Alias
	}{
		Alias: (*Alias)(d),
	}
	if err := json.Unmarshal(data, &aux); err != nil {
		return errors.Wrap(err, "could not parse backend entry")
	}

	for _, port := range d.Ports {
		if d.ContainerConfig.ExposedPorts == nil {
			d.ContainerConfig.ExposedPorts = make(map[Port]struct{})
		}
		d.ContainerConfig.ExposedPorts[port.ContainerPort] = struct{}{}
		if d.HostConfig.PortBindings == nil {
			d.HostConfig.PortBindings = make(map[Port][]PortBinding)
		}
		d.HostConfig.PortBindings[port.ContainerPort] = []PortBinding{
			{
				HostPort: string(port.HostPort),
			},
		}
	}
	return nil
}

func (d *Backend) HasBuildStep() bool {
	return false
}

func (d *Backend) HasLaunchStep() bool {
	return true
}

type PortMapping struct {
	ContainerPort Port
	HostPort      Port

	Original string
}

func (p *PortMapping) MarshalJSON() ([]byte, error) {
	val := p.Original
	if val == "" {
		val = fmt.Sprintf("%v:%v", p.ContainerPort, p.HostPort)
	}
	return json.Marshal(val)
}

func (p *PortMapping) UnmarshalJSON(data []byte) error {
	var value string
	if err := json.Unmarshal(data, &value); err != nil {
		return errors.Wrap(err, "could not parse backend entry")
	}

	ports := strings.Split(value, ":")
	if len(ports) != 2 {
		return fmt.Errorf("%s is not a valid port mapping. Expected a container and host port separated by a colon. Example: 8080:8080.", value)
	}

	var err error
	p.ContainerPort, err = stringToPort(ports[0])
	if err != nil {
		return errors.WithStack(err)
	}
	p.HostPort, err = stringToPort(ports[1])
	if err != nil {
		return errors.WithStack(err)
	}

	p.Original = value

	return nil
}

func stringToPort(value string) (Port, error) {
	invalidErr := fmt.Errorf(
		"%s is not a valid port. Expected a number and optionally 'tcp' or 'udp'. Examples: '8080', '8080/tcp'",
		value,
	)
	segments := strings.Split(value, "/")
	if len(segments) > 2 {
		return Port(""), invalidErr
	}
	if len(segments) == 2 && segments[1] != "tcp" && segments[1] != "udp" {
		return Port(""), invalidErr
	}
	if _, err := strconv.Atoi(segments[0]); err != nil {
		return Port(""), invalidErr
	}
	if len(segments) == 1 {
		return Port(fmt.Sprintf("%s/tcp", value)), nil
	}
	return Port(value), nil
}
