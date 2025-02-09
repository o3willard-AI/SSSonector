// Package interfaces defines the core interfaces for configuration management
package interfaces

import (
	"github.com/o3willard-AI/SSSonector/internal/config/types"
)

// ConfigStore defines the interface for configuration storage
type ConfigStore interface {
	// Load loads the latest configuration from storage
	Load() (*types.AppConfig, error)
	// Store stores the configuration to storage
	Store(*types.AppConfig) error
	// ListVersions lists all available configuration versions for a given type
	ListVersions(configType types.Type) ([]string, error)
}

// ConfigValidator defines the interface for configuration validation
type ConfigValidator interface {
	// Validate validates the configuration
	Validate(*types.AppConfig) error
}

// ConfigManager defines the interface for configuration management
type ConfigManager interface {
	// Get returns the current configuration
	Get() (*types.AppConfig, error)
	// Set sets a new configuration
	Set(*types.AppConfig) error
	// Update updates the current configuration
	Update(*types.AppConfig) error
	// Watch returns a channel that receives configuration updates
	Watch() (<-chan *types.AppConfig, error)
	// Close closes all watchers
	Close() error
}
