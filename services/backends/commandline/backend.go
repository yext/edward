package commandline

import (
	"encoding/json"

	"github.com/pkg/errors"
	"github.com/yext/edward/services"
)

var _ services.Backend = &CommandLineBackend{}

type CommandLineBackend struct {
	// Commands for managing the service
	Commands ServiceConfigCommands `json:"commands"`

	// Checks to perform to ensure that a service has started correctly
	LaunchChecks *LaunchChecks `json:"launch_checks,omitempty"`
}

func (c *CommandLineBackend) UnmarshalJSON(data []byte) error {
	type Alias CommandLineBackend
	aux := &struct {
		*Alias
	}{
		Alias: (*Alias)(c),
	}
	if err := json.Unmarshal(data, &aux); err != nil {
		return errors.Wrap(err, "could not parse command line backend config")
	}
	if err := c.unmarshalLegacyLaunchChecks(data); err != nil {
		return errors.Wrap(err, "could not parse command line backend config")
	}

	if c.LaunchChecks != nil {
		checkCount := 0
		if len(c.LaunchChecks.LogText) > 0 {
			checkCount++
		}
		if len(c.LaunchChecks.Ports) > 0 {
			checkCount++
		}
		if c.LaunchChecks.Wait != 0 {
			checkCount++
		}
		if checkCount > 1 {
			return errors.New("cannot specify multiple launch check types for one service")
		}

	}
	return nil

}

func (c *CommandLineBackend) unmarshalLegacyLaunchChecks(data []byte) error {
	aux := &struct {
		Properties *ServiceConfigProperties `json:"log_properties,omitempty"`
	}{}
	if err := json.Unmarshal(data, &aux); err != nil {
		return errors.Wrap(err, "could not parse legacy properties")
	}
	if aux.Properties != nil {
		if c.LaunchChecks != nil {
			c.LaunchChecks.LogText = aux.Properties.Started
		} else {
			c.LaunchChecks = &LaunchChecks{
				LogText: aux.Properties.Started,
			}
		}
	}
	return nil
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

func (c *CommandLineBackend) Name() string {
	return "commandline"
}

func (c *CommandLineBackend) HasBuildStep() bool {
	return c.Commands.Build != ""
}

func (c *CommandLineBackend) HasLaunchStep() bool {
	return c.Commands.Launch != ""
}

func GetConfigCommandLine(s *services.ServiceConfig) (*CommandLineBackend, error) {
	if cl, ok := s.BackendConfig.(*CommandLineBackend); ok {
		return cl, nil
	}
	return nil, errors.New("service was not a command line service")
}
