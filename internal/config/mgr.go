// Package manager provides configuration management functionality
package config

import (
	"errors"
	"fmt"
	"io/fs"
	"sync"
)

// Manager implements ConfigManager interface
type Manager struct {
	store     ConfigStore
	validator ConfigValidator
	config    *AppConfig
	mu        sync.RWMutex
	watchers  []chan *AppConfig
}

// NewManager creates a new Manager instance
func NewManager(store ConfigStore, validator ConfigValidator) *Manager {
	return &Manager{
		store:     store,
		validator: validator,
		watchers:  make([]chan *AppConfig, 0),
	}
}

// Get returns the current configuration
func (m *Manager) Get() (*AppConfig, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if m.config == nil {
		config, err := m.store.Load()
		if err != nil {
			// A missing config file is not fatal: start from defaults so
			// first-run and test scenarios behave predictably. Any other
			// load error (permissions, corruption) still fails loudly.
			if errors.Is(err, fs.ErrNotExist) {
				m.config = DefaultConfig()
				return m.config, nil
			}
			return nil, fmt.Errorf("failed to load config: %w", err)
		}
		m.config = config
	}

	return m.config, nil
}

// Set sets a new configuration
func (m *Manager) Set(config *AppConfig) error {
	if err := m.validator.Validate(config); err != nil {
		return fmt.Errorf("invalid config: %v", err)
	}

	if err := m.store.Store(config); err != nil {
		return fmt.Errorf("failed to store config: %v", err)
	}

	m.mu.Lock()
	m.config = config
	m.mu.Unlock()

	m.notifyWatchers(config)
	return nil
}

// Update updates the current configuration
func (m *Manager) Update(config *AppConfig) error {
	if err := m.validator.Validate(config); err != nil {
		return fmt.Errorf("invalid config: %v", err)
	}

	m.mu.Lock()
	m.config = config
	m.mu.Unlock()

	if err := m.store.Store(config); err != nil {
		return fmt.Errorf("failed to store config: %v", err)
	}

	m.notifyWatchers(config)
	return nil
}

// Watch returns a channel that receives configuration updates
func (m *Manager) Watch() (<-chan *AppConfig, error) {
	ch := make(chan *AppConfig, 1)

	m.mu.Lock()
	m.watchers = append(m.watchers, ch)
	m.mu.Unlock()

	// Send current config immediately if available
	if m.config != nil {
		ch <- m.config
	}

	return ch, nil
}

// Close closes all watchers
func (m *Manager) Close() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	for _, ch := range m.watchers {
		close(ch)
	}
	m.watchers = nil
	return nil
}

// notifyWatchers notifies all watchers of a configuration change
func (m *Manager) notifyWatchers(config *AppConfig) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	for _, ch := range m.watchers {
		select {
		case ch <- config:
		default:
			// Skip if channel is blocked
		}
	}
}
