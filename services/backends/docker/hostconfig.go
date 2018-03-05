package docker

// HostConfig contains the container options related to starting a container on
// a given host
type HostConfig struct {
	Binds                []string               `json:"binds,omitempty"`
	CapAdd               []string               `json:"capAdd,omitempty"`
	CapDrop              []string               `json:"capDrop,omitempty"`
	GroupAdd             []string               `json:"groupAdd,omitempty"`
	ContainerIDFile      string                 `json:"containerIDFile,omitempty"`
	LxcConf              []KeyValuePair         `json:"lxcConf,omitempty"`
	PortBindings         map[Port][]PortBinding `json:"portBindings,omitempty"`
	Links                []string               `json:"links,omitempty"`
	DNS                  []string               `json:"dns,omitempty"` // For Docker API v1.10 and above only
	DNSOptions           []string               `json:"dnsOptions,omitempty"`
	DNSSearch            []string               `json:"dnsSearch,omitempty"`
	ExtraHosts           []string               `json:"extraHosts,omitempty"`
	VolumesFrom          []string               `json:"volumesFrom,omitempty"`
	UsernsMode           string                 `json:"usernsMode,omitempty"`
	NetworkMode          string                 `json:"networkMode,omitempty"`
	IpcMode              string                 `json:"ipcMode,omitempty"`
	PidMode              string                 `json:"pidMode,omitempty"`
	UTSMode              string                 `json:"utsMode,omitempty"`
	RestartPolicy        RestartPolicy          `json:"restartPolicy,omitempty"`
	Devices              []Device               `json:"devices,omitempty"`
	DeviceCgroupRules    []string               `json:"deviceCgroupRules,omitempty"`
	LogConfig            LogConfig              `json:"logConfig,omitempty"`
	SecurityOpt          []string               `json:"securityOpt,omitempty"`
	Cgroup               string                 `json:"cgroup,omitempty"`
	CgroupParent         string                 `json:"cgroupParent,omitempty"`
	Memory               int64                  `json:"memory,omitempty"`
	MemoryReservation    int64                  `json:"memoryReservation,omitempty"`
	KernelMemory         int64                  `json:"kernelMemory,omitempty"`
	MemorySwap           int64                  `json:"memorySwap,omitempty"`
	MemorySwappiness     int64                  `json:"memorySwappiness,omitempty"`
	CPUShares            int64                  `json:"cpuShares,omitempty"`
	CPUSet               string                 `json:"cpuset,omitempty"`
	CPUSetCPUs           string                 `json:"cpusetCpus,omitempty"`
	CPUSetMEMs           string                 `json:"cpusetMems,omitempty"`
	CPUQuota             int64                  `json:"cpuQuota,omitempty"`
	CPUPeriod            int64                  `json:"cpuPeriod,omitempty"`
	CPURealtimePeriod    int64                  `json:"cpuRealtimePeriod,omitempty"`
	CPURealtimeRuntime   int64                  `json:"cpuRealtimeRuntime,omitempty"`
	BlkioWeight          int64                  `json:"blkioWeight,omitempty"`
	BlkioWeightDevice    []BlockWeight          `json:"blkioWeightDevice,omitempty"`
	BlkioDeviceReadBps   []BlockLimit           `json:"blkioDeviceReadBps,omitempty"`
	BlkioDeviceReadIOps  []BlockLimit           `json:"blkioDeviceReadIOps,omitempty"`
	BlkioDeviceWriteBps  []BlockLimit           `json:"blkioDeviceWriteBps,omitempty"`
	BlkioDeviceWriteIOps []BlockLimit           `json:"blkioDeviceWriteIOps,omitempty"`
	Ulimits              []ULimit               `json:"ulimits,omitempty"`
	VolumeDriver         string                 `json:"volumeDriver,omitempty"`
	OomScoreAdj          int                    `json:"oomScoreAdj,omitempty"`
	PidsLimit            int64                  `json:"pidsLimit,omitempty"`
	ShmSize              int64                  `json:"shmSize,omitempty"`
	Tmpfs                map[string]string      `json:"tmpfs,omitempty"`
	Privileged           bool                   `json:"privileged,omitempty"`
	PublishAllPorts      bool                   `json:"publishAllPorts,omitempty"`
	ReadonlyRootfs       bool                   `json:"readonlyRootfs,omitempty"`
	OOMKillDisable       bool                   `json:"oomKillDisable,omitempty"`
	AutoRemove           bool                   `json:"autoRemove,omitempty"`
	StorageOpt           map[string]string      `json:"storageOpt,omitempty"`
	Sysctls              map[string]string      `json:"sysctls,omitempty"`
	CPUCount             int64                  `json:"cpuCount,omitempty"`
	CPUPercent           int64                  `json:"cpuPercent,omitempty"`
	IOMaximumBandwidth   int64                  `json:"ioMaximumBandwidth,omitempty"`
	IOMaximumIOps        int64                  `json:"ioMaximumIOps,omitempty"`
	Mounts               []HostMount            `json:"mounts,omitempty"`
}

// KeyValuePair is a type for generic key/value pairs as used in the Lxc
// configuration
type KeyValuePair struct {
	Key   string `json:"key,omitempty"`
	Value string `json:"value,omitempty"`
}

// PortBinding represents the host/container port mapping as returned in the
// `docker inspect` json
type PortBinding struct {
	HostIP   string `json:"hostIp,omitempty"`
	HostPort string `json:"hostPort,omitempty"`
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
	Name              string `json:"name,omitempty"`
	MaximumRetryCount int    `json:"maximumRetryCount,omitempty"`
}

// Device represents a device mapping between the Docker host and the
// container.
type Device struct {
	PathOnHost        string `json:"pathOnHost,omitempty"`
	PathInContainer   string `json:"pathInContainer,omitempty"`
	CgroupPermissions string `json:"cgroupPermissions,omitempty"`
}

// LogConfig defines the log driver type and the configuration for it.
type LogConfig struct {
	Type   string            `json:"type,omitempty"`
	Config map[string]string `json:"config,omitempty"`
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
	Name string `json:"name,omitempty"`
	Soft int64  `json:"soft,omitempty"`
	Hard int64  `json:"hard,omitempty"`
}

// HostMount represents a mount point in the container in HostConfig.
//
// It has been added in the version 1.25 of the Docker API
type HostMount struct {
	Target        string         `json:"target,omitempty"`
	Source        string         `json:"source,omitempty"`
	Type          string         `json:"type,omitempty"`
	ReadOnly      bool           `json:"readOnly,omitempty"`
	BindOptions   *BindOptions   `json:"bindOptions,omitempty"`
	VolumeOptions *VolumeOptions `json:"volumeOptions,omitempty"`
	TempfsOptions *TempfsOptions `json:"tempfsOptions,omitempty"`
}

// BindOptions contains optional configuration for the bind type
type BindOptions struct {
	Propagation string `json:"propagation,omitempty"`
}

// VolumeOptions contains optional configuration for the volume type
type VolumeOptions struct {
	NoCopy       bool               `json:"noCopy,omitempty"`
	Labels       map[string]string  `json:"labels,omitempty"`
	DriverConfig VolumeDriverConfig `json:"driverConfig,omitempty"`
}

// TempfsOptions contains optional configuration for the tempfs type
type TempfsOptions struct {
	SizeBytes int64 `json:"sizeBytes,omitempty"`
	Mode      int   `json:"mode,omitempty"`
}

// VolumeDriverConfig holds a map of volume driver specific options
type VolumeDriverConfig struct {
	Name    string            `json:"name,omitempty"`
	Options map[string]string `json:"options,omitempty"`
}
