package adapter

import (
"fmt"
"io"
"net"
)

// Interface represents a virtual network interface
type Interface interface {
io.ReadWriteCloser

// Create creates the virtual interface
Create() error

// Configure sets up the interface with the given configuration
Configure(cfg *Config) error

// Up brings the interface up
Up() error

// Down brings the interface down
Down() error

// Delete removes the interface
Delete() error

// Name returns the interface name
Name() string

// MTU returns the interface MTU
MTU() int

// Address returns the interface IP address
Address() net.IP

// Netmask returns the interface netmask
Netmask() net.IPMask
}

// Config represents interface configuration
type Config struct {
Name    string
Address string
MTU     int
}

// Manager handles virtual interface operations
type Manager struct {
iface Interface
}

// NewManager creates a new interface manager
func NewManager(cfg *Config) (*Manager, error) {
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

// ParseCIDR parses a CIDR string into IP and netmask
func ParseCIDR(cidr string) (net.IP, net.IPMask, error) {
ip, ipnet, err := net.ParseCIDR(cidr)
if err != nil {
return nil, nil, fmt.Errorf("invalid CIDR format: %w", err)
}
return ip, ipnet.Mask, nil
}
