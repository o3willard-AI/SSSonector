package config

// Config represents the main configuration structure
type Config struct {
	Mode     string         `yaml:"mode"` // "server" or "client"
	Network  NetworkConfig  `yaml:"network"`
	TLS      TLSConfig      `yaml:"tls"`
	Logging  LoggingConfig  `yaml:"logging"`
	Monitor  MonitorConfig  `yaml:"monitor"`
	Service  ServiceConfig  `yaml:"service"`
	Throttle ThrottleConfig `yaml:"throttle"`
}

// NetworkConfig holds network-related settings
type NetworkConfig struct {
	Interface  string `yaml:"interface"`   // TUN interface name
	Address    string `yaml:"address"`     // IP address for the interface
	MTU        int    `yaml:"mtu"`         // MTU size (default: 1500)
	QoSEnabled bool   `yaml:"qos_enabled"` // Enable QoS tagging
	QoSTag     int    `yaml:"qos_tag"`     // QoS tag value when enabled

	// Server-specific settings
	ListenAddress string `yaml:"listen_address,omitempty"` // Server listen address
	ListenPort    int    `yaml:"listen_port,omitempty"`    // Server listen port
	MaxClients    int    `yaml:"max_clients,omitempty"`    // Maximum number of simultaneous clients

	// Client-specific settings
	ServerAddress string `yaml:"server_address,omitempty"` // Remote server address
	ServerPort    int    `yaml:"server_port,omitempty"`    // Remote server port
	RetryAttempts int    `yaml:"retry_attempts,omitempty"` // Number of reconnection attempts
	RetryInterval int    `yaml:"retry_interval,omitempty"` // Interval between retries in seconds
}

// TLSConfig holds TLS-related settings
type TLSConfig struct {
	CertFile     string `yaml:"cert_file"`     // Path to certificate file
	KeyFile      string `yaml:"key_file"`      // Path to private key file
	AutoGenerate bool   `yaml:"auto_generate"` // Auto-generate self-signed certificates
	ValidityDays int    `yaml:"validity_days"` // Validity period for auto-generated certs
	KeySize      int    `yaml:"key_size"`      // Key size for auto-generated certs
}

// LoggingConfig holds logging-related settings
type LoggingConfig struct {
	FilePath string `yaml:"file_path"` // Path to log file
	Level    string `yaml:"level"`     // Log level (info or debug)
	MaxSize  int    `yaml:"max_size"`  // Maximum log file size in MB
	Rotate   bool   `yaml:"rotate"`    // Enable log rotation
}

// MonitorConfig holds monitoring-related settings
type MonitorConfig struct {
	SNMPEnabled bool   `yaml:"snmp_enabled"` // Enable SNMP monitoring
	SNMPPort    int    `yaml:"snmp_port"`    // SNMP port number
	Community   string `yaml:"community"`    // SNMP community string
}

// ServiceConfig holds service-related settings
type ServiceConfig struct {
	AutoStart bool   `yaml:"auto_start"` // Start service on system boot
	User      string `yaml:"user"`       // User to run service as
	Group     string `yaml:"group"`      // Group to run service as
}

// ThrottleConfig holds bandwidth throttling settings
type ThrottleConfig struct {
	Enabled    bool  `yaml:"enabled"`     // Enable bandwidth throttling
	UploadKbps int64 `yaml:"upload_kbps"` // Upload limit in Kbps
	DownKbps   int64 `yaml:"down_kbps"`   // Download limit in Kbps
}

// DefaultConfig returns a Config with default values
func DefaultConfig() *Config {
	return &Config{
		Network: NetworkConfig{
			MTU:           1500,
			MaxClients:    5,
			RetryAttempts: 10,
			RetryInterval: 30,
		},
		TLS: TLSConfig{
			AutoGenerate: true,
			ValidityDays: 90,
			KeySize:      2048,
		},
		Logging: LoggingConfig{
			Level:   "info",
			MaxSize: 50,
			Rotate:  true,
		},
		Monitor: MonitorConfig{
			SNMPEnabled: true,
			SNMPPort:    161,
			Community:   "public",
		},
		Throttle: ThrottleConfig{
			Enabled:    false,
			UploadKbps: 1024, // 1 Mbps default if enabled
			DownKbps:   1024,
		},
	}
}
