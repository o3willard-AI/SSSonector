package config

// Config represents the main configuration
type Config struct {
Mode    string        `yaml:"mode"`
Network NetworkConfig `yaml:"network"`
Tunnel  TunnelConfig `yaml:"tunnel"`
Monitor MonitorConfig `yaml:"monitor"`
Logging LoggingConfig `yaml:"logging"`
}

// NetworkConfig represents network interface configuration
type NetworkConfig struct {
Interface string `yaml:"interface"`
Address   string `yaml:"address"`
MTU       int    `yaml:"mtu"`
}

// TunnelConfig represents SSL tunnel configuration
type TunnelConfig struct {
CertFile       string `yaml:"certFile"`
KeyFile        string `yaml:"keyFile"`
CAFile         string `yaml:"caFile"`
ListenAddress  string `yaml:"listenAddress"`
ListenPort     int    `yaml:"listenPort"`
ServerAddress  string `yaml:"serverAddress,omitempty"`
ServerPort     int    `yaml:"serverPort,omitempty"`
MaxClients     int    `yaml:"maxClients,omitempty"`
RetryAttempts  int    `yaml:"retryAttempts"`
RetryInterval  int    `yaml:"retryInterval"`
UploadKbps     int    `yaml:"uploadKbps"`
DownloadKbps   int    `yaml:"downloadKbps"`
}

// MonitorConfig represents monitoring configuration
type MonitorConfig struct {
LogFile        string `yaml:"logFile"`
SNMPEnabled    bool   `yaml:"snmpEnabled"`
SNMPPort       int    `yaml:"snmpPort"`
SNMPCommunity  string `yaml:"snmpCommunity"`
}

// LoggingConfig represents logging configuration
type LoggingConfig struct {
Level    string `yaml:"level"`
FilePath string `yaml:"filePath"`
MaxSize  int    `yaml:"maxSize"`
}
