package config

import (
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"go.uber.org/zap"
)

// Manager implements configuration management
type Manager struct {
	loader     *Loader
	store      ConfigStore
	validator  ConfigValidator
	logger     *zap.Logger
	configPath string
	mu         sync.RWMutex

	// Watchers for configuration changes
	watchers map[ConfigType][]chan *AppConfig
	// Subscribers for configuration change events
	subscribers map[ConfigType][]chan *ConfigChangeEvent
}

// NewManager creates a new configuration manager
func NewManager(configPath string, store ConfigStore, validator ConfigValidator, logger *zap.Logger) *Manager {
	return &Manager{
		loader:      NewLoader(logger),
		store:       store,
		validator:   validator,
		logger:      logger,
		configPath:  configPath,
		watchers:    make(map[ConfigType][]chan *AppConfig),
		subscribers: make(map[ConfigType][]chan *ConfigChangeEvent),
	}
}

// LoadConfig loads the current configuration
func (m *Manager) LoadConfig() (*AppConfig, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	return m.loader.LoadFromFile(m.configPath)
}

// SaveConfig saves the configuration
func (m *Manager) SaveConfig(cfg *AppConfig) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Save configuration to file
	if err := m.loader.SaveToFile(cfg, m.configPath); err != nil {
		return fmt.Errorf("failed to save config: %w", err)
	}

	// Store configuration
	if err := m.store.Store(cfg); err != nil {
		return fmt.Errorf("failed to store config: %w", err)
	}

	// Notify watchers
	m.notifyWatchers(cfg)

	return nil
}

// Watch starts watching for configuration changes
func (m *Manager) Watch(cfgType ConfigType) (<-chan *AppConfig, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	ch := make(chan *AppConfig, 1)
	m.watchers[cfgType] = append(m.watchers[cfgType], ch)

	return ch, nil
}

// StopWatch stops watching for configuration changes
func (m *Manager) StopWatch(cfgType ConfigType) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Close and remove all watchers for this type
	for _, ch := range m.watchers[cfgType] {
		close(ch)
	}
	delete(m.watchers, cfgType)

	return nil
}

// Subscribe subscribes to configuration change notifications
func (m *Manager) Subscribe(cfgType ConfigType) (<-chan *ConfigChangeEvent, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	ch := make(chan *ConfigChangeEvent, 1)
	m.subscribers[cfgType] = append(m.subscribers[cfgType], ch)

	return ch, nil
}

// Unsubscribe unsubscribes from configuration change notifications
func (m *Manager) Unsubscribe(cfgType ConfigType) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Close and remove all subscribers for this type
	for _, ch := range m.subscribers[cfgType] {
		close(ch)
	}
	delete(m.subscribers, cfgType)

	return nil
}

// Apply applies a configuration
func (m *Manager) Apply(cfg *AppConfig) error {
	// Load current configuration
	oldCfg, err := m.LoadConfig()
	if err != nil {
		return fmt.Errorf("failed to load current config: %w", err)
	}

	// Validate new configuration
	if err := m.validator.Validate(cfg); err != nil {
		return fmt.Errorf("invalid configuration: %w", err)
	}

	// Save new configuration
	if err := m.SaveConfig(cfg); err != nil {
		return fmt.Errorf("failed to save config: %w", err)
	}

	// Create change event
	event := &ConfigChangeEvent{
		Type:       TypeServer,
		OldVersion: oldCfg.Metadata.Version,
		NewVersion: cfg.Metadata.Version,
		ChangeType: "update",
		Timestamp:  time.Now(),
	}

	// Notify subscribers
	m.notifySubscribers(event)

	return nil
}

// Rollback rolls back to a previous configuration version
func (m *Manager) Rollback(cfgType ConfigType, version string) error {
	// Load previous configuration version
	cfg, err := m.store.Load(cfgType, version)
	if err != nil {
		return fmt.Errorf("failed to load config version: %w", err)
	}

	// Apply previous configuration
	if err := m.Apply(cfg); err != nil {
		return fmt.Errorf("failed to apply config: %w", err)
	}

	return nil
}

// Diff returns the differences between two configuration versions
func (m *Manager) Diff(cfgType ConfigType, version1, version2 string) (string, error) {
	// Load configurations
	cfg1, err := m.store.Load(cfgType, version1)
	if err != nil {
		return "", fmt.Errorf("failed to load config version 1: %w", err)
	}

	cfg2, err := m.store.Load(cfgType, version2)
	if err != nil {
		return "", fmt.Errorf("failed to load config version 2: %w", err)
	}

	// Marshal configurations to JSON for comparison
	json1, err := json.MarshalIndent(cfg1, "", "  ")
	if err != nil {
		return "", fmt.Errorf("failed to marshal config 1: %w", err)
	}

	json2, err := json.MarshalIndent(cfg2, "", "  ")
	if err != nil {
		return "", fmt.Errorf("failed to marshal config 2: %w", err)
	}

	// Compare JSON strings
	// In a real implementation, you would use a proper diff library
	return fmt.Sprintf("Version %s vs %s:\n%s\n---\n%s", version1, version2, json1, json2), nil
}

// Export exports a configuration to a specific format
func (m *Manager) Export(cfg *AppConfig, format ConfigFormat) ([]byte, error) {
	switch format {
	case FormatJSON:
		return json.MarshalIndent(cfg, "", "  ")
	case FormatYAML:
		// TODO: Implement YAML export
		return nil, fmt.Errorf("YAML export not implemented")
	case FormatTOML:
		// TODO: Implement TOML export
		return nil, fmt.Errorf("TOML export not implemented")
	default:
		return nil, fmt.Errorf("unsupported format: %s", format)
	}
}

// Import imports a configuration from a specific format
func (m *Manager) Import(data []byte, format ConfigFormat) (*AppConfig, error) {
	cfg := DefaultConfig()

	switch format {
	case FormatJSON:
		if err := json.Unmarshal(data, cfg); err != nil {
			return nil, fmt.Errorf("failed to parse JSON: %w", err)
		}
	case FormatYAML:
		// TODO: Implement YAML import
		return nil, fmt.Errorf("YAML import not implemented")
	case FormatTOML:
		// TODO: Implement TOML import
		return nil, fmt.Errorf("TOML import not implemented")
	default:
		return nil, fmt.Errorf("unsupported format: %s", format)
	}

	return cfg, nil
}

// notifyWatchers notifies all watchers of configuration changes
func (m *Manager) notifyWatchers(cfg *AppConfig) {
	for _, watchers := range m.watchers {
		for _, ch := range watchers {
			select {
			case ch <- cfg:
			default:
				// Channel is full, skip notification
			}
		}
	}
}

// notifySubscribers notifies all subscribers of configuration changes
func (m *Manager) notifySubscribers(event *ConfigChangeEvent) {
	for _, subscribers := range m.subscribers {
		for _, ch := range subscribers {
			select {
			case ch <- event:
			default:
				// Channel is full, skip notification
			}
		}
	}
}

// GetStore returns the configuration store
func (m *Manager) GetStore() ConfigStore {
	return m.store
}

// GetValidator returns the configuration validator
func (m *Manager) GetValidator() ConfigValidator {
	return m.validator
}

// GetWatcher returns the configuration watcher
func (m *Manager) GetWatcher() ConfigWatcher {
	return m
}
