package adapter

import (
	"fmt"
	"net"
	"os"
	"os/exec"

	"github.com/songgao/water"
	"go.uber.org/zap"
)

type darwinInterface struct {
	logger *zap.Logger
	iface  *water.Interface
	config *NetworkConfig
}

// NetworkConfig holds network interface configuration
type NetworkConfig struct {
	Interface string
	Address   string
	MTU       int
}

// validateInterfaceConfig validates the interface configuration
func validateInterfaceConfig(cfg *NetworkConfig) error {
	if cfg.Address == "" {
		return fmt.Errorf("interface address is required")
	}
	if cfg.Interface == "" {
		return fmt.Errorf("interface name is required")
	}
	return nil
}

// NewDarwinInterface creates a new macOS network interface
func NewDarwinInterface(logger *zap.Logger, cfg *NetworkConfig) (Interface, error) {
	if err := validateInterfaceConfig(cfg); err != nil {
		return nil, fmt.Errorf("invalid interface config: %w", err)
	}

	iface := &darwinInterface{
		logger: logger,
		config: cfg,
	}

	if err := iface.create(); err != nil {
		return nil, fmt.Errorf("failed to create interface: %w", err)
	}

	return iface, nil
}

// create creates a new TUN interface
func (i *darwinInterface) create() error {
	config := water.Config{
		DeviceType: water.TUN,
	}

	iface, err := water.New(config)
	if err != nil {
		return fmt.Errorf("failed to create TUN interface: %w", err)
	}
	i.iface = iface

	// Configure interface
	if err := i.configure(); err != nil {
		i.iface.Close()
		return fmt.Errorf("failed to configure interface: %w", err)
	}

	return nil
}

// configure configures the network interface
func (i *darwinInterface) configure() error {
	// Set interface up
	cmd := exec.Command("ifconfig", i.iface.Name(), "up")
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("failed to set interface up: %s: %w", string(out), err)
	}

	// Set IP address
	cmd = exec.Command("ifconfig", i.iface.Name(), "inet", i.config.Address, i.config.Address)
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("failed to set interface address: %s: %w", string(out), err)
	}

	// Set MTU if specified
	if i.config.MTU > 0 {
		cmd = exec.Command("ifconfig", i.iface.Name(), "mtu", fmt.Sprintf("%d", i.config.MTU))
		if out, err := cmd.CombinedOutput(); err != nil {
			return fmt.Errorf("failed to set interface MTU: %s: %w", string(out), err)
		}
	}

	return nil
}

// Read reads data from the interface
func (i *darwinInterface) Read(p []byte) (n int, err error) {
	return i.iface.Read(p)
}

// Write writes data to the interface
func (i *darwinInterface) Write(p []byte) (n int, err error) {
	return i.iface.Write(p)
}

// Close closes the interface
func (i *darwinInterface) Close() error {
	if i.iface != nil {
		// Remove interface configuration
		cmd := exec.Command("ifconfig", i.iface.Name(), "down")
		if out, err := cmd.CombinedOutput(); err != nil {
			i.logger.Error("Failed to bring interface down",
				zap.String("interface", i.iface.Name()),
				zap.String("output", string(out)),
				zap.Error(err),
			)
		}

		return i.iface.Close()
	}
	return nil
}

// GetFD returns the file descriptor for the interface
func (i *darwinInterface) GetFD() int {
	if i.iface == nil {
		return -1
	}
	return int(i.iface.ReadWriteCloser.(*os.File).Fd())
}

// GetName returns the interface name
func (i *darwinInterface) GetName() string {
	if i.iface == nil {
		return ""
	}
	return i.iface.Name()
}

// GetHardwareAddr returns the interface hardware address
func (i *darwinInterface) GetHardwareAddr() net.HardwareAddr {
	if i.iface == nil {
		return nil
	}

	iface, err := net.InterfaceByName(i.iface.Name())
	if err != nil {
		return nil
	}
	return iface.HardwareAddr
}

// GetFlags returns the interface flags
func (i *darwinInterface) GetFlags() net.Flags {
	if i.iface == nil {
		return 0
	}

	iface, err := net.InterfaceByName(i.iface.Name())
	if err != nil {
		return 0
	}
	return iface.Flags
}
