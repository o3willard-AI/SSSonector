package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"go.uber.org/zap"
)

// Loader implements configuration loading
type Loader struct {
	logger *zap.Logger
}

// NewLoader creates a new configuration loader
func NewLoader(logger *zap.Logger) *Loader {
	return &Loader{
		logger: logger,
	}
}

// LoadFromFile loads configuration from a file
func (l *Loader) LoadFromFile(path string) (*AppConfig, error) {
	if path == "" {
		l.logger.Info("No configuration file specified, using defaults")
		return DefaultConfig(), nil
	}

	// Read configuration file
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			l.logger.Info("Configuration file not found, using defaults",
				zap.String("path", path),
			)
			return DefaultConfig(), nil
		}
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	// Parse configuration
	cfg := &AppConfig{}
	if err := json.Unmarshal(data, cfg); err != nil {
		return nil, fmt.Errorf("failed to parse config file: %w", err)
	}

	// Update paths to be absolute
	if err := l.resolveConfigPaths(cfg, filepath.Dir(path)); err != nil {
		return nil, fmt.Errorf("failed to resolve paths: %w", err)
	}

	return cfg, nil
}

// SaveToFile saves configuration to a file
func (l *Loader) SaveToFile(cfg *AppConfig, path string) error {
	// Create directory if it doesn't exist
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	// Marshal configuration
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	// Write configuration file
	if err := os.WriteFile(path, data, 0644); err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}

	return nil
}

// resolveConfigPaths resolves relative paths in configuration
func (l *Loader) resolveConfigPaths(cfg *AppConfig, baseDir string) error {
	// Helper function to resolve path
	resolvePath := func(path string) string {
		if path == "" || filepath.IsAbs(path) {
			return path
		}
		return filepath.Join(baseDir, path)
	}

	// Update certificate paths
	cfg.Tunnel.CertFile = resolvePath(cfg.Tunnel.CertFile)
	cfg.Tunnel.KeyFile = resolvePath(cfg.Tunnel.KeyFile)
	cfg.Tunnel.CAFile = resolvePath(cfg.Tunnel.CAFile)

	// Update monitor paths
	if cfg.Monitor.Enabled {
		// Add any monitor-specific paths here
	}

	// Update security paths
	if len(cfg.Security.ACLs) > 0 {
		// Add any security-specific paths here
	}

	return nil
}

// LoadFromEnv loads configuration from environment variables
func (l *Loader) LoadFromEnv() (*AppConfig, error) {
	cfg := DefaultConfig()

	// Load environment variables
	if mode := os.Getenv("SSSONECTOR_MODE"); mode != "" {
		cfg.Mode = Mode(strings.ToLower(mode))
	}

	if mtu := os.Getenv("SSSONECTOR_NETWORK_MTU"); mtu != "" {
		// Parse MTU value
	}

	if proto := os.Getenv("SSSONECTOR_TUNNEL_PROTOCOL"); proto != "" {
		cfg.Tunnel.Protocol = strings.ToLower(proto)
	}

	// Add more environment variable mappings as needed

	return cfg, nil
}

// LoadFromString loads configuration from a string
func (l *Loader) LoadFromString(data string) (*AppConfig, error) {
	cfg := &AppConfig{}
	if err := json.Unmarshal([]byte(data), cfg); err != nil {
		return nil, fmt.Errorf("failed to parse config string: %w", err)
	}
	return cfg, nil
}
