// Package config provides configuration management for the SSSonector service
package config

import (
	"path/filepath"

	"github.com/o3willard-AI/SSSonector/internal/config/interfaces"
	"github.com/o3willard-AI/SSSonector/internal/config/manager"
	"github.com/o3willard-AI/SSSonector/internal/config/store"
	"github.com/o3willard-AI/SSSonector/internal/config/types"
	"github.com/o3willard-AI/SSSonector/internal/config/validator"
)

// DefaultConfigDir is the default configuration directory
const DefaultConfigDir = "/etc/sssonector"

// Re-export types for external use
type (
	AppConfig       = types.AppConfig
	Config          = types.Config
	LoggingConfig   = types.LoggingConfig
	AuthConfig      = types.AuthConfig
	NetworkConfig   = types.NetworkConfig
	TunnelConfig    = types.TunnelConfig
	SecurityConfig  = types.SecurityConfig
	MonitorConfig   = types.MonitorConfig
	MetricsConfig   = types.MetricsConfig
	ThrottleConfig  = types.ThrottleConfig
	ConfigStore     = interfaces.ConfigStore
	ConfigValidator = interfaces.ConfigValidator
	ConfigManager   = interfaces.ConfigManager
	Type            = types.Type
)

// Re-export constants for external use
const (
	TypeServer = types.TypeServer
	TypeClient = types.TypeClient
	ModeServer = types.ModeServer
	ModeClient = types.ModeClient
)

// CreateManager creates a new configuration manager with default store and validator
func CreateManager(configDir string) ConfigManager {
	if configDir == "" {
		configDir = DefaultConfigDir
	}
	configDir = filepath.Clean(configDir)

	s := store.NewFileStore(configDir)
	v := validator.NewValidator()
	return manager.NewManager(s, v)
}

// CreateManagerWithOptions creates a new configuration manager with custom store and validator
func CreateManagerWithOptions(s ConfigStore, v ConfigValidator) ConfigManager {
	return manager.NewManager(s, v)
}

// CreateDefaultConfig returns a default configuration
func CreateDefaultConfig() *AppConfig {
	return types.DefaultConfig()
}

// CreateAppConfig creates a new AppConfig instance
func CreateAppConfig(configType Type) *AppConfig {
	return types.NewAppConfig(configType)
}
