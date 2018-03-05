package docker

import "time"

// Config is the list of configuration options used when creating a container.
// Config does not contain the options that are specific to starting a container on a
// given host.  Those are contained in HostConfig
type Config struct {
	Hostname          string            `json:"hostname,omitempty"`
	Domainname        string            `json:"domainname,omitempty"`
	User              string            `json:"user,omitempty"`
	Memory            int64             `json:"memory,omitempty"`
	MemorySwap        int64             `json:"memorySwap,omitempty"`
	MemoryReservation int64             `json:"memoryReservation,omitempty"`
	KernelMemory      int64             `json:"kernelMemory,omitempty"`
	CPUShares         int64             `json:"cpuShares,omitempty"`
	CPUSet            string            `json:"cpuset,omitempty"`
	PortSpecs         []string          `json:"portSpecs,omitempty"`
	ExposedPorts      map[Port]struct{} `json:"exposedPorts,omitempty"`
	PublishService    string            `json:"publishService,omitempty"`
	StopSignal        string            `json:"stopSignal,omitempty"`
	StopTimeout       int               `json:"stopTimeout,omitempty"`
	Env               []string          `json:"env,omitempty"`
	Cmd               []string          `json:"cmd"`
	Healthcheck       *HealthConfig     `json:"healthcheck,omitempty"`
	DNS               []string          `json:"dns,omitempty"` // For Docker API v1.9 and below only
	Volumes           []string          `json:"volumes,omitempty"`
	VolumeDriver      string            `json:"volumeDriver,omitempty"`
	WorkingDir        string            `json:"workingDir,omitempty"`
	MacAddress        string            `json:"macAddress,omitempty"`
	Entrypoint        []string          `json:"entrypoint"`
	SecurityOpts      []string          `json:"securityOpts,omitempty"`
	OnBuild           []string          `json:"onBuild,omitempty"`
	Mounts            []Mount           `json:"mounts,omitempty"`
	Labels            map[string]string `json:"labels,omitempty"`
	AttachStdin       bool              `json:"attachStdin,omitempty"`
	AttachStdout      bool              `json:"attachStdout,omitempty"`
	AttachStderr      bool              `json:"attachStderr,omitempty"`
	ArgsEscaped       bool              `json:"argsEscaped,omitempty"`
	Tty               bool              `json:"tty,omitempty"`
	OpenStdin         bool              `json:"openStdin,omitempty"`
	StdinOnce         bool              `json:"stdinOnce,omitempty"`
	NetworkDisabled   bool              `json:"networkDisabled,omitempty"`
}

// Port represents the port number and the protocol, in the form
// <number>/<protocol>. For example: 80/tcp.
type Port string

// HealthConfig holds configuration settings for the HEALTHCHECK feature
//
// It has been added in the version 1.24 of the Docker API, available since
// Docker 1.12.
type HealthConfig struct {
	// Test is the test to perform to check that the container is healthy.
	// An empty slice means to inherit the default.
	// The options are:
	// {} : inherit healthcheck
	// {"NONE"} : disable healthcheck
	// {"CMD", args...} : exec arguments directly
	// {"CMD-SHELL", command} : run command with system's default shell
	Test []string `json:"test,omitempty"`

	// Zero means to inherit. Durations are expressed as integer nanoseconds.
	Interval    time.Duration `json:"interval,omitempty"`    // Interval is the time to wait between checks.
	Timeout     time.Duration `json:"timeout,omitempty"`     // Timeout is the time to wait before considering the check to have hung.
	StartPeriod time.Duration `json:"startPeriod,omitempty"` // The start period for the container to initialize before the retries starts to count down.

	// Retries is the number of consecutive failures needed to consider a container as unhealthy.
	// Zero means inherit.
	Retries int `json:"retries,omitempty"`
}

// Mount represents a mount point in the container.
//
// It has been added in the version 1.20 of the Docker API, available since
// Docker 1.8.
type Mount struct {
	Name        string `json:"name"`
	Source      string `json:"source"`
	Destination string `json:"destination"`
	Driver      string `json:"driver"`
	Mode        string `json:"mode"`
	RW          bool   `json:"rw"`
}
