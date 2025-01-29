package config

// Config holds the complete application configuration
type Config struct {
	Mode     string         `mapstructure:"mode"`
	Network  NetworkConfig  `mapstructure:"network"`
	Tunnel   TunnelConfig   `mapstructure:"tunnel"`
	Monitor  MonitorConfig  `mapstructure:"monitor"`
	Logging  LoggingConfig  `mapstructure:"logging"`
	Throttle ThrottleConfig `mapstructure:"throttle"`
}

// NetworkConfig represents network interface configuration
type NetworkConfig struct {
	Interface string `mapstructure:"interface"`
	Address   string `mapstructure:"address"`
	MTU       int    `mapstructure:"mtu"`
}

// TunnelConfig holds SSL tunnel configuration
type TunnelConfig struct {
	CertFile       string  `mapstructure:"cert_file"`
	KeyFile        string  `mapstructure:"key_file"`
	CAFile         string  `mapstructure:"ca_file"`
	ListenAddress  string  `mapstructure:"listen_address"`
	ListenPort     int     `mapstructure:"listen_port"`
	ServerAddress  string  `mapstructure:"server_address"`
	ServerPort     int     `mapstructure:"server_port"`
	MaxClients     int     `mapstructure:"max_clients"`
	RetryAttempts  int     `mapstructure:"retry_attempts"`
	RetryInterval  int     `mapstructure:"retry_interval"`
	UploadKbps     float64 `mapstructure:"upload_kbps"`
	DownloadKbps   float64 `mapstructure:"download_kbps"`
	BandwidthLimit int64   `mapstructure:"bandwidth_limit"`
}

// MonitorConfig holds monitoring configuration
type MonitorConfig struct {
	LogFile       string `mapstructure:"log_file"`
	LogLevel      string `mapstructure:"log_level"`
	SNMPEnabled   bool   `mapstructure:"snmp_enabled"`
	SNMPAddress   string `mapstructure:"snmp_address"`
	SNMPPort      int    `mapstructure:"snmp_port"`
	SNMPCommunity string `mapstructure:"snmp_community"`
}

// LoggingConfig represents logging configuration
type LoggingConfig struct {
	Level    string `mapstructure:"level"`
	FilePath string `mapstructure:"file_path"`
	MaxSize  int    `mapstructure:"max_size"`
}

// ThrottleConfig represents bandwidth throttling configuration
type ThrottleConfig struct {
	UploadKbps   int64 `mapstructure:"upload_kbps"`
	DownloadKbps int64 `mapstructure:"download_kbps"`
}
