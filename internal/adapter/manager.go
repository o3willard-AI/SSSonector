package adapter

import (
	"fmt"
	"runtime"
)

// Manager handles virtual interface operations
type Manager struct {
	iface Interface
}

// NewManager creates a new interface manager
func NewManager(cfg *Config) (*Manager, error) {
	if cfg.Name == "" {
		return nil, fmt.Errorf("interface name is required")
	}

	iface, err := newPlatformInterface(cfg.Name)
	if err != nil {
		return nil, fmt.Errorf("failed to create platform interface: %w", err)
	}

	return &Manager{
		iface: iface,
	}, nil
}

// Create creates and configures the virtual interface
func (m *Manager) Create(cfg *Config) error {
	if err := m.iface.Create(); err != nil {
		return fmt.Errorf("failed to create interface: %w", err)
	}

	if err := m.iface.Configure(cfg); err != nil {
		return fmt.Errorf("failed to configure interface: %w", err)
	}

	if err := m.iface.Up(); err != nil {
		return fmt.Errorf("failed to bring interface up: %w", err)
	}

	return nil
}

// Close cleans up the virtual interface
func (m *Manager) Close() error {
	if err := m.iface.Down(); err != nil {
		return fmt.Errorf("failed to bring interface down: %w", err)
	}

	if err := m.iface.Delete(); err != nil {
		return fmt.Errorf("failed to delete interface: %w", err)
	}

	return nil
}

// GetInterface returns the underlying interface
func (m *Manager) GetInterface() Interface {
	return m.iface
}

// PlatformSupported returns whether the current platform is supported
func PlatformSupported() bool {
	switch runtime.GOOS {
	case "linux", "darwin", "windows":
		return true
	default:
		return false
	}
}

// GetPlatformName returns the current platform name
func GetPlatformName() string {
	return runtime.GOOS
}
