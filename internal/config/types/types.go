package types

import (
	"encoding/json"
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

// Duration is a wrapper around time.Duration for YAML/JSON marshaling
type Duration struct {
	time.Duration
}

// NewDuration creates a new Duration from time.Duration
func NewDuration(d time.Duration) Duration {
	return Duration{Duration: d}
}

// IsZero returns true if the duration is zero
func (d Duration) IsZero() bool {
	return d.Duration == 0
}

// UnmarshalYAML implements yaml.Unmarshaler
func (d *Duration) UnmarshalYAML(unmarshal func(interface{}) error) error {
	var v interface{}
	if err := unmarshal(&v); err != nil {
		return err
	}

	switch value := v.(type) {
	case float64:
		d.Duration = time.Duration(value)
		return nil
	case string:
		var err error
		d.Duration, err = time.ParseDuration(value)
		if err != nil {
			return err
		}
		return nil
	default:
		return fmt.Errorf("invalid duration")
	}
}

// MarshalYAML implements yaml.Marshaler
func (d Duration) MarshalYAML() (interface{}, error) {
	return d.String(), nil
}

// UnmarshalJSON implements json.Unmarshaler
func (d *Duration) UnmarshalJSON(b []byte) error {
	var v interface{}
	if err := json.Unmarshal(b, &v); err != nil {
		return err
	}

	switch value := v.(type) {
	case float64:
		d.Duration = time.Duration(value)
		return nil
	case string:
		var err error
		d.Duration, err = time.ParseDuration(value)
		if err != nil {
			return err
		}
		return nil
	default:
		return fmt.Errorf("invalid duration")
	}
}

// MarshalJSON implements json.Marshaler
func (d Duration) MarshalJSON() ([]byte, error) {
	return json.Marshal(d.String())
}

// AppConfig represents the main application configuration
type AppConfig struct {
	Type     string         `yaml:"type"`
	Config   ServiceConfig  `yaml:"config"`
	Version  string         `yaml:"version"`
	Metadata ConfigMetadata `yaml:"metadata"`
	Throttle ThrottleConfig `yaml:"throttle"`
}

// ServiceConfig represents service-specific configuration
type ServiceConfig struct {
	Mode     string         `yaml:"mode"`
	StateDir string         `yaml:"state_dir"`
	LogDir   string         `yaml:"log_dir"`
	Network  NetworkConfig  `yaml:"network"`
	Tunnel   TunnelConfig   `yaml:"tunnel"`
	Security SecurityConfig `yaml:"security"`
	Monitor  MonitorConfig  `yaml:"monitor"`
	Metrics  MetricsConfig  `yaml:"metrics"`
	Logging  LoggingConfig  `yaml:"logging"`
	SNMP     *SNMPConfig    `yaml:"snmp,omitempty"`
	Auth     *AuthConfig    `yaml:"auth,omitempty"`
}

// LoggingConfig represents logging configuration
type LoggingConfig struct {
	Level  string `yaml:"level"`
	Format string `yaml:"format"`
	Output string `yaml:"output"`
	File   string `yaml:"file,omitempty"`
}

// NetworkConfig represents network configuration
type NetworkConfig struct {
	Name      string   `yaml:"name"`
	Interface string   `yaml:"interface"`
	Address   string   `yaml:"address"`
	MTU       int      `yaml:"mtu"`
	DNS       []string `yaml:"dns,omitempty"`
	Routes    []string `yaml:"routes,omitempty"`
}

// TunnelConfig represents tunnel configuration
type TunnelConfig struct {
	ListenPort    int      `yaml:"listen_port"`
	ServerPort    int      `yaml:"server_port"`
	Protocol      string   `yaml:"protocol"`
	CertFile      string   `yaml:"cert_file"`
	KeyFile       string   `yaml:"key_file"`
	CAFile        string   `yaml:"ca_file"`
	ListenAddress string   `yaml:"listen_address"`
	ServerAddress string   `yaml:"server_address"`
	MaxClients    int      `yaml:"max_clients"`
	Port          int      `yaml:"port"`
	MTU           int      `yaml:"mtu"`
	Compression   bool     `yaml:"compression"`
	Keepalive     Duration `yaml:"keepalive"`
}

// SecurityConfig represents security configuration
type SecurityConfig struct {
	MemoryProtections MemoryProtectionConfig `yaml:"memory_protections"`
	Namespace         NamespaceConfig        `yaml:"namespace"`
	Capabilities      CapabilitiesConfig     `yaml:"capabilities"`
	Seccomp           SeccompConfig          `yaml:"seccomp,omitempty"`
	TLS               TLSConfig              `yaml:"tls,omitempty"`
	AuthMethod        string                 `yaml:"auth_method"`
	CertRotation      CertRotation           `yaml:"cert_rotation,omitempty"`
}

// MemoryProtectionConfig represents memory protection configuration
type MemoryProtectionConfig struct {
	NoExec  bool `yaml:"no_exec"`
	ASLR    bool `yaml:"aslr"`
	Enabled bool `yaml:"enabled"`
}

// NamespaceConfig represents namespace configuration
type NamespaceConfig struct {
	Network bool `yaml:"network"`
	Mount   bool `yaml:"mount"`
	Enabled bool `yaml:"enabled"`
}

// CapabilitiesConfig represents capabilities configuration
type CapabilitiesConfig struct {
	Drop    []string `yaml:"drop"`
	Add     []string `yaml:"add"`
	Enabled bool     `yaml:"enabled"`
}

// SeccompConfig represents seccomp settings
type SeccompConfig struct {
	Enabled bool   `yaml:"enabled"`
	Profile string `yaml:"profile"`
}

// TLSConfig represents TLS configuration
type TLSConfig struct {
	CertFile     string           `yaml:"cert_file"`
	KeyFile      string           `yaml:"key_file"`
	CAFile       string           `yaml:"ca_file"`
	Options      TLSConfigOptions `yaml:"options,omitempty"`
	CertRotation CertRotation     `yaml:"cert_rotation,omitempty"`
	MinVersion   string           `yaml:"min_version"`
	MaxVersion   string           `yaml:"max_version"`
}

// TLSConfigOptions represents additional TLS configuration options
type TLSConfigOptions struct {
	MinVersion   string   `yaml:"min_version"`
	MaxVersion   string   `yaml:"max_version"`
	CipherSuites []string `yaml:"cipher_suites"`
	Ciphers      []string `yaml:"ciphers"`
}

// CertRotation represents certificate rotation settings
type CertRotation struct {
	Enabled  bool     `yaml:"enabled"`
	Interval Duration `yaml:"interval"`
}

// MonitorConfig represents monitoring configuration
type MonitorConfig struct {
	Enabled    bool             `yaml:"enabled"`
	Type       string           `yaml:"type"`
	Interval   Duration         `yaml:"interval"`
	Prometheus PrometheusConfig `yaml:"prometheus,omitempty"`
}

// PrometheusConfig represents Prometheus configuration
type PrometheusConfig struct {
	Enabled    bool   `yaml:"enabled"`
	Port       int    `yaml:"port"`
	Path       string `yaml:"path"`
	BufferSize int    `yaml:"buffer_size"`
}

// MetricsConfig represents metrics configuration
type MetricsConfig struct {
	Enabled    bool     `yaml:"enabled"`
	Address    string   `yaml:"address"`
	Interval   Duration `yaml:"interval"`
	BufferSize int      `yaml:"buffer_size"`
}

// SNMPConfig represents SNMP configuration
type SNMPConfig struct {
	Enabled   bool   `yaml:"enabled"`
	Community string `yaml:"community"`
	Port      int    `yaml:"port"`
}

// AuthConfig represents authentication configuration
type AuthConfig struct {
	CertFile string `yaml:"cert_file"`
	KeyFile  string `yaml:"key_file"`
	CAFile   string `yaml:"ca_file"`
}

// ConfigMetadata represents application metadata
type ConfigMetadata struct {
	Version      string `yaml:"version"`
	Environment  string `yaml:"environment"`
	Region       string `yaml:"region"`
	LastModified string `yaml:"last_modified"`
	Created      string `yaml:"created"`
	Modified     string `yaml:"modified"`
	CreatedAt    string `yaml:"created_at"`
	UpdatedAt    string `yaml:"updated_at"`
}

// ThrottleConfig represents rate limiting configuration
type ThrottleConfig struct {
	Enabled bool  `yaml:"enabled"`
	Rate    int64 `yaml:"rate"`
	Burst   int64 `yaml:"burst"`
}

// DefaultConfig creates a new AppConfig with default values
func DefaultConfig() *AppConfig {
	return &AppConfig{
		Type:    TypeServer.String(),
		Version: "1.0.0",
		Config: ServiceConfig{
			Mode:     ModeServer.String(),
			StateDir: "/var/lib/sssonector",
			LogDir:   "/var/log/sssonector",
			Network: NetworkConfig{
				MTU: 1500,
			},
			Tunnel: TunnelConfig{
				Protocol:    "tcp",
				MaxClients:  1000,
				Port:        8080,
				MTU:         1500,
				Compression: false,
				Keepalive:   NewDuration(time.Minute),
			},
			Security: SecurityConfig{
				MemoryProtections: MemoryProtectionConfig{
					Enabled: true,
				},
				Namespace: NamespaceConfig{
					Enabled: true,
				},
				Capabilities: CapabilitiesConfig{
					Enabled: true,
				},
				TLS: TLSConfig{
					MinVersion: "1.2",
					MaxVersion: "1.3",
				},
			},
			Monitor: MonitorConfig{
				Enabled:  false,
				Type:     "basic",
				Interval: NewDuration(time.Minute),
			},
			Metrics: MetricsConfig{
				Enabled:    false,
				Address:    "localhost:8080",
				Interval:   NewDuration(10 * time.Second),
				BufferSize: 1000,
			},
			Logging: LoggingConfig{
				Level:  "info",
				Format: "text",
				Output: "stdout",
			},
		},
		Metadata: ConfigMetadata{
			Version:     "1.0.0",
			Environment: "development",
			Region:      "local",
		},
		Throttle: ThrottleConfig{
			Enabled: false,
			Rate:    0,
			Burst:   0,
		},
	}
}
