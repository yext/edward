package docker

// HostConfig contains the container options related to starting a container on
// a given host
type HostConfig struct {
	Binds                []string               `json:"binds,omitempty" yaml:"Binds,omitempty" toml:"Binds,omitempty"`
	CapAdd               []string               `json:"capAdd,omitempty" yaml:"CapAdd,omitempty" toml:"CapAdd,omitempty"`
	CapDrop              []string               `json:"capDrop,omitempty" yaml:"CapDrop,omitempty" toml:"CapDrop,omitempty"`
	GroupAdd             []string               `json:"groupAdd,omitempty" yaml:"GroupAdd,omitempty" toml:"GroupAdd,omitempty"`
	ContainerIDFile      string                 `json:"containerIDFile,omitempty" yaml:"ContainerIDFile,omitempty" toml:"ContainerIDFile,omitempty"`
	LxcConf              []KeyValuePair         `json:"lxcConf,omitempty" yaml:"LxcConf,omitempty" toml:"LxcConf,omitempty"`
	PortBindings         map[Port][]PortBinding `json:"portBindings,omitempty" yaml:"PortBindings,omitempty" toml:"PortBindings,omitempty"`
	Links                []string               `json:"links,omitempty" yaml:"Links,omitempty" toml:"Links,omitempty"`
	DNS                  []string               `json:"dns,omitempty" yaml:"Dns,omitempty" toml:"Dns,omitempty"` // For Docker API v1.10 and above only
	DNSOptions           []string               `json:"dnsOptions,omitempty" yaml:"DnsOptions,omitempty" toml:"DnsOptions,omitempty"`
	DNSSearch            []string               `json:"dnsSearch,omitempty" yaml:"DnsSearch,omitempty" toml:"DnsSearch,omitempty"`
	ExtraHosts           []string               `json:"extraHosts,omitempty" yaml:"ExtraHosts,omitempty" toml:"ExtraHosts,omitempty"`
	VolumesFrom          []string               `json:"volumesFrom,omitempty" yaml:"VolumesFrom,omitempty" toml:"VolumesFrom,omitempty"`
	UsernsMode           string                 `json:"usernsMode,omitempty" yaml:"UsernsMode,omitempty" toml:"UsernsMode,omitempty"`
	NetworkMode          string                 `json:"networkMode,omitempty" yaml:"NetworkMode,omitempty" toml:"NetworkMode,omitempty"`
	IpcMode              string                 `json:"ipcMode,omitempty" yaml:"IpcMode,omitempty" toml:"IpcMode,omitempty"`
	PidMode              string                 `json:"pidMode,omitempty" yaml:"PidMode,omitempty" toml:"PidMode,omitempty"`
	UTSMode              string                 `json:"utsMode,omitempty" yaml:"UTSMode,omitempty" toml:"UTSMode,omitempty"`
	RestartPolicy        RestartPolicy          `json:"restartPolicy,omitempty" yaml:"RestartPolicy,omitempty" toml:"RestartPolicy,omitempty"`
	Devices              []Device               `json:"devices,omitempty" yaml:"Devices,omitempty" toml:"Devices,omitempty"`
	DeviceCgroupRules    []string               `json:"deviceCgroupRules,omitempty" yaml:"DeviceCgroupRules,omitempty" toml:"DeviceCgroupRules,omitempty"`
	LogConfig            LogConfig              `json:"logConfig,omitempty" yaml:"LogConfig,omitempty" toml:"LogConfig,omitempty"`
	SecurityOpt          []string               `json:"securityOpt,omitempty" yaml:"SecurityOpt,omitempty" toml:"SecurityOpt,omitempty"`
	Cgroup               string                 `json:"cgroup,omitempty" yaml:"Cgroup,omitempty" toml:"Cgroup,omitempty"`
	CgroupParent         string                 `json:"cgroupParent,omitempty" yaml:"CgroupParent,omitempty" toml:"CgroupParent,omitempty"`
	Memory               int64                  `json:"memory,omitempty" yaml:"Memory,omitempty" toml:"Memory,omitempty"`
	MemoryReservation    int64                  `json:"memoryReservation,omitempty" yaml:"MemoryReservation,omitempty" toml:"MemoryReservation,omitempty"`
	KernelMemory         int64                  `json:"kernelMemory,omitempty" yaml:"KernelMemory,omitempty" toml:"KernelMemory,omitempty"`
	MemorySwap           int64                  `json:"memorySwap,omitempty" yaml:"MemorySwap,omitempty" toml:"MemorySwap,omitempty"`
	MemorySwappiness     int64                  `json:"memorySwappiness,omitempty" yaml:"MemorySwappiness,omitempty" toml:"MemorySwappiness,omitempty"`
	CPUShares            int64                  `json:"cpuShares,omitempty" yaml:"CpuShares,omitempty" toml:"CpuShares,omitempty"`
	CPUSet               string                 `json:"cpuset,omitempty" yaml:"Cpuset,omitempty" toml:"Cpuset,omitempty"`
	CPUSetCPUs           string                 `json:"cpusetCpus,omitempty" yaml:"CpusetCpus,omitempty" toml:"CpusetCpus,omitempty"`
	CPUSetMEMs           string                 `json:"cpusetMems,omitempty" yaml:"CpusetMems,omitempty" toml:"CpusetMems,omitempty"`
	CPUQuota             int64                  `json:"cpuQuota,omitempty" yaml:"CpuQuota,omitempty" toml:"CpuQuota,omitempty"`
	CPUPeriod            int64                  `json:"cpuPeriod,omitempty" yaml:"CpuPeriod,omitempty" toml:"CpuPeriod,omitempty"`
	CPURealtimePeriod    int64                  `json:"cpuRealtimePeriod,omitempty" yaml:"CpuRealtimePeriod,omitempty" toml:"CpuRealtimePeriod,omitempty"`
	CPURealtimeRuntime   int64                  `json:"cpuRealtimeRuntime,omitempty" yaml:"CpuRealtimeRuntime,omitempty" toml:"CpuRealtimeRuntime,omitempty"`
	BlkioWeight          int64                  `json:"blkioWeight,omitempty" yaml:"BlkioWeight,omitempty" toml:"BlkioWeight,omitempty"`
	BlkioWeightDevice    []BlockWeight          `json:"blkioWeightDevice,omitempty" yaml:"BlkioWeightDevice,omitempty" toml:"BlkioWeightDevice,omitempty"`
	BlkioDeviceReadBps   []BlockLimit           `json:"blkioDeviceReadBps,omitempty" yaml:"BlkioDeviceReadBps,omitempty" toml:"BlkioDeviceReadBps,omitempty"`
	BlkioDeviceReadIOps  []BlockLimit           `json:"blkioDeviceReadIOps,omitempty" yaml:"BlkioDeviceReadIOps,omitempty" toml:"BlkioDeviceReadIOps,omitempty"`
	BlkioDeviceWriteBps  []BlockLimit           `json:"blkioDeviceWriteBps,omitempty" yaml:"BlkioDeviceWriteBps,omitempty" toml:"BlkioDeviceWriteBps,omitempty"`
	BlkioDeviceWriteIOps []BlockLimit           `json:"blkioDeviceWriteIOps,omitempty" yaml:"BlkioDeviceWriteIOps,omitempty" toml:"BlkioDeviceWriteIOps,omitempty"`
	Ulimits              []ULimit               `json:"ulimits,omitempty" yaml:"Ulimits,omitempty" toml:"Ulimits,omitempty"`
	VolumeDriver         string                 `json:"volumeDriver,omitempty" yaml:"VolumeDriver,omitempty" toml:"VolumeDriver,omitempty"`
	OomScoreAdj          int                    `json:"oomScoreAdj,omitempty" yaml:"OomScoreAdj,omitempty" toml:"OomScoreAdj,omitempty"`
	PidsLimit            int64                  `json:"pidsLimit,omitempty" yaml:"PidsLimit,omitempty" toml:"PidsLimit,omitempty"`
	ShmSize              int64                  `json:"shmSize,omitempty" yaml:"ShmSize,omitempty" toml:"ShmSize,omitempty"`
	Tmpfs                map[string]string      `json:"tmpfs,omitempty" yaml:"Tmpfs,omitempty" toml:"Tmpfs,omitempty"`
	Privileged           bool                   `json:"privileged,omitempty" yaml:"Privileged,omitempty" toml:"Privileged,omitempty"`
	PublishAllPorts      bool                   `json:"publishAllPorts,omitempty" yaml:"PublishAllPorts,omitempty" toml:"PublishAllPorts,omitempty"`
	ReadonlyRootfs       bool                   `json:"readonlyRootfs,omitempty" yaml:"ReadonlyRootfs,omitempty" toml:"ReadonlyRootfs,omitempty"`
	OOMKillDisable       bool                   `json:"oomKillDisable,omitempty" yaml:"OomKillDisable,omitempty" toml:"OomKillDisable,omitempty"`
	AutoRemove           bool                   `json:"autoRemove,omitempty" yaml:"AutoRemove,omitempty" toml:"AutoRemove,omitempty"`
	StorageOpt           map[string]string      `json:"storageOpt,omitempty" yaml:"StorageOpt,omitempty" toml:"StorageOpt,omitempty"`
	Sysctls              map[string]string      `json:"sysctls,omitempty" yaml:"Sysctls,omitempty" toml:"Sysctls,omitempty"`
	CPUCount             int64                  `json:"cpuCount,omitempty" yaml:"CpuCount,omitempty"`
	CPUPercent           int64                  `json:"cpuPercent,omitempty" yaml:"CpuPercent,omitempty"`
	IOMaximumBandwidth   int64                  `json:"ioMaximumBandwidth,omitempty" yaml:"IOMaximumBandwidth,omitempty"`
	IOMaximumIOps        int64                  `json:"ioMaximumIOps,omitempty" yaml:"IOMaximumIOps,omitempty"`
	Mounts               []HostMount            `json:"mounts,omitempty" yaml:"Mounts,omitempty" toml:"Mounts,omitempty"`
}

// KeyValuePair is a type for generic key/value pairs as used in the Lxc
// configuration
type KeyValuePair struct {
	Key   string `json:"key,omitempty" yaml:"Key,omitempty" toml:"Key,omitempty"`
	Value string `json:"value,omitempty" yaml:"Value,omitempty" toml:"Value,omitempty"`
}

// PortBinding represents the host/container port mapping as returned in the
// `docker inspect` json
type PortBinding struct {
	HostIP   string `json:"hostIp,omitempty" yaml:"HostIp,omitempty" toml:"HostIp,omitempty"`
	HostPort string `json:"hostPort,omitempty" yaml:"HostPort,omitempty" toml:"HostPort,omitempty"`
}

// RestartPolicy represents the policy for automatically restarting a container.
//
// Possible values are:
//
//   - always: the docker daemon will always restart the container
//   - on-failure: the docker daemon will restart the container on failures, at
//                 most MaximumRetryCount times
//   - unless-stopped: the docker daemon will always restart the container except
//                 when user has manually stopped the container
//   - no: the docker daemon will not restart the container automatically
type RestartPolicy struct {
	Name              string `json:"name,omitempty" yaml:"Name,omitempty" toml:"Name,omitempty"`
	MaximumRetryCount int    `json:"maximumRetryCount,omitempty" yaml:"MaximumRetryCount,omitempty" toml:"MaximumRetryCount,omitempty"`
}

// Device represents a device mapping between the Docker host and the
// container.
type Device struct {
	PathOnHost        string `json:"pathOnHost,omitempty" yaml:"PathOnHost,omitempty" toml:"PathOnHost,omitempty"`
	PathInContainer   string `json:"pathInContainer,omitempty" yaml:"PathInContainer,omitempty" toml:"PathInContainer,omitempty"`
	CgroupPermissions string `json:"cgroupPermissions,omitempty" yaml:"CgroupPermissions,omitempty" toml:"CgroupPermissions,omitempty"`
}

// LogConfig defines the log driver type and the configuration for it.
type LogConfig struct {
	Type   string            `json:"type,omitempty" yaml:"Type,omitempty" toml:"Type,omitempty"`
	Config map[string]string `json:"config,omitempty" yaml:"Config,omitempty" toml:"Config,omitempty"`
}

// BlockWeight represents a relative device weight for an individual device inside
// of a container
type BlockWeight struct {
	Path   string `json:"path,omitempty"`
	Weight string `json:"weight,omitempty"`
}

// BlockLimit represents a read/write limit in IOPS or Bandwidth for a device
// inside of a container
type BlockLimit struct {
	Path string `json:"path,omitempty"`
	Rate int64  `json:"rate,omitempty"`
}

// ULimit defines system-wide resource limitations This can help a lot in
// system administration, e.g. when a user starts too many processes and
// therefore makes the system unresponsive for other users.
type ULimit struct {
	Name string `json:"name,omitempty" yaml:"Name,omitempty" toml:"Name,omitempty"`
	Soft int64  `json:"soft,omitempty" yaml:"Soft,omitempty" toml:"Soft,omitempty"`
	Hard int64  `json:"hard,omitempty" yaml:"Hard,omitempty" toml:"Hard,omitempty"`
}

// HostMount represents a mount point in the container in HostConfig.
//
// It has been added in the version 1.25 of the Docker API
type HostMount struct {
	Target        string         `json:"target,omitempty" yaml:"Target,omitempty" toml:"Target,omitempty"`
	Source        string         `json:"source,omitempty" yaml:"Source,omitempty" toml:"Source,omitempty"`
	Type          string         `json:"type,omitempty" yaml:"Type,omitempty" toml:"Type,omitempty"`
	ReadOnly      bool           `json:"readOnly,omitempty" yaml:"ReadOnly,omitempty" toml:"ReadOnly,omitempty"`
	BindOptions   *BindOptions   `json:"bindOptions,omitempty" yaml:"BindOptions,omitempty" toml:"BindOptions,omitempty"`
	VolumeOptions *VolumeOptions `json:"volumeOptions,omitempty" yaml:"VolumeOptions,omitempty" toml:"VolumeOptions,omitempty"`
	TempfsOptions *TempfsOptions `json:"tempfsOptions,omitempty" yaml:"TempfsOptions,omitempty" toml:"TempfsOptions,omitempty"`
}

// BindOptions contains optional configuration for the bind type
type BindOptions struct {
	Propagation string `json:"propagation,omitempty" yaml:"Propagation,omitempty" toml:"Propagation,omitempty"`
}

// VolumeOptions contains optional configuration for the volume type
type VolumeOptions struct {
	NoCopy       bool               `json:"noCopy,omitempty" yaml:"NoCopy,omitempty" toml:"NoCopy,omitempty"`
	Labels       map[string]string  `json:"labels,omitempty" yaml:"Labels,omitempty" toml:"Labels,omitempty"`
	DriverConfig VolumeDriverConfig `json:"driverConfig,omitempty" yaml:"DriverConfig,omitempty" toml:"DriverConfig,omitempty"`
}

// TempfsOptions contains optional configuration for the tempfs type
type TempfsOptions struct {
	SizeBytes int64 `json:"sizeBytes,omitempty" yaml:"SizeBytes,omitempty" toml:"SizeBytes,omitempty"`
	Mode      int   `json:"mode,omitempty" yaml:"Mode,omitempty" toml:"Mode,omitempty"`
}

// VolumeDriverConfig holds a map of volume driver specific options
type VolumeDriverConfig struct {
	Name    string            `json:"name,omitempty" yaml:"Name,omitempty" toml:"Name,omitempty"`
	Options map[string]string `json:"options,omitempty" yaml:"Options,omitempty" toml:"Options,omitempty"`
}
