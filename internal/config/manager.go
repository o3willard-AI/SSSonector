package config

import (
	"fmt"
	"os"
	"sync"
	"time"

	"github.com/fsnotify/fsnotify"
	"go.uber.org/zap"
)

// ConfigWatcher defines the interface for components that need to react to config changes
type ConfigWatcher interface {
	// OnConfigUpdate is called when configuration changes are detected
	// Return error if the component cannot apply the new configuration
	OnConfigUpdate(*Config) error
}

// ConfigManager handles dynamic configuration updates
type ConfigManager struct {
	mu            sync.RWMutex
	currentConfig *Config
	configPath    string
	watchers      []ConfigWatcher
	reloadChan    chan struct{}
	logger        *zap.Logger
	watcher       *fsnotify.Watcher
	done          chan struct{}
}

// NewConfigManager creates a new configuration manager
func NewConfigManager(configPath string, logger *zap.Logger) (*ConfigManager, error) {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, fmt.Errorf("failed to create file watcher: %w", err)
	}

	// Load initial configuration
	cfg, err := LoadConfig(configPath)
	if err != nil {
		watcher.Close()
		return nil, fmt.Errorf("failed to load initial config: %w", err)
	}

	manager := &ConfigManager{
		currentConfig: cfg,
		configPath:    configPath,
		reloadChan:    make(chan struct{}, 1),
		logger:        logger,
		watcher:       watcher,
		done:          make(chan struct{}),
		watchers:      make([]ConfigWatcher, 0),
	}

	// Start watching config file
	if err := watcher.Add(configPath); err != nil {
		watcher.Close()
		return nil, fmt.Errorf("failed to watch config file: %w", err)
	}

	go manager.watchConfig()

	return manager, nil
}

// RegisterWatcher adds a component that needs to be notified of config changes
func (m *ConfigManager) RegisterWatcher(w ConfigWatcher) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.watchers = append(m.watchers, w)
}

// GetConfig returns the current configuration
func (m *ConfigManager) GetConfig() *Config {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.currentConfig
}

// Close stops the configuration manager
func (m *ConfigManager) Close() error {
	close(m.done)
	return m.watcher.Close()
}

// watchConfig monitors the configuration file for changes
func (m *ConfigManager) watchConfig() {
	for {
		select {
		case event, ok := <-m.watcher.Events:
			if !ok {
				return
			}
			if event.Op&fsnotify.Write == fsnotify.Write {
				m.handleConfigChange()
			}
		case err, ok := <-m.watcher.Errors:
			if !ok {
				return
			}
			m.logger.Error("Config watcher error", zap.Error(err))
		case <-m.reloadChan:
			m.handleConfigChange()
		case <-m.done:
			return
		}
	}
}

// TriggerReload manually triggers a configuration reload
func (m *ConfigManager) TriggerReload() {
	select {
	case m.reloadChan <- struct{}{}:
	default:
		// Channel is full, reload already pending
	}
}

// handleConfigChange handles a configuration file change
func (m *ConfigManager) handleConfigChange() {
	// Add small delay to ensure file is fully written
	time.Sleep(100 * time.Millisecond)

	// Load new configuration
	newConfig, err := LoadConfig(m.configPath)
	if err != nil {
		m.logger.Error("Failed to load new config",
			zap.String("path", m.configPath),
			zap.Error(err))
		return
	}

	// Validate configuration change
	if err := m.validateConfigChange(newConfig); err != nil {
		m.logger.Error("Invalid configuration change",
			zap.Error(err))
		return
	}

	// Apply new configuration
	if err := m.applyNewConfig(newConfig); err != nil {
		m.logger.Error("Failed to apply new config",
			zap.Error(err))
		return
	}

	m.logger.Info("Configuration reloaded successfully")
}

// validateConfigChange checks if the new configuration is valid
func (m *ConfigManager) validateConfigChange(newConfig *Config) error {
	m.mu.RLock()
	oldConfig := m.currentConfig
	m.mu.RUnlock()

	// Validate mode hasn't changed
	if newConfig.Mode != oldConfig.Mode {
		return fmt.Errorf("cannot change mode during hot reload")
	}

	// Validate certificate paths exist if changed
	if newConfig.Tunnel.CertFile != oldConfig.Tunnel.CertFile {
		if _, err := os.Stat(newConfig.Tunnel.CertFile); err != nil {
			return fmt.Errorf("certificate file not accessible: %w", err)
		}
	}
	if newConfig.Tunnel.KeyFile != oldConfig.Tunnel.KeyFile {
		if _, err := os.Stat(newConfig.Tunnel.KeyFile); err != nil {
			return fmt.Errorf("key file not accessible: %w", err)
		}
	}
	if newConfig.Tunnel.CAFile != oldConfig.Tunnel.CAFile {
		if _, err := os.Stat(newConfig.Tunnel.CAFile); err != nil {
			return fmt.Errorf("CA file not accessible: %w", err)
		}
	}

	// Validate rate limits are within acceptable ranges
	if newConfig.Tunnel.UploadKbps < 0 {
		return fmt.Errorf("invalid upload rate limit: %d", newConfig.Tunnel.UploadKbps)
	}
	if newConfig.Tunnel.DownloadKbps < 0 {
		return fmt.Errorf("invalid download rate limit: %d", newConfig.Tunnel.DownloadKbps)
	}

	return nil
}

// applyNewConfig applies the new configuration to all registered components
func (m *ConfigManager) applyNewConfig(newConfig *Config) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Notify all watchers
	for _, w := range m.watchers {
		if err := w.OnConfigUpdate(newConfig); err != nil {
			return fmt.Errorf("failed to update component: %w", err)
		}
	}

	// Update current configuration
	m.currentConfig = newConfig
	return nil
}
