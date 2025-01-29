package adapter

import (
	"fmt"
	"runtime"

	"go.uber.org/zap"
)

// Manager handles network interface creation and management
type Manager interface {
	// Create creates a new network interface
	Create(config *NetworkConfig) (Interface, error)
}

type manager struct {
	logger *zap.Logger
}

// NewManager creates a new interface manager
func NewManager(logger *zap.Logger) (Manager, error) {
	return &manager{
		logger: logger,
	}, nil
}

// Create creates a new network interface based on the current platform
func (m *manager) Create(config *NetworkConfig) (Interface, error) {
	switch runtime.GOOS {
	case "darwin":
		return NewDarwinInterface(m.logger, config)
	case "linux":
		return NewLinuxInterface(m.logger, config)
	case "windows":
		return NewWindowsInterface(m.logger, config)
	default:
		return nil, fmt.Errorf("unsupported platform: %s", runtime.GOOS)
	}
}
