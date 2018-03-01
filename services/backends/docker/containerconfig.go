package docker

import "time"

// Config is the list of configuration options used when creating a container.
// Config does not contain the options that are specific to starting a container on a
// given host.  Those are contained in HostConfig
type Config struct {
	Hostname          string            `json:"hostname,omitempty" yaml:"Hostname,omitempty" toml:"Hostname,omitempty"`
	Domainname        string            `json:"domainname,omitempty" yaml:"Domainname,omitempty" toml:"Domainname,omitempty"`
	User              string            `json:"user,omitempty" yaml:"User,omitempty" toml:"User,omitempty"`
	Memory            int64             `json:"memory,omitempty" yaml:"Memory,omitempty" toml:"Memory,omitempty"`
	MemorySwap        int64             `json:"memorySwap,omitempty" yaml:"MemorySwap,omitempty" toml:"MemorySwap,omitempty"`
	MemoryReservation int64             `json:"memoryReservation,omitempty" yaml:"MemoryReservation,omitempty" toml:"MemoryReservation,omitempty"`
	KernelMemory      int64             `json:"kernelMemory,omitempty" yaml:"KernelMemory,omitempty" toml:"KernelMemory,omitempty"`
	CPUShares         int64             `json:"cpuShares,omitempty" yaml:"CpuShares,omitempty" toml:"CpuShares,omitempty"`
	CPUSet            string            `json:"cpuset,omitempty" yaml:"Cpuset,omitempty" toml:"Cpuset,omitempty"`
	PortSpecs         []string          `json:"portSpecs,omitempty" yaml:"PortSpecs,omitempty" toml:"PortSpecs,omitempty"`
	ExposedPorts      map[Port]struct{} `json:"exposedPorts,omitempty" yaml:"ExposedPorts,omitempty" toml:"ExposedPorts,omitempty"`
	PublishService    string            `json:"publishService,omitempty" yaml:"PublishService,omitempty" toml:"PublishService,omitempty"`
	StopSignal        string            `json:"stopSignal,omitempty" yaml:"StopSignal,omitempty" toml:"StopSignal,omitempty"`
	StopTimeout       int               `json:"stopTimeout,omitempty" yaml:"StopTimeout,omitempty" toml:"StopTimeout,omitempty"`
	Env               []string          `json:"env,omitempty" yaml:"Env,omitempty" toml:"Env,omitempty"`
	Cmd               []string          `json:"cmd" yaml:"Cmd" toml:"Cmd"`
	Healthcheck       *HealthConfig     `json:"healthcheck,omitempty" yaml:"Healthcheck,omitempty" toml:"Healthcheck,omitempty"`
	DNS               []string          `json:"dns,omitempty" yaml:"Dns,omitempty" toml:"Dns,omitempty"` // For Docker API v1.9 and below only
	Volumes           []string          `json:"volumes,omitempty" yaml:"Volumes,omitempty" toml:"Volumes,omitempty"`
	VolumeDriver      string            `json:"volumeDriver,omitempty" yaml:"VolumeDriver,omitempty" toml:"VolumeDriver,omitempty"`
	WorkingDir        string            `json:"workingDir,omitempty" yaml:"WorkingDir,omitempty" toml:"WorkingDir,omitempty"`
	MacAddress        string            `json:"macAddress,omitempty" yaml:"MacAddress,omitempty" toml:"MacAddress,omitempty"`
	Entrypoint        []string          `json:"entrypoint" yaml:"Entrypoint" toml:"Entrypoint"`
	SecurityOpts      []string          `json:"securityOpts,omitempty" yaml:"SecurityOpts,omitempty" toml:"SecurityOpts,omitempty"`
	OnBuild           []string          `json:"onBuild,omitempty" yaml:"OnBuild,omitempty" toml:"OnBuild,omitempty"`
	Mounts            []Mount           `json:"mounts,omitempty" yaml:"Mounts,omitempty" toml:"Mounts,omitempty"`
	Labels            map[string]string `json:"labels,omitempty" yaml:"Labels,omitempty" toml:"Labels,omitempty"`
	AttachStdin       bool              `json:"attachStdin,omitempty" yaml:"AttachStdin,omitempty" toml:"AttachStdin,omitempty"`
	AttachStdout      bool              `json:"attachStdout,omitempty" yaml:"AttachStdout,omitempty" toml:"AttachStdout,omitempty"`
	AttachStderr      bool              `json:"attachStderr,omitempty" yaml:"AttachStderr,omitempty" toml:"AttachStderr,omitempty"`
	ArgsEscaped       bool              `json:"argsEscaped,omitempty" yaml:"ArgsEscaped,omitempty" toml:"ArgsEscaped,omitempty"`
	Tty               bool              `json:"tty,omitempty" yaml:"Tty,omitempty" toml:"Tty,omitempty"`
	OpenStdin         bool              `json:"openStdin,omitempty" yaml:"OpenStdin,omitempty" toml:"OpenStdin,omitempty"`
	StdinOnce         bool              `json:"stdinOnce,omitempty" yaml:"StdinOnce,omitempty" toml:"StdinOnce,omitempty"`
	NetworkDisabled   bool              `json:"networkDisabled,omitempty" yaml:"NetworkDisabled,omitempty" toml:"NetworkDisabled,omitempty"`
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
	Test []string `json:"test,omitempty" yaml:"Test,omitempty" toml:"Test,omitempty"`

	// Zero means to inherit. Durations are expressed as integer nanoseconds.
	Interval    time.Duration `json:"interval,omitempty" yaml:"Interval,omitempty" toml:"Interval,omitempty"`          // Interval is the time to wait between checks.
	Timeout     time.Duration `json:"timeout,omitempty" yaml:"Timeout,omitempty" toml:"Timeout,omitempty"`             // Timeout is the time to wait before considering the check to have hung.
	StartPeriod time.Duration `json:"startPeriod,omitempty" yaml:"StartPeriod,omitempty" toml:"StartPeriod,omitempty"` // The start period for the container to initialize before the retries starts to count down.

	// Retries is the number of consecutive failures needed to consider a container as unhealthy.
	// Zero means inherit.
	Retries int `json:"retries,omitempty" yaml:"Retries,omitempty" toml:"Retries,omitempty"`
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
