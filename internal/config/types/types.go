// Package types provides configuration types for the SSSonector service
package types

import (
	"encoding/json"
	"time"
)

// Type represents the configuration type
type Type string

const (
	// TypeServer represents server configuration type
	TypeServer Type = "server"
	// TypeClient represents client configuration type
	TypeClient Type = "client"
	// ModeServer represents server mode
	ModeServer = "server"
	// ModeClient represents client mode
	ModeClient = "client"
)

// String returns the string representation of Type
func (t Type) String() string {
	return string(t)
}

// MarshalYAML implements yaml.Marshaler
func (t Type) MarshalYAML() (interface{}, error) {
	return t.String(), nil
}

// UnmarshalYAML implements yaml.Unmarshaler
func (t *Type) UnmarshalYAML(unmarshal func(interface{}) error) error {
	var str string
	if err := unmarshal(&str); err != nil {
		return err
	}
	*t = Type(str)
	return nil
}

// MarshalJSON implements json.Marshaler
func (t Type) MarshalJSON() ([]byte, error) {
	return json.Marshal(t.String())
}

// UnmarshalJSON implements json.Unmarshaler
func (t *Type) UnmarshalJSON(data []byte) error {
	var str string
	if err := json.Unmarshal(data, &str); err != nil {
		return err
	}
	*t = Type(str)
	return nil
}

// AppConfig represents the application configuration
type AppConfig struct {
	Type     Type           `yaml:"type" json:"type"`
	Config   *Config        `yaml:"config" json:"config"`
	Version  string         `yaml:"version" json:"version"`
	Metadata ConfigMetadata `yaml:"metadata" json:"metadata"`
	Throttle ThrottleConfig `yaml:"throttle" json:"throttle"`
}

// ConfigMetadata represents configuration metadata
type ConfigMetadata struct {
	Version     string    `yaml:"version" json:"version"`
	Created     time.Time `yaml:"created" json:"created"`
	Modified    time.Time `yaml:"modified" json:"modified"`
	CreatedBy   string    `yaml:"created_by" json:"created_by"`
	CreatedAt   time.Time `yaml:"created_at" json:"created_at"`
	UpdatedAt   time.Time `yaml:"updated_at" json:"updated_at"`
	Environment string    `yaml:"environment" json:"environment"`
	Region      string    `yaml:"region" json:"region"`
}

// Config represents the main configuration structure
type Config struct {
	Mode     string         `yaml:"mode" json:"mode"`
	Logging  LoggingConfig  `yaml:"logging" json:"logging"`
	Auth     AuthConfig     `yaml:"auth" json:"auth"`
	Network  NetworkConfig  `yaml:"network" json:"network"`
	Tunnel   TunnelConfig   `yaml:"tunnel" json:"tunnel"`
	Security SecurityConfig `yaml:"security" json:"security"`
	Monitor  MonitorConfig  `yaml:"monitor" json:"monitor"`
	Metrics  MetricsConfig  `yaml:"metrics" json:"metrics"`
	SNMP     SNMPConfig     `yaml:"snmp" json:"snmp"`
}

// LoggingConfig represents logging configuration
type LoggingConfig struct {
	Level  string `yaml:"level" json:"level"`
	File   string `yaml:"file" json:"file"`
	Format string `yaml:"format" json:"format"`
}

// AuthConfig represents authentication configuration
type AuthConfig struct {
	Type          string       `yaml:"type" json:"type"`
	Certificate   string       `yaml:"certificate" json:"certificate"`
	Key           string       `yaml:"key" json:"key"`
	CACertificate string       `yaml:"ca_certificate" json:"ca_certificate"`
	CertFile      string       `yaml:"cert_file" json:"cert_file"`
	KeyFile       string       `yaml:"key_file" json:"key_file"`
	CAFile        string       `yaml:"ca_file" json:"ca_file"`
	AuthMethod    string       `yaml:"auth_method" json:"auth_method"`
	CertRotation  CertRotation `yaml:"cert_rotation" json:"cert_rotation"`
}

// NetworkConfig represents network configuration
type NetworkConfig struct {
	Interface  string     `yaml:"interface" json:"interface"`
	MTU        int        `yaml:"mtu" json:"mtu"`
	Address    string     `yaml:"address" json:"address"`
	DNSServers []string   `yaml:"dns_servers" json:"dns_servers"`
	IPv6       IPv6Config `yaml:"ipv6" json:"ipv6"`
}

// IPv6Config represents IPv6 experimental configuration
type IPv6Config struct {
	// Enabled indicates whether IPv6 support is enabled (experimental)
	Enabled bool `yaml:"enabled" json:"enabled"`
	// Address is the IPv6 address for the interface
	Address string `yaml:"address" json:"address"`
	// Prefix is the IPv6 network prefix length
	Prefix int `yaml:"prefix" json:"prefix"`
}

// TunnelConfig represents tunnel configuration
type TunnelConfig struct {
	ListenAddress string `yaml:"listen_address" json:"listen_address"`
	ListenPort    int    `yaml:"listen_port" json:"listen_port"`
	ServerAddress string `yaml:"server_address" json:"server_address"`
	ServerPort    int    `yaml:"server_port" json:"server_port"`
	Port          int    `yaml:"port" json:"port"`
	Protocol      string `yaml:"protocol" json:"protocol"`
	Compression   bool   `yaml:"compression" json:"compression"`
	Keepalive     string `yaml:"keepalive" json:"keepalive"`
}

// SecurityConfig represents security configuration
type SecurityConfig struct {
	MemoryProtections MemoryProtectionsConfig `yaml:"memory_protections" json:"memory_protections"`
	Namespace         NamespaceConfig         `yaml:"namespace" json:"namespace"`
	Capabilities      CapabilitiesConfig      `yaml:"capabilities" json:"capabilities"`
	Seccomp           SeccompConfig           `yaml:"seccomp" json:"seccomp"`
	TLS               TLSConfigOptions        `yaml:"tls" json:"tls"`
	AuthMethod        string                  `yaml:"auth_method" json:"auth_method"`
	CertRotation      CertRotation            `yaml:"cert_rotation" json:"cert_rotation"`
}

// MemoryProtectionsConfig represents memory protection settings
type MemoryProtectionsConfig struct {
	Enabled bool `yaml:"enabled" json:"enabled"`
}

// NamespaceConfig represents namespace settings
type NamespaceConfig struct {
	Enabled bool `yaml:"enabled" json:"enabled"`
}

// CapabilitiesConfig represents capabilities settings
type CapabilitiesConfig struct {
	Enabled bool `yaml:"enabled" json:"enabled"`
}

// SeccompConfig represents seccomp settings
type SeccompConfig struct {
	Enabled bool `yaml:"enabled" json:"enabled"`
}

// TLSConfigOptions represents TLS security settings
type TLSConfigOptions struct {
	MinVersion string   `yaml:"min_version" json:"min_version"`
	MaxVersion string   `yaml:"max_version" json:"max_version"`
	Ciphers    []string `yaml:"ciphers" json:"ciphers"`
}

// CertRotation represents certificate rotation settings
type CertRotation struct {
	Enabled  bool          `yaml:"enabled" json:"enabled"`
	Interval time.Duration `yaml:"interval" json:"interval"`
}

// MonitorConfig represents monitoring configuration
type MonitorConfig struct {
	Enabled    bool             `yaml:"enabled" json:"enabled"`
	Type       string           `yaml:"type" json:"type"`
	Interval   time.Duration    `yaml:"interval" json:"interval"`
	Prometheus PrometheusConfig `yaml:"prometheus" json:"prometheus"`
}

// PrometheusConfig represents Prometheus monitoring settings
type PrometheusConfig struct {
	Enabled    bool   `yaml:"enabled" json:"enabled"`
	Port       int    `yaml:"port" json:"port"`
	Path       string `yaml:"path" json:"path"`
	BufferSize int    `yaml:"buffer_size" json:"buffer_size"`
}

// MetricsConfig represents metrics configuration
type MetricsConfig struct {
	Enabled    bool          `yaml:"enabled" json:"enabled"`
	Address    string        `yaml:"address" json:"address"`
	Interval   time.Duration `yaml:"interval" json:"interval"`
	BufferSize int           `yaml:"buffer_size" json:"buffer_size"`
}

// SNMPConfig represents SNMP monitoring configuration
type SNMPConfig struct {
	Enabled   bool   `yaml:"enabled" json:"enabled"`
	Port      int    `yaml:"port" json:"port"`
	Community string `yaml:"community" json:"community"`
}

// ThrottleConfig represents rate limiting configuration
type ThrottleConfig struct {
	Enabled bool    `yaml:"enabled" json:"enabled"`
	Rate    float64 `yaml:"rate" json:"rate"`
	Burst   int     `yaml:"burst" json:"burst"`
}

// DefaultConfig returns a default configuration
func DefaultConfig() *AppConfig {
	return NewAppConfig(TypeServer)
}

// NewAppConfig creates a new AppConfig instance
func NewAppConfig(configType Type) *AppConfig {
	return &AppConfig{
		Type:    configType,
		Config:  &Config{Mode: string(configType)},
		Version: "1.0.0",
		Metadata: ConfigMetadata{
			Version:     "1.0.0",
			Created:     time.Now(),
			Modified:    time.Now(),
			CreatedBy:   "system",
			CreatedAt:   time.Now(),
			UpdatedAt:   time.Now(),
			Environment: "development",
			Region:      "local",
		},
		Throttle: ThrottleConfig{
			Enabled: false,
			Rate:    1024 * 1024, // 1MB/s default
			Burst:   1024 * 1024, // 1MB burst
		},
	}
}
