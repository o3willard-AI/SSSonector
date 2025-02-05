package config

// Config represents the main application configuration
type Config struct {
	Mode    string        `yaml:"mode"`
	Network NetworkConfig `yaml:"network"`
	Tunnel  TunnelConfig  `yaml:"tunnel"`
	Monitor MonitorConfig `yaml:"monitor"`
}

// NetworkConfig contains network interface settings
type NetworkConfig struct {
	Interface string `yaml:"interface"`
	Address   string `yaml:"address"`
	MTU       int    `yaml:"mtu"`
}

// TunnelConfig contains SSL tunnel settings
type TunnelConfig struct {
	// Certificate paths
	CertFile string `yaml:"cert_file"`
	KeyFile  string `yaml:"key_file"`
	CAFile   string `yaml:"ca_file"`

	// Server settings
	ListenAddress string `yaml:"listen_address,omitempty"`
	ListenPort    int    `yaml:"listen_port,omitempty"`
	MaxClients    int    `yaml:"max_clients,omitempty"`

	// Client settings
	ServerAddress string `yaml:"server_address,omitempty"`
	ServerPort    int    `yaml:"server_port,omitempty"`

	// Rate limiting settings
	UploadKbps   int `yaml:"upload_kbps"`
	DownloadKbps int `yaml:"download_kbps"`
}

// MonitorConfig contains monitoring settings
type MonitorConfig struct {
	LogFile       string `yaml:"log_file"`
	SNMPEnabled   bool   `yaml:"snmp_enabled"`
	SNMPPort      int    `yaml:"snmp_port"`
	SNMPCommunity string `yaml:"snmp_community"`
}
