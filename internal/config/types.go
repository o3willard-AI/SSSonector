package config

import (
	"path/filepath"
)

// Config represents the application configuration
type Config struct {
	Mode     string         `yaml:"mode"`
	Network  NetworkConfig  `yaml:"network"`
	Tunnel   TunnelConfig   `yaml:"tunnel"`
	Logging  LoggingConfig  `yaml:"logging"`
	Throttle ThrottleConfig `yaml:"throttle"`
	Monitor  MonitorConfig  `yaml:"monitor"`
}

// NetworkConfig represents network-related configuration
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
	UploadKbps    int64  `yaml:"upload_kbps,omitempty"`
	DownloadKbps  int64  `yaml:"download_kbps,omitempty"`
}

// LoggingConfig represents logging configuration
type LoggingConfig struct {
	Level string `yaml:"level"`
	File  string `yaml:"file"`
}

// ThrottleConfig represents rate limiting configuration
type ThrottleConfig struct {
	Enabled    bool  `yaml:"enabled"`
	RateLimit  int64 `yaml:"rate_limit"`
	BurstLimit int64 `yaml:"burst_limit"`
}

// MonitorConfig represents monitoring configuration
type MonitorConfig struct {
	Enabled        bool   `yaml:"enabled"`
	SNMPEnabled    bool   `yaml:"snmp_enabled"`
	SNMPAddress    string `yaml:"snmp_address"`
	SNMPPort       int    `yaml:"snmp_port"`
	SNMPCommunity  string `yaml:"snmp_community"`
	SNMPVersion    string `yaml:"snmp_version"`
	LogFile        string `yaml:"log_file"`
	UpdateInterval int    `yaml:"update_interval"`
}

// UpdateCertificatePaths updates all certificate-related paths in the configuration
func (c *Config) UpdateCertificatePaths(certDir string) {
	c.Tunnel.CertFile = filepath.Join(certDir, c.Mode+".crt")
	c.Tunnel.KeyFile = filepath.Join(certDir, c.Mode+".key")
	c.Tunnel.CAFile = filepath.Join(certDir, "ca.crt")
}

// DefaultConfig returns a configuration with default values
func DefaultConfig() *Config {
	return &Config{
		Network: NetworkConfig{
			Interface: "tun0",
			MTU:       1500,
		},
		Tunnel: TunnelConfig{
			CertFile:      "/etc/sssonector/certs/server.crt",
			KeyFile:       "/etc/sssonector/certs/server.key",
			CAFile:        "/etc/sssonector/certs/ca.crt",
			ListenAddress: "0.0.0.0",
			ListenPort:    8443,
			MaxClients:    10,
			UploadKbps:    1000, // 1 Mbps
			DownloadKbps:  1000, // 1 Mbps
		},
		Logging: LoggingConfig{
			Level: "info",
			File:  "/var/log/sssonector.log",
		},
		Throttle: ThrottleConfig{
			Enabled:    false,
			RateLimit:  1000000, // 1 MB/s
			BurstLimit: 2000000, // 2 MB/s
		},
		Monitor: MonitorConfig{
			Enabled:        false,
			SNMPEnabled:    false,
			SNMPAddress:    "127.0.0.1",
			SNMPPort:       161,
			SNMPCommunity:  "public",
			SNMPVersion:    "2c",
			LogFile:        "/var/log/sssonector_metrics.log",
			UpdateInterval: 30,
		},
	}
}
