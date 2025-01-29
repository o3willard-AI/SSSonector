package adapter

import (
	"fmt"
	"runtime"

	"github.com/o3willard-AI/SSSonector/internal/config"
	"go.uber.org/zap"
)

// Manager handles network interface creation and management
type Manager struct {
	logger *zap.Logger
	config *config.NetworkConfig
}

// NewManager creates a new interface manager
func NewManager(logger *zap.Logger, cfg *config.NetworkConfig) *Manager {
	return &Manager{
		logger: logger,
		config: cfg,
	}
}

// CreateInterface creates a new network interface based on the current OS
func (m *Manager) CreateInterface() (Interface, error) {
	m.logger.Info("Creating network interface",
		zap.String("os", runtime.GOOS),
		zap.String("interface", m.config.Interface),
	)

	var iface Interface
	var err error

	switch runtime.GOOS {
	case "linux":
		iface, err = newLinuxInterface(m.config.Interface)
	case "darwin":
		iface, err = newDarwinInterface(m.config.Interface)
	case "windows":
		iface, err = newWindowsInterface(m.config.Interface)
	default:
		return nil, fmt.Errorf("unsupported operating system: %s", runtime.GOOS)
	}

	if err != nil {
		return nil, fmt.Errorf("failed to create interface: %w", err)
	}

	// Configure interface
	if err := iface.SetMTU(m.config.MTU); err != nil {
		return nil, fmt.Errorf("failed to set MTU: %w", err)
	}

	// Parse IP address and netmask
	ip, ipNet, err := parseIPNet(m.config.Address)
	if err != nil {
		return nil, fmt.Errorf("failed to parse IP address: %w", err)
	}

	if err := iface.SetIPAddress(ip, ipNet.Mask); err != nil {
		return nil, fmt.Errorf("failed to set IP address: %w", err)
	}

	if err := iface.Up(); err != nil {
		return nil, fmt.Errorf("failed to bring interface up: %w", err)
	}

	m.logger.Info("Network interface created and configured",
		zap.String("name", iface.Name()),
		zap.String("address", m.config.Address),
		zap.Int("mtu", m.config.MTU),
	)

	return iface, nil
}
