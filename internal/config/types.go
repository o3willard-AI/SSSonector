package config

// Config represents the main application configuration
type Config struct {
	Mode     string         `yaml:"mode"`
	Network  NetworkConfig  `yaml:"network"`
	Tunnel   TunnelConfig   `yaml:"tunnel"`
	Monitor  MonitorConfig  `yaml:"monitor"`
	Logging  LoggingConfig  `yaml:"logging"`
	Throttle ThrottleConfig `yaml:"throttle"`
}

// NetworkConfig represents network interface configuration
type NetworkConfig struct {
	Interface string `yaml:"interface"`
	Address   string `yaml:"address"`
	MTU       int    `yaml:"mtu"`
}

// TunnelConfig represents SSL tunnel configuration
type TunnelConfig struct {
	CertFile      string `yaml:"cert_file"`
	KeyFile       string `yaml:"key_file"`
	CAFile        string `yaml:"ca_file"`
	ListenAddress string `yaml:"listen_address,omitempty"`
	ListenPort    int    `yaml:"listen_port,omitempty"`
	ServerAddress string `yaml:"server_address,omitempty"`
	ServerPort    int    `yaml:"server_port,omitempty"`
	MaxClients    int    `yaml:"max_clients,omitempty"`
	RetryAttempts int    `yaml:"retry_attempts"`
	RetryInterval int    `yaml:"retry_interval"`
}

// MonitorConfig represents monitoring configuration
type MonitorConfig struct {
	LogFile       string `yaml:"log_file"`
	LogLevel      string `yaml:"log_level"`
	SNMPEnabled   bool   `yaml:"snmp_enabled"`
	SNMPAddress   string `yaml:"snmp_address"`
	SNMPPort      int    `yaml:"snmp_port"`
	SNMPCommunity string `yaml:"snmp_community"`
}

// LoggingConfig represents logging configuration
type LoggingConfig struct {
	Level    string `yaml:"level"`
	FilePath string `yaml:"file_path"`
	MaxSize  int    `yaml:"max_size"`
}

// ThrottleConfig represents bandwidth throttling configuration
type ThrottleConfig struct {
	UploadKbps   int64 `yaml:"upload_kbps"`
	DownloadKbps int64 `yaml:"download_kbps"`
}
