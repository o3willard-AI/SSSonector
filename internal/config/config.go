package config

import (
	"fmt"
	"os"
	"path/filepath"

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

	return cfg, nil
}

// resolvePath resolves a path relative to the config file
func resolvePath(configPath, path string) string {
	if filepath.IsAbs(path) {
		return path
	}
	return filepath.Join(filepath.Dir(configPath), path)
}
