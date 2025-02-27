package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/o3willard-AI/SSSonector/internal/config/types"
	"gopkg.in/yaml.v2"
)

// LoadConfig loads configuration from the specified file
func LoadConfig(path string) (*types.AppConfig, error) {
	// Read configuration file
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	// Parse configuration
	cfg := types.DefaultConfig()
	if err := yaml.Unmarshal(data, cfg); err != nil {
		return nil, fmt.Errorf("failed to parse config file: %w", err)
	}

	// Create state directory if it doesn't exist
	if cfg.Config.StateDir == "" {
		cfg.Config.StateDir = "/var/lib/sssonector"
	}
	if err := os.MkdirAll(cfg.Config.StateDir, 0700); err != nil {
		return nil, fmt.Errorf("failed to create state directory: %w", err)
	}

	// Create log directory if it doesn't exist
	if cfg.Config.LogDir == "" {
		cfg.Config.LogDir = "/var/log/sssonector"
	}
	if err := os.MkdirAll(cfg.Config.LogDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create log directory: %w", err)
	}

	// Resolve certificate paths
	if cfg.Config.TLS != nil {
		if cfg.Config.TLS.CertFile != "" {
			cfg.Config.TLS.CertFile = resolvePath(path, cfg.Config.TLS.CertFile)
		}
		if cfg.Config.TLS.KeyFile != "" {
			cfg.Config.TLS.KeyFile = resolvePath(path, cfg.Config.TLS.KeyFile)
		}
		if cfg.Config.TLS.CAFile != "" {
			cfg.Config.TLS.CAFile = resolvePath(path, cfg.Config.TLS.CAFile)
		}
	}

	// Set configuration directory
	cfg.ConfigDir = filepath.Dir(path)

	// Set adapter configuration
	cfg.Adapter = &types.AdapterConfig{
		RetryAttempts:  3,
		RetryDelay:     types.NewDuration(1 * time.Second),
		CleanupTimeout: types.NewDuration(30 * time.Second),
	}

	// Ensure all configurations are properly initialized with defaults
	if cfg.Config != nil {
		// Network configuration
		if cfg.Config.Network == nil {
			cfg.Config.Network = types.NewNetworkConfig()
		}
		if cfg.Config.Network.Interface == "" {
			cfg.Config.Network.Interface = "tun0"
		}
		if cfg.Config.Network.MTU == 0 {
			cfg.Config.Network.MTU = 1500
		}
		// Ensure address is set and in CIDR format
		if cfg.Config.Network.Address == "" {
			if cfg.Config.Mode == types.ModeServer {
				cfg.Config.Network.Address = "10.0.0.1/24"
			} else {
				cfg.Config.Network.Address = "10.0.0.2/24"
			}
		} else if !strings.Contains(cfg.Config.Network.Address, "/") {
			cfg.Config.Network.Address = cfg.Config.Network.Address + "/24"
		}

		// Tunnel configuration
		if cfg.Config.Tunnel == nil {
			cfg.Config.Tunnel = types.NewTunnelConfig()
		}
		if cfg.Config.Tunnel.ListenPort == 0 {
			cfg.Config.Tunnel.ListenPort = 8080
		}
		if cfg.Config.Tunnel.ListenAddress == "" {
			cfg.Config.Tunnel.ListenAddress = "0.0.0.0"
		}
		if cfg.Config.Tunnel.Protocol == "" {
			cfg.Config.Tunnel.Protocol = "tcp"
		}
		if cfg.Config.Tunnel.MaxClients == 0 {
			cfg.Config.Tunnel.MaxClients = 1000
		}
	}

	return cfg, nil
}

// resolvePath resolves a path relative to the config file
func resolvePath(configPath, path string) string {
	if filepath.IsAbs(path) {
		return path
	}
	return filepath.Join(filepath.Dir(configPath), path)
}
