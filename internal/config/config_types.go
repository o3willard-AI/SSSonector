package config

import "time"

// Mode represents the operating mode
type Mode string

const (
	// ModeServer represents server mode
	ModeServer Mode = "server"
	// ModeClient represents client mode
	ModeClient Mode = "client"
)

// ConfigType represents the type of configuration
type ConfigType string

const (
	// TypeServer represents server configuration
	TypeServer ConfigType = "server"
	// TypeClient represents client configuration
	TypeClient ConfigType = "client"
	// TypeTunnel represents tunnel configuration
	TypeTunnel ConfigType = "tunnel"
	// TypeSecurity represents security configuration
	TypeSecurity ConfigType = "security"
	// TypeMonitoring represents monitoring configuration
	TypeMonitoring ConfigType = "monitoring"
)

// ConfigFormat represents the format of configuration data
type ConfigFormat string

const (
	// FormatJSON represents JSON format
	FormatJSON ConfigFormat = "json"
	// FormatYAML represents YAML format
	FormatYAML ConfigFormat = "yaml"
	// FormatTOML represents TOML format
	FormatTOML ConfigFormat = "toml"
)

// ConfigMetadata contains metadata about a configuration
type ConfigMetadata struct {
	// Version of configuration
	Version string `json:"version"`
	// Creation time
	CreatedAt time.Time `json:"created_at"`
	// Last update time
	UpdatedAt time.Time `json:"updated_at"`
	// Last validator
	LastValidator string `json:"last_validator,omitempty"`
	// Environment (e.g., production, staging)
	Environment string `json:"environment"`
	// Region (e.g., us-west-1)
	Region string `json:"region"`
	// Tags for categorization
	Tags map[string]string `json:"tags,omitempty"`
}

// NetworkConfig represents network configuration
type NetworkConfig struct {
	// Interface name
	Interface string `json:"interface"`
	// MTU size
	MTU int `json:"mtu"`
	// IP address
	IPAddress string `json:"ip_address"`
	// Subnet mask
	SubnetMask string `json:"subnet_mask"`
	// Gateway address
	Gateway string `json:"gateway"`
	// DNS servers
	DNSServers []string `json:"dns_servers"`
}

// TunnelConfig represents tunnel configuration
type TunnelConfig struct {
	// Server address
	ServerAddress string `json:"server_address"`
	// Server port
	ServerPort int `json:"server_port"`
	// Listen address
	ListenAddress string `json:"listen_address"`
	// Listen port
	ListenPort int `json:"listen_port"`
	// Protocol (tcp, udp)
	Protocol string `json:"protocol"`
	// Encryption algorithm
	Encryption string `json:"encryption"`
	// Compression algorithm
	Compression string `json:"compression"`
	// Keep alive interval
	KeepAlive time.Duration `json:"keep_alive"`
	// Maximum connections
	MaxConnections int `json:"max_connections"`
	// Maximum clients
	MaxClients int `json:"max_clients"`
	// Buffer size
	BufferSize int `json:"buffer_size"`
	// Upload bandwidth limit (Kbps)
	UploadKbps int `json:"upload_kbps"`
	// Download bandwidth limit (Kbps)
	DownloadKbps int `json:"download_kbps"`
	// Certificate paths
	CertFile string `json:"cert_file"`
	KeyFile  string `json:"key_file"`
	CAFile   string `json:"ca_file"`
}

// MonitorConfig represents monitoring configuration
type MonitorConfig struct {
	// Enable monitoring
	Enabled bool `json:"enabled"`
	// Metrics collection interval
	Interval time.Duration `json:"interval"`
	// Log file path
	LogFile string `json:"log_file"`
	// SNMP configuration
	SNMPEnabled   bool   `json:"snmp_enabled"`
	SNMPPort      int    `json:"snmp_port"`
	SNMPAddress   string `json:"snmp_address"`
	SNMPCommunity string `json:"snmp_community"`
	// Prometheus configuration
	Prometheus struct {
		Enabled bool   `json:"enabled"`
		Port    int    `json:"port"`
		Path    string `json:"path"`
	} `json:"prometheus"`
	// Log level
	LogLevel string `json:"log_level"`
}

// ThrottleConfig represents rate limiting configuration
type ThrottleConfig struct {
	// Enable rate limiting
	Enabled bool `json:"enabled"`
	// Rate limit (requests per second)
	Rate float64 `json:"rate"`
	// Burst size
	Burst int `json:"burst"`
	// Per-client limits
	PerClient bool `json:"per_client"`
}

// SecurityConfig represents security configuration
type SecurityConfig struct {
	// TLS configuration
	TLS struct {
		MinVersion string   `json:"min_version"`
		MaxVersion string   `json:"max_version"`
		Ciphers    []string `json:"ciphers"`
	} `json:"tls"`
	// Authentication method
	AuthMethod string `json:"auth_method"`
	// Access control lists
	ACLs []struct {
		Network string `json:"network"`
		Action  string `json:"action"`
	} `json:"acls"`
	// Certificate rotation
	CertRotation struct {
		Enabled  bool          `json:"enabled"`
		Interval time.Duration `json:"interval"`
	} `json:"cert_rotation"`
}

// AppConfig represents the complete application configuration
type AppConfig struct {
	// Configuration metadata
	Metadata ConfigMetadata `json:"metadata"`
	// Operating mode
	Mode Mode `json:"mode"`
	// Network configuration
	Network NetworkConfig `json:"network"`
	// Tunnel configuration
	Tunnel TunnelConfig `json:"tunnel"`
	// Monitor configuration
	Monitor MonitorConfig `json:"monitor"`
	// Throttle configuration
	Throttle ThrottleConfig `json:"throttle"`
	// Security configuration
	Security SecurityConfig `json:"security"`
}

// ConfigStore defines the interface for configuration storage
type ConfigStore interface {
	// Store stores a configuration
	Store(cfg *AppConfig) error
	// Load loads a configuration by type and version
	Load(cfgType ConfigType, version string) (*AppConfig, error)
	// Delete deletes a configuration by type and version
	Delete(cfgType ConfigType, version string) error
	// List lists all configurations
	List() ([]*AppConfig, error)
	// ListByType lists configurations by type
	ListByType(cfgType ConfigType) ([]*AppConfig, error)
	// ListVersions lists all versions of a configuration type
	ListVersions(cfgType ConfigType) ([]string, error)
	// GetLatest gets the latest version of a configuration type
	GetLatest(cfgType ConfigType) (*AppConfig, error)
}

// ConfigValidator defines the interface for configuration validation
type ConfigValidator interface {
	// Validate validates a configuration
	Validate(cfg *AppConfig) error
	// ValidateSchema validates a configuration against its schema
	ValidateSchema(cfg *AppConfig, schema []byte) error
}

// ConfigWatcher defines the interface for configuration change notifications
type ConfigWatcher interface {
	// Watch starts watching for configuration changes
	Watch(cfgType ConfigType) (<-chan *AppConfig, error)
	// StopWatch stops watching for configuration changes
	StopWatch(cfgType ConfigType) error
}

// ConfigManager defines the interface for configuration management
type ConfigManager interface {
	// GetStore returns the configuration store
	GetStore() ConfigStore
	// GetValidator returns the configuration validator
	GetValidator() ConfigValidator
	// GetWatcher returns the configuration watcher
	GetWatcher() ConfigWatcher
	// Apply applies a configuration
	Apply(cfg *AppConfig) error
	// Rollback rolls back to a previous configuration version
	Rollback(cfgType ConfigType, version string) error
	// Diff returns the differences between two configuration versions
	Diff(cfgType ConfigType, version1, version2 string) (string, error)
	// Export exports a configuration to a specific format
	Export(cfg *AppConfig, format ConfigFormat) ([]byte, error)
	// Import imports a configuration from a specific format
	Import(data []byte, format ConfigFormat) (*AppConfig, error)
}

// DefaultConfig returns the default configuration
func DefaultConfig() *AppConfig {
	now := time.Now()
	return &AppConfig{
		Metadata: ConfigMetadata{
			Version:     "1.0.0",
			CreatedAt:   now,
			UpdatedAt:   now,
			Environment: "development",
			Region:      "local",
			Tags:        make(map[string]string),
		},
		Mode: ModeClient,
		Network: NetworkConfig{
			MTU: 1500,
		},
		Tunnel: TunnelConfig{
			Protocol:       "tcp",
			Encryption:     "aes-256-gcm",
			Compression:    "none",
			KeepAlive:      30 * time.Second,
			MaxConnections: 100,
			MaxClients:     50,
			BufferSize:     65536,
			UploadKbps:     10000, // 10 Mbps
			DownloadKbps:   10000, // 10 Mbps
			ListenPort:     8080,
		},
		Monitor: MonitorConfig{
			Enabled:       true,
			Interval:      60 * time.Second,
			SNMPEnabled:   false,
			SNMPPort:      161,
			SNMPCommunity: "public",
			LogLevel:      "info",
			Prometheus: struct {
				Enabled bool   `json:"enabled"`
				Port    int    `json:"port"`
				Path    string `json:"path"`
			}{
				Enabled: true,
				Port:    9090,
				Path:    "/metrics",
			},
		},
		Throttle: ThrottleConfig{
			Enabled:   true,
			Rate:      1000,
			Burst:     2000,
			PerClient: true,
		},
		Security: SecurityConfig{
			TLS: struct {
				MinVersion string   `json:"min_version"`
				MaxVersion string   `json:"max_version"`
				Ciphers    []string `json:"ciphers"`
			}{
				MinVersion: "1.2",
				MaxVersion: "1.3",
				Ciphers: []string{
					"TLS_ECDHE_ECDSA_WITH_AES_256_GCM_SHA384",
					"TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384",
				},
			},
			AuthMethod: "certificate",
			CertRotation: struct {
				Enabled  bool          `json:"enabled"`
				Interval time.Duration `json:"interval"`
			}{
				Enabled:  true,
				Interval: 24 * time.Hour * 30, // 30 days
			},
		},
	}
}
