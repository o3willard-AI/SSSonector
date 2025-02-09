package daemon

// NamespaceConfig represents namespace configuration
type NamespaceConfig struct {
	Network NetworkNamespaceConfig
	Mount   MountNamespaceConfig
	PID     PIDNamespaceConfig
	User    UserNamespaceConfig
	UTS     UTSNamespaceConfig
	IPC     IPCNamespaceConfig
}

// NetworkNamespaceConfig represents network namespace configuration
type NetworkNamespaceConfig struct {
	Enabled bool   // Whether to use network namespace
	Type    string // Type of network namespace (private, host, container)
	Name    string // Name of network namespace (for container type)
}

// MountNamespaceConfig represents mount namespace configuration
type MountNamespaceConfig struct {
	Enabled bool     // Whether to use mount namespace
	Mounts  []Mount  // Mount points
	Tmpfs   []Tmpfs  // Tmpfs mounts
	Bind    []Bind   // Bind mounts
	Symlink []string // Symlinks to create
}

// PIDNamespaceConfig represents PID namespace configuration
type PIDNamespaceConfig struct {
	Enabled bool // Whether to use PID namespace
	Init    bool // Whether to run init process
}

// UserNamespaceConfig represents user namespace configuration
type UserNamespaceConfig struct {
	Enabled    bool // Whether to use user namespace
	UIDMap     []IDMap
	GIDMap     []IDMap
	NoNewPrivs bool // Disable gaining new privileges
}

// UTSNamespaceConfig represents UTS namespace configuration
type UTSNamespaceConfig struct {
	Enabled    bool   // Whether to use UTS namespace
	Hostname   string // Hostname in namespace
	Domainname string // Domain name in namespace
}

// IPCNamespaceConfig represents IPC namespace configuration
type IPCNamespaceConfig struct {
	Enabled bool // Whether to use IPC namespace
	Private bool // Whether to use private IPC namespace
}

// Mount represents a mount point
type Mount struct {
	Source string // Source path
	Target string // Target path
	Type   string // Mount type (proc, sysfs, etc.)
	Flags  uint   // Mount flags
	Data   string // Mount data
}

// Tmpfs represents a tmpfs mount
type Tmpfs struct {
	Target string // Target path
	Size   string // Size limit
	Mode   uint   // Permission mode
}

// Bind represents a bind mount
type Bind struct {
	Source string // Source path
	Target string // Target path
	RO     bool   // Read-only mount
}

// IDMap represents a UID/GID mapping
type IDMap struct {
	ContainerID int // ID in container
	HostID      int // ID on host
	Size        int // Range size
}

// GetDefaultNamespaceConfig returns default namespace configuration
func GetDefaultNamespaceConfig() *NamespaceConfig {
	return &NamespaceConfig{
		Network: NetworkNamespaceConfig{
			Enabled: true,
			Type:    "private",
		},
		Mount: MountNamespaceConfig{
			Enabled: true,
			Mounts: []Mount{
				{
					Source: "proc",
					Target: "/proc",
					Type:   "proc",
				},
				{
					Source: "sysfs",
					Target: "/sys",
					Type:   "sysfs",
					Flags:  0x4, // MS_NOSUID
				},
				{
					Source: "devtmpfs",
					Target: "/dev",
					Type:   "devtmpfs",
					Flags:  0x4, // MS_NOSUID
				},
			},
			Tmpfs: []Tmpfs{
				{
					Target: "/tmp",
					Size:   "10%",
					Mode:   0o1777,
				},
				{
					Target: "/run",
					Size:   "10%",
					Mode:   0o0755,
				},
			},
		},
		PID: PIDNamespaceConfig{
			Enabled: true,
			Init:    true,
		},
		User: UserNamespaceConfig{
			Enabled: true,
			UIDMap: []IDMap{
				{
					ContainerID: 0,
					HostID:      1000,
					Size:        1,
				},
			},
			GIDMap: []IDMap{
				{
					ContainerID: 0,
					HostID:      1000,
					Size:        1,
				},
			},
			NoNewPrivs: true,
		},
		UTS: UTSNamespaceConfig{
			Enabled:    true,
			Hostname:   "sssonector",
			Domainname: "local",
		},
		IPC: IPCNamespaceConfig{
			Enabled: true,
			Private: true,
		},
	}
}
