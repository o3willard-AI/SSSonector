package config

// Config holds the complete application configuration
type Config struct {
	Mode    string        `yaml:"mode"`
	Network NetworkConfig `yaml:"network"`
	Tunnel  TunnelConfig  `yaml:"tunnel"`
	Monitor MonitorConfig `yaml:"monitor"`
}

// NetworkConfig holds network interface configuration
type NetworkConfig struct {
	Interface string `yaml:"interface"`
	Address   string `yaml:"address"`
	MTU       int    `yaml:"mtu"`
}

// TunnelConfig holds tunnel configuration
type TunnelConfig struct {
	CertFile      string `yaml:"cert_file"`
	KeyFile       string `yaml:"key_file"`
	CAFile        string `yaml:"ca_file"`
	ListenAddress string `yaml:"listen_address"`
	ListenPort    int    `yaml:"listen_port"`
	ServerAddress string `yaml:"server_address"`
	ServerPort    int    `yaml:"server_port"`
	MaxClients    int    `yaml:"max_clients"`
	UploadKbps    int    `yaml:"upload_kbps"`
	DownloadKbps  int    `yaml:"download_kbps"`
}

// MonitorConfig holds monitoring configuration
type MonitorConfig struct {
	LogFile       string `yaml:"log_file"`
	SNMPEnabled   bool   `yaml:"snmp_enabled"`
	SNMPPort      int    `yaml:"snmp_port"`
	SNMPCommunity string `yaml:"snmp_community"`
}
