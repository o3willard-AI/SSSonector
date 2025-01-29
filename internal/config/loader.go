package config

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/spf13/viper"
)

const (
	envPrefix = "SSL_TUNNEL"
)

// Load reads and parses the configuration from the specified file
func Load(configFile string) (*Config, error) {
	v := viper.New()

	// Set up Viper for file reading
	v.SetConfigFile(configFile)
	ext := filepath.Ext(configFile)
	v.SetConfigType(strings.TrimPrefix(ext, "."))

	// Set up environment variables
	v.SetEnvPrefix(envPrefix)
	v.AutomaticEnv()
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))

	// Read configuration file
	if err := v.ReadInConfig(); err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	var cfg Config
	if err := v.Unmarshal(&cfg); err != nil {
		return nil, fmt.Errorf("failed to unmarshal config: %w", err)
	}

	if err := validateConfig(&cfg); err != nil {
		return nil, fmt.Errorf("invalid configuration: %w", err)
	}

	return &cfg, nil
}

// validateConfig performs validation on the configuration
func validateConfig(cfg *Config) error {
	if cfg.Mode != "server" && cfg.Mode != "client" {
		return fmt.Errorf("invalid mode: %s (must be 'server' or 'client')", cfg.Mode)
	}

	if err := validateNetworkConfig(&cfg.Network, cfg.Mode); err != nil {
		return fmt.Errorf("network configuration error: %w", err)
	}

	if err := validateTLSConfig(&cfg.TLS); err != nil {
		return fmt.Errorf("TLS configuration error: %w", err)
	}

	return nil
}

// validateNetworkConfig validates network-related configuration
func validateNetworkConfig(cfg *NetworkConfig, mode string) error {
	if cfg.Interface == "" {
		return fmt.Errorf("interface name is required")
	}

	if cfg.Address == "" {
		return fmt.Errorf("interface address is required")
	}

	if cfg.MTU < 576 || cfg.MTU > 9000 {
		return fmt.Errorf("invalid MTU: %d (must be between 576 and 9000)", cfg.MTU)
	}

	switch mode {
	case "server":
		if cfg.ListenPort < 1 || cfg.ListenPort > 65535 {
			return fmt.Errorf("invalid listen port: %d (must be between 1 and 65535)", cfg.ListenPort)
		}
		if cfg.MaxClients < 1 {
			return fmt.Errorf("max clients must be at least 1")
		}
	case "client":
		if cfg.ServerPort < 1 || cfg.ServerPort > 65535 {
			return fmt.Errorf("invalid server port: %d (must be between 1 and 65535)", cfg.ServerPort)
		}
		if cfg.ServerAddress == "" {
			return fmt.Errorf("server address is required")
		}
		if cfg.RetryAttempts < 0 {
			return fmt.Errorf("retry attempts cannot be negative")
		}
		if cfg.RetryInterval < 1 {
			return fmt.Errorf("retry interval must be at least 1 second")
		}
	}

	return nil
}

// validateTLSConfig validates TLS-related configuration
func validateTLSConfig(cfg *TLSConfig) error {
	if !cfg.AutoGenerate {
		if cfg.CertFile == "" {
			return fmt.Errorf("TLS certificate file is required when auto-generate is disabled")
		}
		if cfg.KeyFile == "" {
			return fmt.Errorf("TLS key file is required when auto-generate is disabled")
		}
	}

	if cfg.KeySize < 2048 {
		return fmt.Errorf("key size must be at least 2048 bits")
	}

	if cfg.ValidityDays < 1 {
		return fmt.Errorf("validity days must be at least 1")
	}

	return nil
}
