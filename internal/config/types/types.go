package types

import (
	"fmt"
	"time"
)

// Type represents the configuration type
type Type string

const (
	TypeServer Type = "server"
	TypeClient Type = "client"
)

// String returns the string representation of Type
func (t Type) String() string {
	return string(t)
}

// Mode represents the operation mode
type Mode string

const (
	ModeServer Mode = "server"
	ModeClient Mode = "client"
)

// String returns the string representation of Mode
func (m Mode) String() string {
	return string(m)
}

// ParseMode parses a string into Mode
func ParseMode(s string) (Mode, error) {
	switch s {
	case string(ModeServer):
		return ModeServer, nil
	case string(ModeClient):
		return ModeClient, nil
	default:
		return "", fmt.Errorf("invalid mode: %s", s)
	}
}

// ParseType parses a string into Type
func ParseType(s string) (Type, error) {
	switch s {
	case string(TypeServer):
		return TypeServer, nil
	case string(TypeClient):
		return TypeClient, nil
	default:
		return "", fmt.Errorf("invalid type: %s", s)
	}
}

// Duration is a wrapper around time.Duration for YAML marshaling
type Duration struct {
	time.Duration
}

// UnmarshalYAML implements yaml.Unmarshaler
func (d *Duration) UnmarshalYAML(unmarshal func(interface{}) error) error {
	var s string
	if err := unmarshal(&s); err != nil {
		return err
	}

	duration, err := time.ParseDuration(s)
	if err != nil {
		return err
	}

	d.Duration = duration
	return nil
}

// MarshalYAML implements yaml.Marshaler
func (d Duration) MarshalYAML() (interface{}, error) {
	return d.String(), nil
}

// Config represents the application configuration
type Config struct {
	// Mode specifies the operation mode ("server" or "client")
	Mode Mode `yaml:"mode"`

	// StateDir specifies the directory for state files
	StateDir string `yaml:"state_dir"`

	// LogDir specifies the directory for log files
	LogDir string `yaml:"log_dir"`

	// TLS configuration
	TLS *TLSConfig `yaml:"tls,omitempty"`

	// Tunnel configuration
	Tunnel *TunnelConfig `yaml:"tunnel,omitempty"`

	// Rate limiting configuration
	RateLimit *RateLimitConfig `yaml:"rate_limit,omitempty"`

	// Network configuration
	Network *NetworkConfig `yaml:"network,omitempty"`

	// Security configuration
	Security *SecurityConfig `yaml:"security,omitempty"`

	// Monitor configuration
	Monitor *MonitorConfig `yaml:"monitor,omitempty"`

	// Metrics configuration
	Metrics *MetricsConfig `yaml:"metrics,omitempty"`

	// Logging configuration
	Logging *LoggingConfig `yaml:"logging,omitempty"`

	// Authentication configuration
	Auth *AuthConfig `yaml:"auth,omitempty"`

	// SNMP configuration
	SNMP *SNMPConfig `yaml:"snmp,omitempty"`
}

// TLSConfig represents TLS configuration
type TLSConfig struct {
	// CertFile specifies the path to the certificate file
	CertFile string `yaml:"cert_file"`

	// KeyFile specifies the path to the private key file
	KeyFile string `yaml:"key_file"`

	// CAFile specifies the path to the CA certificate file
	CAFile string `yaml:"ca_file"`

	// Options specifies additional TLS options
	Options *TLSConfigOptions `yaml:"options,omitempty"`

	// CertRotation specifies certificate rotation settings
	CertRotation *CertRotation `yaml:"cert_rotation,omitempty"`

	// MinVersion specifies the minimum TLS version
	MinVersion string `yaml:"min_version"`

	// MaxVersion specifies the maximum TLS version
	MaxVersion string `yaml:"max_version"`
}

// TLSConfigOptions represents additional TLS configuration options
type TLSConfigOptions struct {
	// MinVersion specifies the minimum TLS version
	MinVersion string `yaml:"min_version"`

	// MaxVersion specifies the maximum TLS version
	MaxVersion string `yaml:"max_version"`

	// CipherSuites specifies allowed cipher suites
	CipherSuites []string `yaml:"cipher_suites"`

	// Ciphers specifies allowed ciphers (legacy)
	Ciphers []string `yaml:"ciphers"`
}

// CertRotation represents certificate rotation settings
type CertRotation struct {
	// Enabled specifies whether certificate rotation is enabled
	Enabled bool `yaml:"enabled"`

	// Interval specifies the rotation interval
	Interval Duration `yaml:"interval"`
}

// TunnelConfig represents tunnel configuration
type TunnelConfig struct {
	// ServerAddress specifies the server address
	ServerAddress string `yaml:"server_address"`

	// ServerPort specifies the server port
	ServerPort int `yaml:"server_port"`

	// Interface specifies the tunnel interface name
	Interface string `yaml:"interface"`

	// MTU specifies the tunnel MTU
	MTU int `yaml:"mtu"`

	// LocalAddress specifies the local tunnel address
	LocalAddress string `yaml:"local_address"`

	// RemoteAddress specifies the remote tunnel address
	RemoteAddress string `yaml:"remote_address"`

	// ListenAddress specifies the listen address
	ListenAddress string `yaml:"listen_address"`

	// ListenPort specifies the listen port
	ListenPort int `yaml:"listen_port"`

	// Protocol specifies the tunnel protocol
	Protocol string `yaml:"protocol"`

	// Compression specifies whether compression is enabled
	Compression bool `yaml:"compression"`

	// Keepalive specifies keepalive settings in seconds
	Keepalive Duration `yaml:"keepalive"`

	// Port specifies the tunnel port
	Port int `yaml:"port"`
}

// RateLimitConfig represents rate limiting configuration
type RateLimitConfig struct {
	// Enabled specifies whether rate limiting is enabled
	Enabled bool `yaml:"enabled"`

	// Rate specifies the rate limit in bytes per second
	Rate int64 `yaml:"rate"`

	// Burst specifies the burst size in bytes
	Burst int64 `yaml:"burst"`
}

// NetworkConfig represents network configuration
type NetworkConfig struct {
	// MTU specifies the network MTU
	MTU int `yaml:"mtu"`

	// DNS specifies DNS settings
	DNS []string `yaml:"dns"`

	// DNSServers specifies DNS servers (legacy)
	DNSServers []string `yaml:"dns_servers"`

	// Routes specifies network routes
	Routes []string `yaml:"routes"`

	// Interface specifies the network interface
	Interface string `yaml:"interface"`

	// Address specifies the network address
	Address string `yaml:"address"`
}

// SecurityConfig represents security configuration
type SecurityConfig struct {
	// MemoryProtections specifies memory protection settings
	MemoryProtections *MemoryProtectionsConfig `yaml:"memory_protections,omitempty"`

	// Namespace specifies namespace settings
	Namespace *NamespaceConfig `yaml:"namespace,omitempty"`

	// Capabilities specifies capability settings
	Capabilities *CapabilitiesConfig `yaml:"capabilities,omitempty"`

	// Seccomp specifies seccomp settings
	Seccomp *SeccompConfig `yaml:"seccomp,omitempty"`

	// TLS specifies TLS settings
	TLS *TLSConfig `yaml:"tls,omitempty"`

	// AuthMethod specifies the authentication method
	AuthMethod string `yaml:"auth_method"`

	// CertRotation specifies certificate rotation settings
	CertRotation *CertRotation `yaml:"cert_rotation,omitempty"`
}

// MemoryProtectionsConfig represents memory protection settings
type MemoryProtectionsConfig struct {
	// NoExec specifies whether to enable NX protection
	NoExec bool `yaml:"no_exec"`

	// ASLR specifies whether to enable ASLR
	ASLR bool `yaml:"aslr"`

	// Enabled specifies whether memory protections are enabled
	Enabled bool `yaml:"enabled"`
}

// NamespaceConfig represents namespace settings
type NamespaceConfig struct {
	// Network specifies whether to use network namespace
	Network bool `yaml:"network"`

	// Mount specifies whether to use mount namespace
	Mount bool `yaml:"mount"`

	// Enabled specifies whether namespaces are enabled
	Enabled bool `yaml:"enabled"`
}

// CapabilitiesConfig represents capability settings
type CapabilitiesConfig struct {
	// Drop specifies capabilities to drop
	Drop []string `yaml:"drop"`

	// Add specifies capabilities to add
	Add []string `yaml:"add"`

	// Enabled specifies whether capabilities are enabled
	Enabled bool `yaml:"enabled"`
}

// SeccompConfig represents seccomp settings
type SeccompConfig struct {
	// Enabled specifies whether seccomp is enabled
	Enabled bool `yaml:"enabled"`

	// Profile specifies the seccomp profile
	Profile string `yaml:"profile"`
}

// MonitorConfig represents monitoring configuration
type MonitorConfig struct {
	// Enabled specifies whether monitoring is enabled
	Enabled bool `yaml:"enabled"`

	// Interval specifies the monitoring interval
	Interval Duration `yaml:"interval"`

	// Prometheus specifies Prometheus settings
	Prometheus *PrometheusConfig `yaml:"prometheus,omitempty"`

	// Type specifies the monitor type
	Type string `yaml:"type"`
}

// PrometheusConfig represents Prometheus configuration
type PrometheusConfig struct {
	// Enabled specifies whether Prometheus is enabled
	Enabled bool `yaml:"enabled"`

	// Port specifies the Prometheus port
	Port int `yaml:"port"`

	// Path specifies the metrics path
	Path string `yaml:"path"`

	// BufferSize specifies the metrics buffer size
	BufferSize int `yaml:"buffer_size"`
}

// MetricsConfig represents metrics configuration
type MetricsConfig struct {
	// Enabled specifies whether metrics are enabled
	Enabled bool `yaml:"enabled"`

	// Interval specifies the metrics interval
	Interval Duration `yaml:"interval"`

	// Address specifies the metrics address
	Address string `yaml:"address"`

	// BufferSize specifies the metrics buffer size
	BufferSize int `yaml:"buffer_size"`
}

// LoggingConfig represents logging configuration
type LoggingConfig struct {
	// Level specifies the log level
	Level string `yaml:"level"`

	// Format specifies the log format
	Format string `yaml:"format"`

	// Output specifies the log output
	Output string `yaml:"output"`

	// File specifies the log file
	File string `yaml:"file"`
}

// AuthConfig represents authentication configuration
type AuthConfig struct {
	// CertFile specifies the certificate file
	CertFile string `yaml:"cert_file"`

	// KeyFile specifies the key file
	KeyFile string `yaml:"key_file"`

	// CAFile specifies the CA file
	CAFile string `yaml:"ca_file"`
}

// SNMPConfig represents SNMP configuration
type SNMPConfig struct {
	// Enabled specifies whether SNMP is enabled
	Enabled bool `yaml:"enabled"`

	// Community specifies the SNMP community
	Community string `yaml:"community"`

	// Port specifies the SNMP port
	Port int `yaml:"port"`
}

// ThrottleConfig represents throttling configuration
type ThrottleConfig struct {
	// Enabled specifies whether throttling is enabled
	Enabled bool `yaml:"enabled"`

	// Rate specifies the throttle rate
	Rate int64 `yaml:"rate"`

	// Burst specifies the burst size
	Burst int64 `yaml:"burst"`
}

// AdapterConfig represents adapter configuration
type AdapterConfig struct {
	// RetryAttempts specifies the number of retry attempts
	RetryAttempts int `yaml:"retry_attempts"`

	// RetryDelay specifies the delay between retries in milliseconds
	RetryDelay int64 `yaml:"retry_delay"`

	// CleanupTimeout specifies the cleanup timeout in milliseconds
	CleanupTimeout int64 `yaml:"cleanup_timeout"`

	// ValidateState specifies whether to validate state transitions
	ValidateState bool `yaml:"validate_state"`
}

// ConfigMetadata represents configuration metadata
type ConfigMetadata struct {
	// Version specifies the configuration version
	Version string `yaml:"version"`

	// LastModified specifies when the configuration was last modified
	LastModified string `yaml:"last_modified"`

	// Created specifies when the configuration was created
	Created string `yaml:"created"`

	// Modified specifies when the configuration was modified
	Modified string `yaml:"modified"`

	// CreatedBy specifies who created the configuration
	CreatedBy string `yaml:"created_by"`

	// CreatedAt specifies when the configuration was created
	CreatedAt string `yaml:"created_at"`

	// UpdatedAt specifies when the configuration was updated
	UpdatedAt string `yaml:"updated_at"`

	// Environment specifies the configuration environment
	Environment string `yaml:"environment"`

	// Region specifies the configuration region
	Region string `yaml:"region"`
}

// AppConfig represents the complete application configuration
type AppConfig struct {
	// Type specifies the configuration type
	Type Type `yaml:"type"`

	// Version specifies the configuration version
	Version string `yaml:"version"`

	// Config contains the parsed configuration
	Config *Config `yaml:"config"`

	// ConfigDir specifies the configuration directory
	ConfigDir string `yaml:"config_dir"`

	// Metadata contains configuration metadata
	Metadata *ConfigMetadata `yaml:"metadata,omitempty"`

	// Throttle contains throttling configuration
	Throttle *ThrottleConfig `yaml:"throttle,omitempty"`

	// Adapter contains adapter configuration
	Adapter *AdapterConfig `yaml:"adapter,omitempty"`
}

// DefaultConfig returns the default configuration
func DefaultConfig() *AppConfig {
	return &AppConfig{
		Type:    TypeServer,
		Version: "1.0.0",
		Config: &Config{
			Mode:     ModeServer,
			StateDir: "/var/lib/sssonector",
			LogDir:   "/var/log/sssonector",
		},
	}
}

// NewAppConfig creates a new application configuration
func NewAppConfig() *AppConfig {
	return DefaultConfig()
}

// NewConfig creates a new configuration
func NewConfig() *Config {
	return &Config{
		Mode:     ModeServer,
		StateDir: "/var/lib/sssonector",
		LogDir:   "/var/log/sssonector",
	}
}

// NewThrottleConfig creates a new throttle configuration
func NewThrottleConfig() *ThrottleConfig {
	return &ThrottleConfig{
		Enabled: false,
		Rate:    0,
		Burst:   0,
	}
}
