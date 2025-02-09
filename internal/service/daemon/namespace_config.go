package daemon

// NamespaceConfig represents Linux namespace configuration
type NamespaceConfig struct {
	// Namespace configurations
	Mount   MountNamespaceConfig
	PID     PIDNamespaceConfig
	IPC     IPCNamespaceConfig
	UTS     UTSNamespaceConfig
	User    UserNamespaceConfig
	Network NetworkNamespaceConfig
}

// MountNamespaceConfig represents mount namespace configuration
type MountNamespaceConfig struct {
	Enabled bool
	Mounts  []MountConfig
	Tmpfs   []TmpfsMount
	Bind    []BindMount
}

// TmpfsMount represents a tmpfs mount configuration
type TmpfsMount struct {
	Target string
	Size   string
	Mode   uint32
}

// BindMount represents a bind mount configuration
type BindMount struct {
	Source string
	Target string
	RO     bool
}

// MountConfig represents mount configuration
type MountConfig struct {
	Source     string
	Target     string
	Type       string // bind, tmpfs, proc, etc.
	Flags      uint
	Data       string
	Private    bool
	Slave      bool
	Shared     bool
	Unbindable bool
}

// PIDNamespaceConfig represents PID namespace configuration
type PIDNamespaceConfig struct {
	Enabled bool
	Init    bool // Run init process
}

// IPCNamespaceConfig represents IPC namespace configuration
type IPCNamespaceConfig struct {
	Enabled bool
}

// UTSNamespaceConfig represents UTS namespace configuration
type UTSNamespaceConfig struct {
	Enabled    bool
	Hostname   string
	Domainname string
}

// UserNamespaceConfig represents user namespace configuration
type UserNamespaceConfig struct {
	Enabled    bool
	UIDMap     []IDMap
	GIDMap     []IDMap
	NoNewPrivs bool // Disable gaining new privileges
}

// NetworkNamespaceConfig represents network namespace configuration
type NetworkNamespaceConfig struct {
	Enabled   bool
	Type      string // none, bridge, host
	Name      string // network interface name
	IPAddress string // IP address with CIDR (e.g., "10.0.0.2/24")
	Gateway   string // Gateway IP address
	DNS       []string
}

// IDMap represents a UID/GID mapping
type IDMap struct {
	ContainerID int
	HostID      int
	Size        int
}

// GetDefaultNamespaceConfig returns default namespace configuration
func GetDefaultNamespaceConfig() *NamespaceConfig {
	return &NamespaceConfig{
		Mount: MountNamespaceConfig{
			Enabled: true,
			Mounts: []MountConfig{
				{
					Source:  "/etc/resolv.conf",
					Target:  "/etc/resolv.conf",
					Type:    "bind",
					Flags:   MS_BIND | MS_RDONLY,
					Private: true,
				},
				{
					Source:  "/etc/hosts",
					Target:  "/etc/hosts",
					Type:    "bind",
					Flags:   MS_BIND | MS_RDONLY,
					Private: true,
				},
			},
			Tmpfs: []TmpfsMount{
				{
					Target: "/tmp",
					Size:   "1G",
					Mode:   0755,
				},
			},
			Bind: []BindMount{
				{
					Source: "/etc/ssl/certs",
					Target: "/etc/ssl/certs",
					RO:     true,
				},
			},
		},
		PID: PIDNamespaceConfig{
			Enabled: true,
			Init:    true,
		},
		IPC: IPCNamespaceConfig{
			Enabled: true,
		},
		UTS: UTSNamespaceConfig{
			Enabled:    true,
			Hostname:   "sssonector",
			Domainname: "local",
		},
		User: UserNamespaceConfig{
			Enabled:    false,
			NoNewPrivs: true,
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
		},
		Network: NetworkNamespaceConfig{
			Enabled:   true,
			Type:      "bridge",
			Name:      "sssonector0",
			IPAddress: "10.0.0.2/24",
			Gateway:   "10.0.0.1",
			DNS:       []string{"8.8.8.8", "8.8.4.4"},
		},
	}
}
