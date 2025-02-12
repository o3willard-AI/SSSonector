package types

import "time"

// NewDuration creates a new Duration from time.Duration
func NewDuration(d time.Duration) Duration {
	return Duration{Duration: d}
}

// NewLoggingConfig creates a new LoggingConfig
func NewLoggingConfig() *LoggingConfig {
	return &LoggingConfig{
		Level:  "info",
		Format: "text",
		Output: "stdout",
	}
}

// NewNetworkConfig creates a new NetworkConfig
func NewNetworkConfig() *NetworkConfig {
	return &NetworkConfig{
		MTU:    1500,
		DNS:    []string{},
		Routes: []string{},
	}
}

// NewTunnelConfig creates a new TunnelConfig
func NewTunnelConfig() *TunnelConfig {
	return &TunnelConfig{
		MTU:         1500,
		Protocol:    "tcp",
		Compression: false,
		Keepalive:   NewDuration(60 * time.Second),
	}
}

// NewSecurityConfig creates a new SecurityConfig
func NewSecurityConfig() *SecurityConfig {
	return &SecurityConfig{
		MemoryProtections: NewMemoryProtectionsConfig(),
		Namespace:         NewNamespaceConfig(),
		Capabilities:      NewCapabilitiesConfig(),
		Seccomp:           NewSeccompConfig(),
		TLS:               NewTLSConfig(),
	}
}

// NewMemoryProtectionsConfig creates a new MemoryProtectionsConfig
func NewMemoryProtectionsConfig() *MemoryProtectionsConfig {
	return &MemoryProtectionsConfig{
		NoExec:  true,
		ASLR:    true,
		Enabled: true,
	}
}

// NewNamespaceConfig creates a new NamespaceConfig
func NewNamespaceConfig() *NamespaceConfig {
	return &NamespaceConfig{
		Network: true,
		Mount:   true,
		Enabled: true,
	}
}

// NewCapabilitiesConfig creates a new CapabilitiesConfig
func NewCapabilitiesConfig() *CapabilitiesConfig {
	return &CapabilitiesConfig{
		Drop:    []string{},
		Add:     []string{},
		Enabled: true,
	}
}

// NewSeccompConfig creates a new SeccompConfig
func NewSeccompConfig() *SeccompConfig {
	return &SeccompConfig{
		Enabled: true,
		Profile: "default",
	}
}

// NewTLSConfig creates a new TLSConfig
func NewTLSConfig() *TLSConfig {
	return &TLSConfig{
		Options:      NewTLSConfigOptions(),
		CertRotation: NewCertRotation(),
		MinVersion:   "1.2",
		MaxVersion:   "1.3",
	}
}

// NewTLSConfigOptions creates a new TLSConfigOptions
func NewTLSConfigOptions() *TLSConfigOptions {
	return &TLSConfigOptions{
		MinVersion:   "1.2",
		MaxVersion:   "1.3",
		CipherSuites: []string{},
		Ciphers:      []string{},
	}
}

// NewCertRotation creates a new CertRotation
func NewCertRotation() *CertRotation {
	return &CertRotation{
		Enabled:  false,
		Interval: NewDuration(24 * time.Hour),
	}
}

// NewMonitorConfig creates a new MonitorConfig
func NewMonitorConfig() *MonitorConfig {
	return &MonitorConfig{
		Enabled:    false,
		Interval:   NewDuration(time.Minute),
		Prometheus: NewPrometheusConfig(),
		Type:       "basic",
	}
}

// NewPrometheusConfig creates a new PrometheusConfig
func NewPrometheusConfig() *PrometheusConfig {
	return &PrometheusConfig{
		Enabled:    false,
		Port:       9090,
		Path:       "/metrics",
		BufferSize: 1000,
	}
}

// NewMetricsConfig creates a new MetricsConfig
func NewMetricsConfig() *MetricsConfig {
	return &MetricsConfig{
		Enabled:    false,
		Interval:   NewDuration(10 * time.Second),
		Address:    "localhost:8080",
		BufferSize: 1000,
	}
}

// NewSNMPConfig creates a new SNMPConfig
func NewSNMPConfig() *SNMPConfig {
	return &SNMPConfig{
		Enabled:   false,
		Community: "public",
		Port:      161,
	}
}

// NewConfigMetadata creates a new ConfigMetadata
func NewConfigMetadata() *ConfigMetadata {
	now := time.Now().Format(time.RFC3339)
	return &ConfigMetadata{
		Version:      "1.0.0",
		LastModified: now,
		Created:      now,
		Modified:     now,
		CreatedAt:    now,
		UpdatedAt:    now,
	}
}

// String returns the string representation of Mode
func (m Mode) MarshalText() ([]byte, error) {
	return []byte(m.String()), nil
}

// UnmarshalText implements encoding.TextUnmarshaler
func (m *Mode) UnmarshalText(text []byte) error {
	mode, err := ParseMode(string(text))
	if err != nil {
		return err
	}
	*m = mode
	return nil
}

// String returns the string representation of Type
func (t Type) MarshalText() ([]byte, error) {
	return []byte(t.String()), nil
}

// UnmarshalText implements encoding.TextUnmarshaler
func (t *Type) UnmarshalText(text []byte) error {
	typ, err := ParseType(string(text))
	if err != nil {
		return err
	}
	*t = typ
	return nil
}
