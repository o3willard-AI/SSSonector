package types

import (
	"encoding/json"
	"fmt"
	"time"

	"go.uber.org/zap/zapcore"
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

// UnmarshalYAML implements yaml.Unmarshaler
func (t *Type) UnmarshalYAML(unmarshal func(interface{}) error) error {
	var s string
	if err := unmarshal(&s); err != nil {
		return err
	}
	parsed, err := ParseType(s)
	if err != nil {
		return err
	}
	*t = parsed
	return nil
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

// UnmarshalYAML implements yaml.Unmarshaler
func (m *Mode) UnmarshalYAML(unmarshal func(interface{}) error) error {
	var s string
	if err := unmarshal(&s); err != nil {
		return err
	}
	parsed, err := ParseMode(s)
	if err != nil {
		return err
	}
	*m = parsed
	return nil
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

// CLIFlags represents command-line interface flags
type CLIFlags struct {
	ConfigFile string
	Debug      bool
	Version    bool
	Help       bool
}

// NewCLIFlags creates a new CLIFlags with default values
func NewCLIFlags() *CLIFlags {
	return &CLIFlags{
		ConfigFile: "/etc/sssonector/config.yaml",
		Debug:      false,
		Version:    false,
		Help:       false,
	}
}

// Common timeout durations
var (
	DefaultTimeout = NewDuration(30 * time.Second)
)

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
	Type      Type            `yaml:"type"`
	Config    *ServiceConfig  `yaml:"config"`
	Version   string          `yaml:"version"`
	Metadata  *ConfigMetadata `yaml:"metadata"`
	Throttle  *ThrottleConfig `yaml:"throttle"`
	ConfigDir string          `yaml:"-"`
	Adapter   *AdapterConfig  `yaml:"-"`
}

// AdapterConfig represents adapter-specific configuration
type AdapterConfig struct {
	RetryAttempts  int      `yaml:"retry_attempts"`
	RetryDelay     Duration `yaml:"retry_delay"`
	CleanupTimeout Duration `yaml:"cleanup_timeout"`
}

// ServiceConfig represents service-specific configuration
type ServiceConfig struct {
	Mode     Mode            `yaml:"mode"`
	StateDir string          `yaml:"state_dir"`
	LogDir   string          `yaml:"log_dir"`
	Network  *NetworkConfig  `yaml:"network"`
	Tunnel   *TunnelConfig   `yaml:"tunnel"`
	Security *SecurityConfig `yaml:"security"`
	Monitor  *MonitorConfig  `yaml:"monitor"`
	Metrics  *MetricsConfig  `yaml:"metrics"`
	Logging  *LoggingConfig  `yaml:"logging"`
	SNMP     *SNMPConfig     `yaml:"snmp,omitempty"`
	Auth     *AuthConfig     `yaml:"auth,omitempty"`
	TLS      *TLSConfig      `yaml:"tls,omitempty"`
}

// Constructor functions for each config type
func NewServiceConfig() *ServiceConfig {
	return &ServiceConfig{
		Network:  NewNetworkConfig(),
		Tunnel:   NewTunnelConfig(),
		Security: NewSecurityConfig(),
		Monitor:  NewMonitorConfig(),
		Metrics:  NewMetricsConfig(),
		Logging:  NewLoggingConfig(),
	}
}

func NewNetworkConfig() *NetworkConfig {
	return &NetworkConfig{
		MTU: 1500,
	}
}

func NewTunnelConfig() *TunnelConfig {
	return &TunnelConfig{
		Protocol:    "tcp",
		MaxClients:  1000,
		Port:        8080,
		MTU:         1500,
		Compression: false,
		Keepalive:   NewDuration(time.Minute),
	}
}

func NewSecurityConfig() *SecurityConfig {
	return &SecurityConfig{
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
	}
}

func NewMonitorConfig() *MonitorConfig {
	return &MonitorConfig{
		Type:     "basic",
		Interval: NewDuration(time.Minute),
	}
}

func NewMetricsConfig() *MetricsConfig {
	return &MetricsConfig{
		Address:    "localhost:8080",
		Interval:   NewDuration(10 * time.Second),
		BufferSize: 1000,
	}
}

func NewLoggingConfig() *LoggingConfig {
	return &LoggingConfig{
		Level:       "info",
		Format:      "text",
		Output:      "stdout",
		StartupLogs: true,
	}
}

func NewConfigMetadata() *ConfigMetadata {
	return &ConfigMetadata{
		Version:     "1.0.0",
		Environment: "development",
		Region:      "local",
	}
}

func NewThrottleConfig() *ThrottleConfig {
	return &ThrottleConfig{
		Enabled: false,
		Rate:    0,
		Burst:   0,
	}
}

func NewAdapterConfig() *AdapterConfig {
	return &AdapterConfig{
		RetryAttempts:  3,
		RetryDelay:     NewDuration(time.Second),
		CleanupTimeout: NewDuration(30 * time.Second),
	}
}

// DefaultConfig creates a new AppConfig with default values
func DefaultConfig() *AppConfig {
	return &AppConfig{
		Type:     TypeServer,
		Version:  "1.0.0",
		Config:   NewServiceConfig(),
		Metadata: NewConfigMetadata(),
		Throttle: NewThrottleConfig(),
		Adapter:  NewAdapterConfig(),
	}
}

// StartupPhase represents a startup phase
type StartupPhase string

const (
	StartupPhasePreStartup     StartupPhase = "PreStartup"
	StartupPhaseInitialization StartupPhase = "Initialization"
	StartupPhaseConnection     StartupPhase = "Connection"
	StartupPhaseListen         StartupPhase = "Listen"
)

// ValidatePhaseTransition validates if a phase transition is allowed
func ValidatePhaseTransition(current, next StartupPhase) error {
	// Define valid transitions
	validTransitions := map[StartupPhase][]StartupPhase{
		"":                         {StartupPhasePreStartup},
		StartupPhasePreStartup:     {StartupPhaseInitialization},
		StartupPhaseInitialization: {StartupPhaseConnection},
		StartupPhaseConnection:     {StartupPhaseListen},
		StartupPhaseListen:         {},
	}

	// Special case: empty current phase can only transition to PreStartup
	if current == "" {
		if next == StartupPhasePreStartup {
			return nil
		}
		return fmt.Errorf("must start with PreStartup phase")
	}

	// Validate current phase exists
	allowed, exists := validTransitions[current]
	if !exists {
		return fmt.Errorf("invalid current phase: %s", current)
	}

	// Empty next phase means we're done
	if next == "" {
		return nil
	}

	// Check if transition is allowed
	for _, validNext := range allowed {
		if next == validNext {
			return nil
		}
	}

	return fmt.Errorf("invalid phase transition: %s -> %s", current, next)
}

// StartupComponent represents a startup component
type StartupComponent string

const (
	StartupComponentConfig     StartupComponent = "Config"
	StartupComponentSecurity   StartupComponent = "Security"
	StartupComponentNetwork    StartupComponent = "Network"
	StartupComponentTLS        StartupComponent = "TLS"
	StartupComponentAdapter    StartupComponent = "Adapter"
	StartupComponentConnection StartupComponent = "Connection"
)

// StartupLog represents a detailed startup log entry
type StartupLog struct {
	Phase     StartupPhase           `json:"phase" yaml:"phase"`
	Component StartupComponent       `json:"component" yaml:"component"`
	Operation string                 `json:"operation" yaml:"operation"`
	Details   map[string]interface{} `json:"details,omitempty" yaml:"details,omitempty"`
	Duration  Duration               `json:"duration,omitempty" yaml:"duration,omitempty"`
	Status    string                 `json:"status" yaml:"status"`
	Error     string                 `json:"error,omitempty" yaml:"error,omitempty"`
	Timestamp time.Time              `json:"timestamp" yaml:"timestamp"`
}

// MarshalLogObject implements zapcore.ObjectMarshaler
func (s *StartupLog) MarshalLogObject(enc zapcore.ObjectEncoder) error {
	enc.AddString("phase", string(s.Phase))
	enc.AddString("component", string(s.Component))
	enc.AddString("operation", s.Operation)
	enc.AddString("status", s.Status)
	enc.AddTime("timestamp", s.Timestamp)

	if s.Error != "" {
		enc.AddString("error", s.Error)
	}

	if s.Duration.Duration > 0 {
		enc.AddDuration("duration", s.Duration.Duration)
	}

	if len(s.Details) > 0 {
		enc.AddObject("details", zapcore.ObjectMarshalerFunc(func(enc zapcore.ObjectEncoder) error {
			for k, v := range s.Details {
				switch val := v.(type) {
				case string:
					enc.AddString(k, val)
				case int:
					enc.AddInt(k, val)
				case bool:
					enc.AddBool(k, val)
				case float64:
					enc.AddFloat64(k, val)
				case time.Time:
					enc.AddTime(k, val)
				case time.Duration:
					enc.AddDuration(k, val)
				default:
					enc.AddReflected(k, val)
				}
			}
			return nil
		}))
	}

	return nil
}

// LoggingConfig represents logging configuration
type LoggingConfig struct {
	Level       string `yaml:"level"`
	Format      string `yaml:"format"`
	Output      string `yaml:"output"`
	File        string `yaml:"file,omitempty"`
	StartupLogs bool   `yaml:"startup_logs"`
}

type NetworkConfig struct {
	Name       string   `yaml:"name"`
	Interface  string   `yaml:"interface"`
	Address    string   `yaml:"address"`
	MTU        int      `yaml:"mtu"`
	DNS        []string `yaml:"dns,omitempty"`
	Routes     []string `yaml:"routes,omitempty"`
	DNSServers []string `yaml:"dns_servers,omitempty"` // Legacy field for backward compatibility
}

// UnmarshalYAML implements yaml.Unmarshaler
func (n *NetworkConfig) UnmarshalYAML(unmarshal func(interface{}) error) error {
	type alias NetworkConfig
	aux := &struct {
		*alias
	}{
		alias: (*alias)(n),
	}
	if err := unmarshal(aux); err != nil {
		return err
	}

	// Sync DNS and DNSServers fields
	if len(n.DNS) > 0 && len(n.DNSServers) == 0 {
		n.DNSServers = n.DNS
	} else if len(n.DNSServers) > 0 && len(n.DNS) == 0 {
		n.DNS = n.DNSServers
	}

	return nil
}

type TunnelConfig struct {
	ListenPort    int      `yaml:"listen_port"`
	ServerPort    int      `yaml:"server_port"`
	Protocol      string   `yaml:"protocol"`
	CertFile      string   `yaml:"cert_file"`
	KeyFile       string   `yaml:"key_file"`
	CAFile        string   `yaml:"ca_file"`
	ListenAddress string   `yaml:"listen_address"`
	ServerAddress string   `yaml:"server_address"`
	Server        string   `yaml:"server"` // Full server address:port for client mode
	MaxClients    int      `yaml:"max_clients"`
	Port          int      `yaml:"port"`
	MTU           int      `yaml:"mtu"`
	Compression   bool     `yaml:"compression"`
	Keepalive     Duration `yaml:"keepalive"`
}

type SecurityConfig struct {
	MemoryProtections MemoryProtectionConfig `yaml:"memory_protections"`
	Namespace         NamespaceConfig        `yaml:"namespace"`
	Capabilities      CapabilitiesConfig     `yaml:"capabilities"`
	Seccomp           SeccompConfig          `yaml:"seccomp,omitempty"`
	TLS               TLSConfig              `yaml:"tls,omitempty"`
	AuthMethod        string                 `yaml:"auth_method"`
	CertRotation      CertRotation           `yaml:"cert_rotation,omitempty"`
}

type MemoryProtectionConfig struct {
	NoExec  bool `yaml:"no_exec"`
	ASLR    bool `yaml:"aslr"`
	Enabled bool `yaml:"enabled"`
}

type NamespaceConfig struct {
	Network bool `yaml:"network"`
	Mount   bool `yaml:"mount"`
	Enabled bool `yaml:"enabled"`
}

type CapabilitiesConfig struct {
	Drop    []string `yaml:"drop"`
	Add     []string `yaml:"add"`
	Enabled bool     `yaml:"enabled"`
}

type SeccompConfig struct {
	Enabled bool   `yaml:"enabled"`
	Profile string `yaml:"profile"`
}

type TLSConfig struct {
	CertFile     string           `yaml:"cert_file"`
	KeyFile      string           `yaml:"key_file"`
	CAFile       string           `yaml:"ca_file"`
	Options      TLSConfigOptions `yaml:"options,omitempty"`
	CertRotation CertRotation     `yaml:"cert_rotation,omitempty"`
	MinVersion   string           `yaml:"min_version"`
	MaxVersion   string           `yaml:"max_version"`
}

type TLSConfigOptions struct {
	MinVersion   string   `yaml:"min_version"`
	MaxVersion   string   `yaml:"max_version"`
	CipherSuites []string `yaml:"cipher_suites"`
	Ciphers      []string `yaml:"ciphers"`
}

type CertRotation struct {
	Enabled  bool     `yaml:"enabled"`
	Interval Duration `yaml:"interval"`
}

type MonitorConfig struct {
	Enabled    bool             `yaml:"enabled"`
	Type       string           `yaml:"type"`
	Interval   Duration         `yaml:"interval"`
	Prometheus PrometheusConfig `yaml:"prometheus,omitempty"`
}

type PrometheusConfig struct {
	Enabled    bool   `yaml:"enabled"`
	Port       int    `yaml:"port"`
	Path       string `yaml:"path"`
	BufferSize int    `yaml:"buffer_size"`
}

type MetricsConfig struct {
	Enabled    bool     `yaml:"enabled"`
	Address    string   `yaml:"address"`
	Interval   Duration `yaml:"interval"`
	BufferSize int      `yaml:"buffer_size"`
}

type SNMPConfig struct {
	Enabled   bool   `yaml:"enabled"`
	Community string `yaml:"community"`
	Port      int    `yaml:"port"`
}

type AuthConfig struct {
	CertFile string `yaml:"cert_file"`
	KeyFile  string `yaml:"key_file"`
	CAFile   string `yaml:"ca_file"`
}

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

type ThrottleConfig struct {
	Enabled bool  `yaml:"enabled"`
	Rate    int64 `yaml:"rate"`
	Burst   int64 `yaml:"burst"`
}
