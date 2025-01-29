package adapter

import (
	"fmt"
	"net"

	"SSSonector/internal/config"

	"go.uber.org/zap"
)

// InterfaceType represents the type of virtual network interface
type InterfaceType string

const (
	// TUN represents a network layer (L3) interface
	TUN InterfaceType = "tun"
	// TAP represents a data link layer (L2) interface
	TAP InterfaceType = "tap"
)

// Interface represents a virtual network interface
type Interface interface {
	// Name returns the interface name
	Name() string

	// Type returns the interface type (tun/tap)
	Type() InterfaceType

	// IP returns the interface IP address
	IP() net.IP

	// Read reads a packet from the interface
	Read(packet []byte) (int, error)

	// Write writes a packet to the interface
	Write(packet []byte) (int, error)

	// Close closes the interface
	Close() error

	// SetMTU sets the interface MTU
	SetMTU(mtu int) error

	// SetFlags sets the interface flags (up/down, etc.)
	SetFlags(flags int) error
}

// Manager handles virtual interface creation and management
type Manager interface {
	// Create creates a new virtual interface
	Create(cfg *config.NetworkConfig) (Interface, error)

	// Remove removes a virtual interface
	Remove(name string) error

	// Get retrieves an existing virtual interface
	Get(name string) (Interface, error)
}

// NewManager creates a new platform-specific interface manager
func NewManager(logger *zap.Logger) (Manager, error) {
	// Platform-specific implementation will be selected at build time
	return newManager(logger)
}

// interfaceBase provides common functionality for virtual interfaces
type interfaceBase struct {
	name string
	typ  InterfaceType
	ip   net.IP
}

func (i *interfaceBase) Name() string {
	return i.name
}

func (i *interfaceBase) Type() InterfaceType {
	return i.typ
}

func (i *interfaceBase) IP() net.IP {
	return i.ip
}

// validateNetworkConfig validates the network configuration for interface creation
func validateNetworkConfig(cfg *config.NetworkConfig) error {
	if cfg.Interface == "" {
		return fmt.Errorf("interface name is required")
	}

	if cfg.Address == "" {
		return fmt.Errorf("interface address is required")
	}

	ip := net.ParseIP(cfg.Address)
	if ip == nil {
		return fmt.Errorf("invalid IP address: %s", cfg.Address)
	}

	if cfg.MTU < 576 || cfg.MTU > 9000 {
		return fmt.Errorf("invalid MTU: %d (must be between 576 and 9000)", cfg.MTU)
	}

	return nil
}
