package config

// ConfigStore defines the interface for configuration storage
type ConfigStore interface {
	Store(cfg *AppConfig) error
	Load(cfgType ConfigType, version string) (*AppConfig, error)
	Delete(cfgType ConfigType, version string) error
	List() ([]*AppConfig, error)
	ListByType(cfgType ConfigType) ([]*AppConfig, error)
	ListVersions(cfgType ConfigType) ([]string, error)
	GetLatest(cfgType ConfigType) (*AppConfig, error)
}

// ConfigValidator defines the interface for configuration validation
type ConfigValidator interface {
	Validate(cfg *AppConfig) error
	ValidateSchema(cfg *AppConfig, schema []byte) error
}

// ConfigWatcher defines the interface for configuration watching
type ConfigWatcher interface {
	Watch(cfgType ConfigType) (<-chan *AppConfig, error)
	StopWatch(cfgType ConfigType) error
}

// ConfigManager defines the interface for configuration management
type ConfigManager interface {
	GetStore() ConfigStore
	GetValidator() ConfigValidator
	GetWatcher() ConfigWatcher
	Apply(cfg *AppConfig) error
	Rollback(cfgType ConfigType, version string) error
	Diff(cfgType ConfigType, version1, version2 string) (string, error)
	Export(cfg *AppConfig, format ConfigFormat) ([]byte, error)
	Import(data []byte, format ConfigFormat) (*AppConfig, error)
	Subscribe(cfgType ConfigType) (<-chan *ConfigChangeEvent, error)
	Unsubscribe(cfgType ConfigType) error
}

// DefaultConfig returns a new configuration with default values
func DefaultConfig() *AppConfig {
	return &AppConfig{
		Mode: ModeServer,
		Network: NetworkConfig{
			Interface: "eth0",
			MTU:       1500,
		},
		Tunnel: TunnelConfig{
			Protocol:       "tcp",
			Encryption:     "aes-256-gcm",
			Compression:    "none",
			MaxConnections: 100,
			MaxClients:     50,
			BufferSize:     65536,
		},
		Monitor: MonitorConfig{
			Enabled:  true,
			Interval: 60,
			LogLevel: "info",
		},
		Security: SecurityConfig{
			TLS: TLSConfig{
				MinVersion: "1.2",
				MaxVersion: "1.3",
				Ciphers: []string{
					"TLS_ECDHE_ECDSA_WITH_AES_256_GCM_SHA384",
					"TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384",
				},
			},
			AuthMethod: "certificate",
		},
	}
}
