// Package interfaces defines the core interfaces for configuration management
package config

// ConfigStore defines the interface for configuration storage
type ConfigStore interface {
	// Load loads the latest configuration from storage
	Load() (*AppConfig, error)
	// Store stores the configuration to storage
	Store(*AppConfig) error
	// ListVersions lists all available configuration versions for a given type
	ListVersions(configType Type) ([]string, error)
}

// ConfigValidator defines the interface for configuration validation
type ConfigValidator interface {
	// Validate validates the configuration
	Validate(*AppConfig) error
}

// ConfigManager defines the interface for configuration management
type ConfigManager interface {
	// Get returns the current configuration
	Get() (*AppConfig, error)
	// Set sets a new configuration
	Set(*AppConfig) error
	// Update updates the current configuration
	Update(*AppConfig) error
	// Watch returns a channel that receives configuration updates
	Watch() (<-chan *AppConfig, error)
	// Close closes all watchers
	Close() error
}
