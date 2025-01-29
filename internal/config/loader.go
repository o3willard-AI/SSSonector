package config

import (
	"fmt"
	"net"
	"os"
	"path/filepath"

	"github.com/spf13/viper"
)

// LoadConfig loads configuration from the specified file
func LoadConfig(configFile string) (*Config, error) {
	viper.SetConfigFile(configFile)

	if err := viper.ReadInConfig(); err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	var config Config
	if err := viper.Unmarshal(&config); err != nil {
		return nil, fmt.Errorf("failed to unmarshal config: %w", err)
	}

	if err := validateConfig(&config); err != nil {
		return nil, fmt.Errorf("invalid configuration: %w", err)
	}

	// Resolve certificate paths
	config.Tunnel.CertFile = resolvePath(configFile, config.Tunnel.CertFile)
	config.Tunnel.KeyFile = resolvePath(configFile, config.Tunnel.KeyFile)
	if config.Tunnel.CAFile != "" {
		config.Tunnel.CAFile = resolvePath(configFile, config.Tunnel.CAFile)
	}

	return &config, nil
}

// validateConfig validates the configuration values
func validateConfig(config *Config) error {
	// Validate mode
	if config.Mode != "server" && config.Mode != "client" {
		return fmt.Errorf("invalid mode: %s (must be 'server' or 'client')", config.Mode)
	}

	// Validate network configuration
	if config.Network.MTU < 1280 || config.Network.MTU > 9000 {
		return fmt.Errorf("invalid MTU: %d (must be between 1280 and 9000)", config.Network.MTU)
	}

	if config.Network.Address != "" {
		if _, _, err := net.ParseCIDR(config.Network.Address); err != nil {
			return fmt.Errorf("invalid network address: %s", config.Network.Address)
		}
	}

	// Validate tunnel configuration
	if config.Mode == "server" {
		if config.Tunnel.ListenPort < 1 || config.Tunnel.ListenPort > 65535 {
			return fmt.Errorf("invalid listen port: %d", config.Tunnel.ListenPort)
		}
	} else {
		if config.Tunnel.ServerPort < 1 || config.Tunnel.ServerPort > 65535 {
			return fmt.Errorf("invalid server port: %d", config.Tunnel.ServerPort)
		}
		if config.Tunnel.ServerAddress == "" {
			return fmt.Errorf("server address is required in client mode")
		}
	}

	if config.Tunnel.RetryAttempts < 0 {
		return fmt.Errorf("invalid retry attempts: %d", config.Tunnel.RetryAttempts)
	}

	if config.Tunnel.RetryInterval < 0 {
		return fmt.Errorf("invalid retry interval: %d", config.Tunnel.RetryInterval)
	}

	// Validate certificate files
	if !fileExists(config.Tunnel.CertFile) {
		return fmt.Errorf("certificate file not found: %s", config.Tunnel.CertFile)
	}

	if !fileExists(config.Tunnel.KeyFile) {
		return fmt.Errorf("private key file not found: %s", config.Tunnel.KeyFile)
	}

	if config.Tunnel.CAFile != "" && !fileExists(config.Tunnel.CAFile) {
		return fmt.Errorf("CA certificate file not found: %s", config.Tunnel.CAFile)
	}

	// Validate monitor configuration
	if config.Monitor.SNMPEnabled {
		if config.Monitor.SNMPPort < 1 || config.Monitor.SNMPPort > 65535 {
			return fmt.Errorf("invalid SNMP port: %d", config.Monitor.SNMPPort)
		}
		if config.Monitor.SNMPCommunity == "" {
			return fmt.Errorf("SNMP community string is required when SNMP is enabled")
		}
	}

	return nil
}

// resolvePath resolves a path relative to the config file location
func resolvePath(configFile, path string) string {
	if filepath.IsAbs(path) {
		return path
	}
	return filepath.Join(filepath.Dir(configFile), path)
}

// fileExists checks if a file exists and is not a directory
func fileExists(filename string) bool {
	info, err := os.Stat(filename)
	if os.IsNotExist(err) {
		return false
	}
	return !info.IsDir()
}
