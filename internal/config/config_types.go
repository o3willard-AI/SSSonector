package config

import "time"

// ConfigType represents a configuration type
type ConfigType string

// ConfigFormat represents a configuration format
type ConfigFormat string

// Mode represents an operating mode
type Mode string

const (
	// Configuration types
	TypeServer     ConfigType = "server"
	TypeClient     ConfigType = "client"
	TypeTunnel     ConfigType = "tunnel"
	TypeSecurity   ConfigType = "security"
	TypeMonitoring ConfigType = "monitoring"

	// Configuration formats
	FormatJSON ConfigFormat = "json"
	FormatYAML ConfigFormat = "yaml"
	FormatTOML ConfigFormat = "toml"

	// Operating modes
	ModeServer Mode = "server"
	ModeClient Mode = "client"
)

// AppConfig represents the complete application configuration
type AppConfig struct {
	Metadata ConfigMetadata // Configuration metadata
	Mode     Mode           // Operating mode
	Network  NetworkConfig  // Network configuration
	Tunnel   TunnelConfig   // Tunnel configuration
	Monitor  MonitorConfig  // Monitor configuration
	Throttle ThrottleConfig // Rate limiting configuration
	Security SecurityConfig // Security configuration
}

// ConfigMetadata provides metadata for configuration tracking
type ConfigMetadata struct {
	Version       string            // Configuration version
	CreatedAt     time.Time         // Creation timestamp
	UpdatedAt     time.Time         // Last update timestamp
	LastValidator string            // Last validator identifier
	ValidatedAt   *time.Time        // Last validation timestamp
	Environment   string            // Deployment environment
	Region        string            // Geographic region
	Tags          map[string]string // Custom metadata tags
}

// NetworkConfig represents network configuration
type NetworkConfig struct {
	Interface  string   // Network interface name
	MTU        int      // Maximum Transmission Unit
	IPAddress  string   // IP address
	SubnetMask string   // Subnet mask
	Gateway    string   // Gateway address
	DNSServers []string // DNS server addresses
}

// TunnelConfig represents tunnel configuration
type TunnelConfig struct {
	Protocol       string // Protocol (tcp/udp)
	Encryption     string // Encryption algorithm
	Compression    string // Compression algorithm
	MaxConnections int    // Maximum concurrent connections
	MaxClients     int    // Maximum connected clients
	BufferSize     int    // Buffer size for data transfer
	CertFile       string // Certificate file path
	KeyFile        string // Private key file path
	CAFile         string // CA certificate file path
	ServerAddress  string // Server address (client mode)
	ServerPort     int    // Server port (client mode)
	ListenAddress  string // Listen address (server mode)
	ListenPort     int    // Listen port (server mode)
	KeepAlive      int    // Keep-alive interval in seconds
}

// MonitorConfig represents monitoring configuration
type MonitorConfig struct {
	Enabled       bool             // Enable monitoring
	Interval      time.Duration    // Monitoring interval
	LogLevel      string           // Log level
	SNMPEnabled   bool             // Enable SNMP monitoring
	SNMPPort      int              // SNMP port
	SNMPCommunity string           // SNMP community string
	Prometheus    PrometheusConfig // Prometheus configuration
}

// PrometheusConfig represents Prometheus monitoring configuration
type PrometheusConfig struct {
	Enabled bool   // Enable Prometheus metrics
	Port    int    // Prometheus metrics port
	Path    string // Metrics endpoint path
}

// ThrottleConfig represents rate limiting configuration
type ThrottleConfig struct {
	Enabled bool    // Enable rate limiting
	Rate    float64 // Rate limit in operations per second
	Burst   int     // Maximum burst size
}

// SecurityConfig represents security configuration
type SecurityConfig struct {
	TLS          TLSConfig          // TLS configuration
	AuthMethod   string             // Authentication method
	ACLs         []ACLConfig        // Access Control Lists
	CertRotation CertRotationConfig // Certificate rotation settings
}

// TLSConfig represents TLS configuration
type TLSConfig struct {
	MinVersion string   // Minimum TLS version
	MaxVersion string   // Maximum TLS version
	Ciphers    []string // Allowed cipher suites
}

// ACLConfig represents an Access Control List entry
type ACLConfig struct {
	Network string // Network CIDR
	Action  string // Allow/Deny action
}

// CertRotationConfig represents certificate rotation configuration
type CertRotationConfig struct {
	Enabled  bool          // Enable certificate rotation
	Interval time.Duration // Rotation interval
}

// ConfigChangeEvent represents a configuration change event
type ConfigChangeEvent struct {
	Type       ConfigType // Configuration type
	OldVersion string     // Previous version
	NewVersion string     // New version
	ChangeType string     // Type of change
	Timestamp  time.Time  // Event timestamp
}
