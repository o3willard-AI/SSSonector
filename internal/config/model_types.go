// Package types provides configuration types for the SSSonector service
package config

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
	Version       string    `yaml:"version" json:"version"`
	Created       time.Time `yaml:"created" json:"created"`
	Modified      time.Time `yaml:"modified" json:"modified"`
	CreatedBy     string    `yaml:"created_by" json:"created_by"`
	CreatedAt     time.Time `yaml:"created_at" json:"created_at"`
	UpdatedAt     time.Time `yaml:"updated_at" json:"updated_at"`
	Environment   string    `yaml:"environment" json:"environment"`
	Region        string    `yaml:"region" json:"region"`
	SchemaVersion string    `yaml:"schema_version" json:"schema_version"`
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
	Facade   FacadeConfig   `yaml:"facade" json:"facade"`
	NAT      NATConfig      `yaml:"nat" json:"nat"`
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
	Name       string     `yaml:"name" json:"name"`
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

	Reconnect ReconnectConfig `yaml:"reconnect" json:"reconnect"`

	// Dead-peer detection (seconds; 0 disables). Keepalive drives TCP
	// probes; IdleTimeout closes connections silent for the whole window.
	KeepAliveSeconds   int `yaml:"keepalive_seconds" json:"keepalive_seconds"`
	IdleTimeoutSeconds int `yaml:"idle_timeout_seconds" json:"idle_timeout_seconds"`
}

// ReconnectConfig tunes the client's automatic reconnection behavior.
type ReconnectConfig struct {
	MaxAttempts  int           `yaml:"max_attempts" json:"max_attempts"`
	InitialDelay time.Duration `yaml:"initial_delay" json:"initial_delay"`
	MaxDelay     time.Duration `yaml:"max_delay" json:"max_delay"`
	Jitter       float64       `yaml:"jitter" json:"jitter"`
}

// Normalized fills zero fields with production defaults.
func (r ReconnectConfig) Normalized() ReconnectConfig {
	if r.MaxAttempts <= 0 {
		r.MaxAttempts = 10
	}
	if r.InitialDelay <= 0 {
		r.InitialDelay = 1 * time.Second
	}
	if r.MaxDelay <= 0 {
		r.MaxDelay = 30 * time.Second
	}
	if r.Jitter <= 0 {
		r.Jitter = 0.2
	}
	return r
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

	// AllowPlaintext explicitly permits running without TLS when
	// certificates are missing or unusable. It defaults to false: the
	// service refuses to start rather than silently downgrading.
	AllowPlaintext bool `yaml:"allow_plaintext" json:"allow_plaintext"`
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

	// ListenAddress restricts the metrics/health bind interface.
	// Empty binds all interfaces; production should pin a loopback or
	// management-network address.
	ListenAddress string `yaml:"listen_address" json:"listen_address"`
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
	Address   string `yaml:"address" json:"address"`
	Port      int    `yaml:"port" json:"port"`
	Community string `yaml:"community" json:"community"`
}

// FacadeConfig represents HTTPS facade configuration for firewall traversal.
// The facade allows tunnel traffic to traverse firewalls that only permit
// standard HTTPS (port 443) by disguising tunnel connections as WebSocket
// upgrades over a legitimate HTTPS web server.
type FacadeConfig struct {
	// Enabled activates the HTTPS facade (server) or fallback (client)
	Enabled bool `yaml:"enabled" json:"enabled"`
	// ListenAddress is the address the facade server binds to (server only)
	ListenAddress string `yaml:"listen_address" json:"listen_address"`
	// ListenPort is the port the facade server listens on (server only, typically 443)
	ListenPort int `yaml:"listen_port" json:"listen_port"`
	// ServerAddress is the facade server address to connect to (client only, defaults to tunnel.server_address)
	ServerAddress string `yaml:"server_address" json:"server_address"`
	// ServerPort is the facade server port to connect to (client only, typically 443)
	ServerPort int `yaml:"server_port" json:"server_port"`
	// Hostname is the server hostname for TLS SNI (server only)
	Hostname string `yaml:"hostname" json:"hostname"`
	// WebRoot is the content returned for GET / (makes the server look like a real website)
	WebRoot string `yaml:"web_root" json:"web_root"`
	// TokenSecret is the shared secret for HMAC token authentication.
	// If empty, the secret is derived from the CA certificate.
	TokenSecret string `yaml:"token_secret" json:"token_secret"`
	// TokenTTL is the validity duration for authentication tokens (default 30s)
	TokenTTL time.Duration `yaml:"token_ttl" json:"token_ttl"`
	// DirectTimeout is how long the client waits for a direct connection before
	// falling back to the facade (client only, default 3s)
	DirectTimeout time.Duration `yaml:"direct_timeout" json:"direct_timeout"`
	// TLS holds TLS configuration specific to the facade. If cert/key/ca are empty,
	// they are inherited from the auth section.
	TLS FacadeTLSConfig `yaml:"tls" json:"tls"`
	// TunnelPorts lists the tunnel ports this facade routes to (server only)
	TunnelPorts []int `yaml:"tunnel_ports" json:"tunnel_ports"`
}

// FacadeTLSConfig represents TLS configuration specific to the HTTPS facade.
// If fields are empty, they inherit from the main auth configuration.
type FacadeTLSConfig struct {
	CertFile string `yaml:"cert_file" json:"cert_file"`
	KeyFile  string `yaml:"key_file" json:"key_file"`
	CAFile   string `yaml:"ca_file" json:"ca_file"`
}

// NATConfig represents the optional NAT/PAT subsystem configuration.
// Absent or disabled, the daemon performs no translation of any kind;
// the data path remains the raw L3 pipe it is today.
type NATConfig struct {
	// Enabled activates the NAT/PAT engine. Defaults to false: an empty
	// or missing nat section never enables translation (fail closed).
	Enabled bool `yaml:"enabled" json:"enabled"`
	// Forward configures stateful forward NAT (jump-host use case):
	// tunnel-side clients reaching networks on this host's other
	// interfaces via source-NAT.
	Forward NATForwardConfig `yaml:"forward" json:"forward"`
	// Reverse configures reverse PAT (service publishing): public
	// listeners on this host relayed through the tunnel to a service
	// behind the peer's TUN.
	Reverse NATReverseConfig `yaml:"reverse" json:"reverse"`
}

// NATForwardConfig configures stateful forward NAT (jump-host scenario).
type NATForwardConfig struct {
	// Enabled activates the forward path. Requires NATConfig.Enabled.
	Enabled bool `yaml:"enabled" json:"enabled"`
	// Rules list permitted flows. Empty when enabled means deny-all:
	// the ACL model is fail closed by construction.
	Rules []NATForwardRule `yaml:"rules" json:"rules"`
}

// NATForwardRule is one permitted forward flow class. First match wins;
// if no rule matches, the flow is denied.
type NATForwardRule struct {
	// Comment is an optional human-readable label (logging only).
	Comment string `yaml:"comment" json:"comment"`
	// SrcCIDR is the tunnel-side source network (e.g. the client TUN
	// subnet). Required, must parse as CIDR.
	SrcCIDR string `yaml:"src_cidr" json:"src_cidr"`
	// DstCIDR is the egress-side destination network reachable after
	// translation. Required, must parse as CIDR.
	DstCIDR string `yaml:"dst_cidr" json:"dst_cidr"`
	// Ports is the permitted service (destination port) list. Empty
	// means the rule matches no ports and therefore denies everything
	// it would otherwise cover — fail closed.
	Ports []int `yaml:"ports" json:"ports"`
}

// NATReverseConfig configures reverse PAT (publishing services through
// the tunnel to the internet-facing host).
type NATReverseConfig struct {
	// Enabled activates the reverse path. Requires NATConfig.Enabled.
	Enabled bool `yaml:"enabled" json:"enabled"`
	// Listeners are the public port mappings. Empty when enabled means
	// no public exposure at all (deny-all by absence).
	Listeners []NATListenerRule `yaml:"listeners" json:"listeners"`
}

// NATListenerRule maps one public listening port to a service behind the
// peer's TUN. Access is gated by AllowedCIDRs (default deny).
type NATListenerRule struct {
	// Comment is an optional human-readable label (logging only).
	Comment string `yaml:"comment" json:"comment"`
	// ListenPort is the public port this host listens on.
	ListenPort int `yaml:"listen_port" json:"listen_port"`
	// Dst is the tunnel-side service address in host:port form (e.g.
	// the peer TUN IP and the service port, "10.77.0.2:80").
	Dst string `yaml:"dst" json:"dst"`
	// AllowedCIDRs restricts which internet-side source networks may
	// use this listener. Empty denies every source (fail closed).
	AllowedCIDRs []string `yaml:"allowed_cidrs" json:"allowed_cidrs"`
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
		Version: "2.0.0",
		Metadata: ConfigMetadata{
			Version:       "2.0.0",
			Created:       time.Now(),
			Modified:      time.Now(),
			CreatedBy:     "system",
			CreatedAt:     time.Now(),
			UpdatedAt:     time.Now(),
			Environment:   "development",
			Region:        "local",
			SchemaVersion: CurrentSchemaVersion,
		},
		Throttle: ThrottleConfig{
			Enabled: false,
			Rate:    1024 * 1024, // 1MB/s default
			Burst:   1024 * 1024, // 1MB burst
		},
	}
}
