package services

import (
	"crypto/sha1"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"path"
	"runtime"
	"strconv"

	"github.com/pkg/errors"
	"github.com/yext/edward/common"
	"github.com/yext/edward/warmup"
)

var _ ServiceOrGroup = &ServiceConfig{}

// ServiceConfig represents a service that can be managed by Edward
type ServiceConfig struct {
	// Service name, used to identify in commands
	Name string `json:"name"`
	// Alternative names for this service
	Aliases []string `json:"aliases,omitempty"`
	// Service description
	Description string `json:"description,omitempty"`
	// Optional path to service. If nil, uses cwd
	Path *string `json:"path,omitempty"`

	// Does this service require sudo privileges?
	RequiresSudo bool `json:"requiresSudo,omitempty"`

	// Checks to perform to ensure that a service has started correctly
	LaunchChecks *LaunchChecks `json:"launch_checks,omitempty"`

	// Env holds environment variables for a service, for example: GOPATH=~/gocode/
	// These will be added to the vars in the environment under which the Edward command was run
	Env []string `json:"env,omitempty"`

	Platform string `json:"platform,omitempty"`

	// Path to watch for updates, relative to config file. If specified, will enable hot reloading.
	WatchJSON json.RawMessage `json:"watch,omitempty"`

	// Action for warming up this service
	Warmup *warmup.Warmup `json:"warmup,omitempty"`

	// Path to config file from which this service was loaded
	// This may be the file that imported the config containing the service definition.
	ConfigFile string `json:"-"`

	// Logger for actions on this service
	Logger common.Logger `json:"-"`

	BackendConfig Backend `json:"-"`
}

// Matches returns true if the service name or an alias matches the provided name.
func (c *ServiceConfig) Matches(name string) bool {
	if c.Name == name {
		return true
	}
	for _, alias := range c.Aliases {
		if alias == name {
			return true
		}
	}
	return false
}

// UnmarshalJSON provides additional handling when unmarshaling a service from config.
// Currently, this handles legacy fields and fields with multiple possible types.
func (c *ServiceConfig) UnmarshalJSON(data []byte) error {
	type Alias ServiceConfig
	aux := &struct {
		*Alias
	}{
		Alias: (*Alias)(c),
	}
	if err := json.Unmarshal(data, &aux); err != nil {
		return errors.Wrap(err, "could not parse service config")
	}

	if err := c.unmarshalLegacyLaunchChecks(data); err != nil {
		return errors.WithStack(err)
	}

	if err := c.unmarshalType(data); err != nil {
		return errors.WithStack(err)
	}

	return errors.WithStack(c.validate())
}

func (c *ServiceConfig) MarshalJSON() ([]byte, error) {
	type Alias ServiceConfig
	aux := &struct {
		*Alias
	}{
		Alias: (*Alias)(c),
	}
	auxMap, err := toMapViaJson(aux)
	if err != nil {
		return nil, errors.WithMessage(err, "config")
	}
	typeMap, err := toMapViaJson(c.BackendConfig)
	if err != nil {
		return nil, errors.WithMessage(err, "type config")
	}

	for key, value := range typeMap {
		auxMap[key] = value
	}
	for typeName, loader := range loaders {
		if loader.Handles(c.BackendConfig) {
			auxMap["backend"] = typeName
		}
	}

	return json.Marshal(auxMap)
}

func toMapViaJson(v interface{}) (map[string]interface{}, error) {
	data, err := json.Marshal(v)
	if err != nil {
		return nil, errors.WithMessage(err, "initial marshal")
	}
	var dmap = make(map[string]interface{})
	err = json.Unmarshal(data, &dmap)
	if err != nil {
		return nil, errors.WithMessage(err, "unmarshalling to map")
	}
	return dmap, nil
}

func (c *ServiceConfig) unmarshalLegacyLaunchChecks(data []byte) error {
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

func (c *ServiceConfig) unmarshalType(data []byte) error {
	aux := &struct {
		// Backend of service, controlling how this service is built and launched.
		// Defaults to the command line type.
		Backend BackendName `json:"backend"`
	}{}
	if err := json.Unmarshal(data, &aux); err != nil {
		return errors.Wrap(err, "could not parse legacy properties")
	}
	if aux.Backend == "" {
		aux.Backend = defaultType
	}

	var (
		loader BackendLoader
		ok     bool
	)
	if loader, ok = loaders[aux.Backend]; !ok {
		return fmt.Errorf("unknown config type: %s", aux.Backend)
	}
	config := loader.New()
	if err := json.Unmarshal(data, config); err != nil {
		return errors.Wrap(err, fmt.Sprintf("could not parse config of type '%s'", aux.Backend))
	}
	c.BackendConfig = config
	return nil

}

// validate checks if this config is allowed
func (c *ServiceConfig) validate() error {
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

// SetWatch sets the watch configuration for this service
func (c *ServiceConfig) SetWatch(watch ServiceWatch) error {
	msg, err := json.Marshal(watch)
	if err != nil {
		return errors.WithStack(err)
	}
	c.WatchJSON = json.RawMessage(msg)
	return nil
}

// Watch returns the watch configuration for this service
func (c *ServiceConfig) Watch() ([]ServiceWatch, error) {
	var watch = ServiceWatch{
		Service: c,
	}

	if len(c.WatchJSON) == 0 {
		return nil, nil
	}

	var err error

	// Handle multiple
	err = json.Unmarshal(c.WatchJSON, &watch)
	if err == nil {
		return []ServiceWatch{watch}, nil
	}

	// Handle string version
	var include string
	err = json.Unmarshal(c.WatchJSON, &include)
	if err != nil {
		return nil, err
	}
	if include != "" {
		watch.IncludedPaths = append(watch.IncludedPaths, include)
		return []ServiceWatch{watch}, nil
	}

	return nil, nil
}

// ServiceWatch defines a set of directories to be watched for changes to a service's source.
type ServiceWatch struct {
	Service       *ServiceConfig `json:"-"`
	IncludedPaths []string       `json:"include,omitempty"`
	ExcludedPaths []string       `json:"exclude,omitempty"`
}

// MatchesPlatform determines whether or not this service can be run on the current OS
func (c *ServiceConfig) MatchesPlatform() bool {
	return len(c.Platform) == 0 || c.Platform == runtime.GOOS
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

// ServiceConfigProperties provides a set of regexes to detect properties of a service
// Deprecated: This has been dropped in favour of LaunchChecks
type ServiceConfigProperties struct {
	// Regex to detect a line indicating the service has started successfully
	Started string `json:"started,omitempty"`
	// Custom properties, mapping a property name to a regex
	Custom map[string]string `json:"-"`
}

// GetName returns the name for this service
func (c *ServiceConfig) GetName() string {
	return c.Name
}

// GetDescription returns the description for this service
func (c *ServiceConfig) GetDescription() string {
	return c.Description
}

// IsSudo returns true if this service requires sudo to run.
// If this service is excluded by cfg, then will always return false.
func (c *ServiceConfig) IsSudo(cfg OperationConfig) bool {
	if cfg.IsExcluded(c) {
		return false
	}

	return c.RequiresSudo
}

// GetRunLog returns the path to the run log for this service
func (c *ServiceConfig) GetRunLog(logDir string) string {
	return path.Join(logDir, c.Name+".log")
}

// IdentifyingFilename returns a filename that can be used to identify this service uniquely among all services
// that may be configured on a machine.
// The filename will be based on the service name and the path to its Edward config. It does not include an extension.
func (c *ServiceConfig) IdentifyingFilename() string {
	name := c.Name
	hasher := sha1.New()
	hasher.Write([]byte(c.ConfigFile))
	sha := base64.URLEncoding.EncodeToString(hasher.Sum(nil))
	return fmt.Sprintf("%v.%v", sha, name)
}

func (c *ServiceConfig) GetPid(pidFile string) (int, error) {
	dat, err := ioutil.ReadFile(pidFile)
	if err != nil {
		return 0, errors.WithStack(err)
	}
	pid, err := strconv.Atoi(string(dat))
	if err != nil {
		return 0, errors.WithStack(err)
	}
	return pid, nil
}

func (c *ServiceConfig) GetStateBase(stateDir string) string {
	return path.Join(stateDir, c.IdentifyingFilename())
}

func (c *ServiceConfig) GetStatePath(stateDir string) string {
	return fmt.Sprintf("%v.state", c.GetStateBase(stateDir))
}

func (c *ServiceConfig) GetPidPathLegacy(pidDir string) string {
	name := c.Name
	return path.Join(pidDir, fmt.Sprintf("%v.pid", name))
}
