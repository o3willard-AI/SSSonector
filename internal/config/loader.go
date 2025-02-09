package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"go.uber.org/zap"
)

// Loader handles configuration loading and validation
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
	// Read configuration file
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	// Parse configuration
	cfg := DefaultConfig()
	if err := json.Unmarshal(data, cfg); err != nil {
		return nil, fmt.Errorf("failed to parse config file: %w", err)
	}

	// Validate configuration
	if err := l.validateConfig(cfg); err != nil {
		return nil, fmt.Errorf("invalid configuration: %w", err)
	}

	// Update paths to be absolute
	if err := l.resolveConfigPaths(cfg, filepath.Dir(path)); err != nil {
		return nil, fmt.Errorf("failed to resolve paths: %w", err)
	}

	return cfg, nil
}

// validateConfig validates the configuration
func (l *Loader) validateConfig(cfg *AppConfig) error {
	// Validate mode
	if cfg.Mode != ModeServer && cfg.Mode != ModeClient {
		return fmt.Errorf("invalid mode: %s", cfg.Mode)
	}

	// Validate network configuration
	if cfg.Network.MTU < 576 || cfg.Network.MTU > 65535 {
		return fmt.Errorf("invalid MTU: %d", cfg.Network.MTU)
	}

	// Validate tunnel configuration
	if cfg.Mode == ModeServer {
		if cfg.Tunnel.ListenPort <= 0 || cfg.Tunnel.ListenPort > 65535 {
			return fmt.Errorf("invalid listen port: %d", cfg.Tunnel.ListenPort)
		}
	} else {
		if cfg.Tunnel.ServerPort <= 0 || cfg.Tunnel.ServerPort > 65535 {
			return fmt.Errorf("invalid server port: %d", cfg.Tunnel.ServerPort)
		}
	}

	// Validate monitor configuration
	if cfg.Monitor.Enabled {
		if cfg.Monitor.SNMPEnabled {
			if cfg.Monitor.SNMPPort <= 0 || cfg.Monitor.SNMPPort > 65535 {
				return fmt.Errorf("invalid SNMP port: %d", cfg.Monitor.SNMPPort)
			}
		}
		if cfg.Monitor.Prometheus.Enabled {
			if cfg.Monitor.Prometheus.Port <= 0 || cfg.Monitor.Prometheus.Port > 65535 {
				return fmt.Errorf("invalid Prometheus port: %d", cfg.Monitor.Prometheus.Port)
			}
		}
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

	// Resolve certificate paths
	cfg.Tunnel.CertFile = resolvePath(cfg.Tunnel.CertFile)
	cfg.Tunnel.KeyFile = resolvePath(cfg.Tunnel.KeyFile)
	cfg.Tunnel.CAFile = resolvePath(cfg.Tunnel.CAFile)

	// Resolve log file path
	cfg.Monitor.LogFile = resolvePath(cfg.Monitor.LogFile)

	return nil
}

// SaveToFile saves configuration to a file
func (l *Loader) SaveToFile(cfg *AppConfig, path string) error {
	// Create parent directory if it doesn't exist
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
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

// LoadFromEnv loads configuration from environment variables
func (l *Loader) LoadFromEnv() (*AppConfig, error) {
	cfg := DefaultConfig()

	// Helper function to get environment variable
	getEnv := func(key, defaultValue string) string {
		if value := os.Getenv(key); value != "" {
			return value
		}
		return defaultValue
	}

	// Load mode
	if mode := getEnv("SSSONECTOR_MODE", ""); mode != "" {
		cfg.Mode = Mode(strings.ToLower(mode))
	}

	// Load network configuration
	cfg.Network.Interface = getEnv("SSSONECTOR_NETWORK_INTERFACE", cfg.Network.Interface)
	cfg.Network.IPAddress = getEnv("SSSONECTOR_NETWORK_IP", cfg.Network.IPAddress)
	cfg.Network.SubnetMask = getEnv("SSSONECTOR_NETWORK_MASK", cfg.Network.SubnetMask)

	// Load tunnel configuration
	cfg.Tunnel.ServerAddress = getEnv("SSSONECTOR_TUNNEL_SERVER", cfg.Tunnel.ServerAddress)
	cfg.Tunnel.Protocol = getEnv("SSSONECTOR_TUNNEL_PROTOCOL", cfg.Tunnel.Protocol)
	cfg.Tunnel.Encryption = getEnv("SSSONECTOR_TUNNEL_ENCRYPTION", cfg.Tunnel.Encryption)

	// Load monitor configuration
	if enabled := getEnv("SSSONECTOR_MONITOR_ENABLED", ""); enabled != "" {
		cfg.Monitor.Enabled = enabled == "true"
	}
	cfg.Monitor.LogLevel = getEnv("SSSONECTOR_LOG_LEVEL", cfg.Monitor.LogLevel)

	// Validate configuration
	if err := l.validateConfig(cfg); err != nil {
		return nil, fmt.Errorf("invalid configuration: %w", err)
	}

	return cfg, nil
}
