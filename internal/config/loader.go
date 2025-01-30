package config

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/viper"
)

// LoadConfig loads the configuration from the specified file
func LoadConfig(configPath string) (*Config, error) {
	v := viper.New()
	v.SetConfigType("yaml")

	// Read config file
	configFile, err := os.Open(configPath)
	if err != nil {
		return nil, fmt.Errorf("error opening config file: %w", err)
	}
	defer configFile.Close()

	if err := v.ReadConfig(configFile); err != nil {
		return nil, fmt.Errorf("error reading config file: %w", err)
	}

	// Set default values
	setDefaults(v)

	// Unmarshal config
	var config Config
	if err := v.Unmarshal(&config); err != nil {
		return nil, fmt.Errorf("error unmarshaling config: %w", err)
	}

	// Validate config
	if err := validateConfig(&config); err != nil {
		return nil, fmt.Errorf("config validation error: %w", err)
	}

	// Resolve paths
	resolvePaths(&config, filepath.Dir(configPath))

	return &config, nil
}

// setDefaults sets default values for configuration
func setDefaults(v *viper.Viper) {
	v.SetDefault("network.mtu", 1500)
	v.SetDefault("tunnel.retry_attempts", 3)
	v.SetDefault("tunnel.retry_interval", 5)
	v.SetDefault("monitor.snmp_enabled", false)
	v.SetDefault("monitor.snmp_port", 161)
	v.SetDefault("monitor.snmp_community", "public")
	v.SetDefault("logging.level", "info")
	v.SetDefault("logging.max_size", 100)
	v.SetDefault("throttle.upload_kbps", 10240)   // 10 Mbps
	v.SetDefault("throttle.download_kbps", 10240) // 10 Mbps
}

// validateConfig validates the configuration
func validateConfig(cfg *Config) error {
	if cfg.Mode != "server" && cfg.Mode != "client" {
		return fmt.Errorf("invalid mode: %s (must be 'server' or 'client')", cfg.Mode)
	}

	if cfg.Network.Interface == "" {
		return fmt.Errorf("network interface is required")
	}

	if cfg.Network.Address == "" {
		return fmt.Errorf("network address is required")
	}

	if cfg.Network.MTU < 576 || cfg.Network.MTU > 9000 {
		return fmt.Errorf("invalid MTU: %d (must be between 576 and 9000)", cfg.Network.MTU)
	}

	if cfg.Tunnel.CertFile == "" {
		return fmt.Errorf("certificate file is required")
	}

	if cfg.Tunnel.KeyFile == "" {
		return fmt.Errorf("key file is required")
	}

	if cfg.Tunnel.CAFile == "" {
		return fmt.Errorf("CA file is required")
	}

	if cfg.Mode == "server" {
		if cfg.Tunnel.ListenAddress == "" {
			return fmt.Errorf("listen address is required in server mode")
		}
		if cfg.Tunnel.ListenPort <= 0 || cfg.Tunnel.ListenPort > 65535 {
			return fmt.Errorf("invalid listen port: %d", cfg.Tunnel.ListenPort)
		}
	} else {
		if cfg.Tunnel.ServerAddress == "" {
			return fmt.Errorf("server address is required in client mode")
		}
		if cfg.Tunnel.ServerPort <= 0 || cfg.Tunnel.ServerPort > 65535 {
			return fmt.Errorf("invalid server port: %d", cfg.Tunnel.ServerPort)
		}
	}

	return nil
}

// resolvePaths resolves relative paths in the configuration
func resolvePaths(cfg *Config, baseDir string) {
	if !filepath.IsAbs(cfg.Tunnel.CertFile) {
		cfg.Tunnel.CertFile = filepath.Join(baseDir, cfg.Tunnel.CertFile)
	}
	if !filepath.IsAbs(cfg.Tunnel.KeyFile) {
		cfg.Tunnel.KeyFile = filepath.Join(baseDir, cfg.Tunnel.KeyFile)
	}
	if !filepath.IsAbs(cfg.Tunnel.CAFile) {
		cfg.Tunnel.CAFile = filepath.Join(baseDir, cfg.Tunnel.CAFile)
	}
	if !filepath.IsAbs(cfg.Monitor.LogFile) {
		cfg.Monitor.LogFile = filepath.Join(baseDir, cfg.Monitor.LogFile)
	}
	if !filepath.IsAbs(cfg.Logging.FilePath) {
		cfg.Logging.FilePath = filepath.Join(baseDir, cfg.Logging.FilePath)
	}
}
