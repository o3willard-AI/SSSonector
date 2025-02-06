package config

import (
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v2"
)

// expandEnvVars replaces ${var} or $var in the string according to the values
// of the current environment variables.
func expandEnvVars(s string) string {
	return os.ExpandEnv(s)
}

// LoadConfig loads and validates the configuration from a YAML file
func LoadConfig(path string) (*Config, error) {
	// Read configuration file
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %v", err)
	}

	// Parse YAML
	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("failed to parse config file: %v", err)
	}

	// Expand environment variables in certificate paths
	cfg.Tunnel.CertFile = expandEnvVars(cfg.Tunnel.CertFile)
	cfg.Tunnel.KeyFile = expandEnvVars(cfg.Tunnel.KeyFile)
	cfg.Tunnel.CAFile = expandEnvVars(cfg.Tunnel.CAFile)

	// Validate configuration
	if err := validateConfig(&cfg); err != nil {
		return nil, fmt.Errorf("invalid configuration: %v", err)
	}

	// Set default values if needed
	setDefaults(&cfg)

	return &cfg, nil
}

// validateConfig performs validation of the configuration
func validateConfig(cfg *Config) error {
	// Validate mode
	if cfg.Mode != "server" && cfg.Mode != "client" {
		return fmt.Errorf("invalid mode: %s (must be 'server' or 'client')", cfg.Mode)
	}

	// Validate network settings
	if cfg.Network.Interface == "" {
		return fmt.Errorf("network interface not specified")
	}
	if cfg.Network.Address == "" {
		return fmt.Errorf("network address not specified")
	}
	if cfg.Network.MTU < 1280 || cfg.Network.MTU > 9000 {
		return fmt.Errorf("invalid MTU value: %d (must be between 1280 and 9000)", cfg.Network.MTU)
	}

	// Validate tunnel settings
	if err := validateTunnelConfig(&cfg.Tunnel, cfg.Mode); err != nil {
		return fmt.Errorf("invalid tunnel configuration: %v", err)
	}

	// Validate monitor settings
	if err := validateMonitorConfig(&cfg.Monitor); err != nil {
		return fmt.Errorf("invalid monitor configuration: %v", err)
	}

	return nil
}

// validateTunnelConfig validates tunnel-specific configuration
func validateTunnelConfig(cfg *TunnelConfig, mode string) error {
	// Validate certificate paths
	if cfg.CertFile == "" {
		return fmt.Errorf("certificate file not specified")
	}
	if cfg.KeyFile == "" {
		return fmt.Errorf("private key file not specified")
	}
	if cfg.CAFile == "" {
		return fmt.Errorf("CA certificate file not specified")
	}

	// Validate server-specific settings
	if mode == "server" {
		if cfg.ListenAddress == "" {
			return fmt.Errorf("server listen address not specified")
		}
		if cfg.ListenPort <= 0 || cfg.ListenPort > 65535 {
			return fmt.Errorf("invalid server listen port: %d", cfg.ListenPort)
		}
		if cfg.MaxClients <= 0 {
			return fmt.Errorf("invalid max clients value: %d", cfg.MaxClients)
		}
	}

	// Validate client-specific settings
	if mode == "client" {
		if cfg.ServerAddress == "" {
			return fmt.Errorf("server address not specified")
		}
		if cfg.ServerPort <= 0 || cfg.ServerPort > 65535 {
			return fmt.Errorf("invalid server port: %d", cfg.ServerPort)
		}
	}

	// Validate rate limiting settings
	if cfg.UploadKbps < 0 {
		return fmt.Errorf("invalid upload rate limit: %d", cfg.UploadKbps)
	}
	if cfg.DownloadKbps < 0 {
		return fmt.Errorf("invalid download rate limit: %d", cfg.DownloadKbps)
	}

	return nil
}

// validateMonitorConfig validates monitoring configuration
func validateMonitorConfig(cfg *MonitorConfig) error {
	if cfg.LogFile == "" {
		return fmt.Errorf("log file not specified")
	}

	// Create log directory if it doesn't exist
	logDir := filepath.Dir(cfg.LogFile)
	if err := os.MkdirAll(logDir, 0755); err != nil {
		return fmt.Errorf("failed to create log directory: %v", err)
	}

	// Validate SNMP settings if enabled
	if cfg.SNMPEnabled {
		if cfg.SNMPPort <= 0 || cfg.SNMPPort > 65535 {
			return fmt.Errorf("invalid SNMP port: %d", cfg.SNMPPort)
		}
		if cfg.SNMPCommunity == "" {
			return fmt.Errorf("SNMP community string not specified")
		}
	}

	return nil
}

// setDefaults sets default values for optional configuration fields
func setDefaults(cfg *Config) {
	// Set default MTU if not specified
	if cfg.Network.MTU == 0 {
		cfg.Network.MTU = 1500
	}

	// Set default server settings
	if cfg.Mode == "server" {
		if cfg.Tunnel.MaxClients == 0 {
			cfg.Tunnel.MaxClients = 10
		}
	}

	// Set default rate limits if not specified
	if cfg.Tunnel.UploadKbps == 0 {
		cfg.Tunnel.UploadKbps = 10240 // 10 Mbps
	}
	if cfg.Tunnel.DownloadKbps == 0 {
		cfg.Tunnel.DownloadKbps = 10240 // 10 Mbps
	}

	// Set default SNMP port if enabled but not specified
	if cfg.Monitor.SNMPEnabled && cfg.Monitor.SNMPPort == 0 {
		cfg.Monitor.SNMPPort = 161
	}
}
