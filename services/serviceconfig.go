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
	"time"

	"github.com/pkg/errors"
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

	Backends []*BackendConfig `json:"backends"`

	// Timeout for terminating a service runner. If termination has not completed after this amount
	// of time, the runner will be killed.
	TerminationTimeout *Duration `json:"terminationTimeout,omitempty"`
}

// GetTerminationTimeout returns the timeout for termination, if no timeout is set, the
// default of 30s will be returned
func (c *ServiceConfig) GetTerminationTimeout() time.Duration {
	if c.TerminationTimeout == nil {
		return 30 * time.Second
	}
	return c.TerminationTimeout.Duration
}

// Backend returns the default backend for this service
func (c *ServiceConfig) Backend() Backend {
	for _, backendConfig := range c.Backends {
		return backendConfig.Config
	}
	return nil
}

// BackendConfig provides backend configuration for json
type BackendConfig struct {
	Name string `json:"name"`
	Type string `json:"type"`

	Config Backend `json:"-"`
}

var _ json.Marshaler = &BackendConfig{}
var _ json.Unmarshaler = &BackendConfig{}

func (c *BackendConfig) UnmarshalJSON(data []byte) error {
	type Alias BackendConfig
	aux := &struct {
		*Alias
	}{
		Alias: (*Alias)(c),
	}
	if err := json.Unmarshal(data, &aux); err != nil {
		return errors.Wrap(err, "could not parse backend entry")
	}
	var (
		loader BackendLoader
		ok     bool
	)
	if loader, ok = loaders[aux.Type]; !ok {
		return fmt.Errorf("unknown config type: %s", aux.Type)
	}
	c.Config = loader.New()
	if err := json.Unmarshal(data, &c.Config); err != nil {
		return errors.Wrap(err, "could not parse backend config")
	}

	return nil
}

func (c *BackendConfig) MarshalJSON() ([]byte, error) {
	if c.Type == "" {
		return nil, errors.New("no type specified for backend")
	}

	data, err := json.Marshal(c.Config)
	if err != nil {
		return nil, errors.WithStack(err)
	}

	var m = make(map[string]interface{})
	err = json.Unmarshal(data, &m)
	if err != nil {
		return nil, errors.WithStack(err)
	}

	if m == nil {
		m = make(map[string]interface{})
	}
	m["name"] = c.Name
	m["type"] = c.Type

	d, err := json.Marshal(&m)
	if err != nil {
		return nil, errors.WithStack(err)
	}
	return d, nil
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
	if err := json.Unmarshal(data, aux); err != nil {
		return errors.Wrap(err, "could not parse service config")
	}
	for _, m := range legacyUnmarshalers {
		if err := m.Unmarshal(data, c); err != nil {
			return errors.Wrap(err, "could not parse legacy configuration")
		}
	}

	if len(c.Backends) == 0 {
		return errors.New("no backends specified in config")
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
	return c.IdentifyingFilenameWithEncoding(base64.URLEncoding)
}

// IdentifyingFilenameWithEncoding is equivalent to IdentifyingFilenameWithEncoding
func (c *ServiceConfig) IdentifyingFilenameWithEncoding(encoding *base64.Encoding) string {
	name := c.Name
	sha := encoding.EncodeToString(c.configHash())
	return fmt.Sprintf("%v.%v", sha, name)
}

// configHash returns a sha1 hash representing the config file for this service
func (c *ServiceConfig) configHash() []byte {
	hasher := sha1.New()
	hasher.Write([]byte(c.ConfigFile))
	return hasher.Sum(nil)
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
