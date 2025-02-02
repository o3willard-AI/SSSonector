package config

import (
	"fmt"
	"io/ioutil"

	"gopkg.in/yaml.v3"
)

// Load reads and parses the configuration file
func Load(path string) (*Config, error) {
	data, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	cfg := &Config{}
	if err := yaml.Unmarshal(data, cfg); err != nil {
		return nil, fmt.Errorf("failed to parse config file: %w", err)
	}

	if err := validate(cfg); err != nil {
		return nil, fmt.Errorf("config validation error: %w", err)
	}

	return cfg, nil
}

// validate checks the configuration for required fields and valid values
func validate(cfg *Config) error {
	if cfg.Mode != "server" && cfg.Mode != "client" {
		return fmt.Errorf("invalid mode: %s (must be 'server' or 'client')", cfg.Mode)
	}

	if cfg.Network.Interface == "" {
		return fmt.Errorf("network interface is required")
	}

	if cfg.Network.Address == "" {
		return fmt.Errorf("network address is required")
	}

	if cfg.Network.MTU < 1280 || cfg.Network.MTU > 9000 {
		return fmt.Errorf("invalid MTU: %d (must be between 1280 and 9000)", cfg.Network.MTU)
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
		if cfg.Tunnel.ListenPort < 1 || cfg.Tunnel.ListenPort > 65535 {
			return fmt.Errorf("invalid listen port: %d", cfg.Tunnel.ListenPort)
		}
		if cfg.Tunnel.MaxClients < 1 {
			return fmt.Errorf("max clients must be greater than 0")
		}
	} else {
		if cfg.Tunnel.ServerAddress == "" {
			return fmt.Errorf("server address is required in client mode")
		}
		if cfg.Tunnel.ServerPort < 1 || cfg.Tunnel.ServerPort > 65535 {
			return fmt.Errorf("invalid server port: %d", cfg.Tunnel.ServerPort)
		}
	}

	if cfg.Tunnel.UploadKbps < 0 {
		return fmt.Errorf("upload bandwidth limit cannot be negative")
	}

	if cfg.Tunnel.DownloadKbps < 0 {
		return fmt.Errorf("download bandwidth limit cannot be negative")
	}

	return nil
}
