package fake

import (
	"github.com/yext/edward/services"
)

var _ services.Backend = &Backend{}

type Backend struct {
}

// ServiceConfigProperties provides a set of regexes to detect properties of a service
// Deprecated: This has been dropped in favour of LaunchChecks
type ServiceConfigProperties struct {
	// Regex to detect a line indicating the service has started successfully
	Started string `json:"started,omitempty"`
	// Custom properties, mapping a property name to a regex
	Custom map[string]string `json:"-"`
}

// LaunchChecks defines the mechanism for testing whether a service has started successfully
type LaunchChecks struct {
	// A string to look for in the service's logs that indicates it has completed startup.
	LogText string `json:"log_text,omitempty"`
	// One or more specific ports that are expected to be opened when this service starts.
	Ports []int `json:"ports,omitempty"`
	// Wait for a specified amount of time (in ms) before calling the service started if still running.
	Wait int64 `json:"wait,omitempty"`
}

// ServiceConfigCommands define the commands for building, launching and stopping a service
// All commands are optional
type ServiceConfigCommands struct {
	// Command to build
	Build string `json:"build,omitempty"`
	// Command to launch
	Launch string `json:"launch,omitempty"`
	// Optional command to stop
	Stop string `json:"stop,omitempty"`
}

func (c *Backend) Name() string {
	return "fake"
}

func (c *Backend) HasBuildStep() bool {
	return false
}

func (c *Backend) HasLaunchStep() bool {
	return false
}
