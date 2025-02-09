// Package manager provides configuration management functionality
package manager

import (
	"fmt"
	"sync"

	"github.com/o3willard-AI/SSSonector/internal/config/interfaces"
	"github.com/o3willard-AI/SSSonector/internal/config/types"
)

// Manager implements ConfigManager interface
type Manager struct {
	store     interfaces.ConfigStore
	validator interfaces.ConfigValidator
	config    *types.AppConfig
	mu        sync.RWMutex
	watchers  []chan *types.AppConfig
}

// NewManager creates a new Manager instance
func NewManager(store interfaces.ConfigStore, validator interfaces.ConfigValidator) *Manager {
	return &Manager{
		store:     store,
		validator: validator,
		watchers:  make([]chan *types.AppConfig, 0),
	}
}

// Get returns the current configuration
func (m *Manager) Get() (*types.AppConfig, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if m.config == nil {
		config, err := m.store.Load()
		if err != nil {
			return nil, fmt.Errorf("failed to load config: %v", err)
		}
		m.config = config
	}

	return m.config, nil
}

// Set sets a new configuration
func (m *Manager) Set(config *types.AppConfig) error {
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
func (m *Manager) Update(config *types.AppConfig) error {
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
func (m *Manager) Watch() (<-chan *types.AppConfig, error) {
	ch := make(chan *types.AppConfig, 1)

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
func (m *Manager) notifyWatchers(config *types.AppConfig) {
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
