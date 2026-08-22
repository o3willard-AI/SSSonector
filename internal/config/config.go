// Package config provides configuration management for the SSSonector service
package config

import (
	"path/filepath"
)

// DefaultConfigDir is the default configuration directory
const DefaultConfigDir = "/etc/sssonector"

// CreateManager creates a new configuration manager with default store and validator
func CreateManager(configDir string) ConfigManager {
	if configDir == "" {
		configDir = DefaultConfigDir
	}
	configDir = filepath.Clean(configDir)

	s := NewFileStore(configDir)
	v := NewValidator()
	return NewManager(s, v)
}

// CreateManagerWithOptions creates a new configuration manager with custom store and validator
func CreateManagerWithOptions(s ConfigStore, v ConfigValidator) ConfigManager {
	return NewManager(s, v)
}

// CreateDefaultConfig returns a default configuration
func CreateDefaultConfig() *AppConfig {
	return DefaultConfig()
}

// CreateAppConfig creates a new AppConfig instance
func CreateAppConfig(configType Type) *AppConfig {
	return NewAppConfig(configType)
}

// CreateConfigLoader creates a new configuration loader
func CreateConfigLoader() *ConfigLoader {
	return NewConfigLoader()
}

// LoadConfigFile loads and upgrades a configuration file
func LoadConfigFile(filename string) (*AppConfig, error) {
	l := CreateConfigLoader()
	return l.LoadFile(filename)
}

// LoadConfigString loads and upgrades configuration from string
func LoadConfigString(content, format string) (*AppConfig, error) {
	l := CreateConfigLoader()
	return l.LoadFromString(content, format)
}
