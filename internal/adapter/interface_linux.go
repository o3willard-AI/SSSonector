package adapter

import (
	"fmt"
	"net"
	"os"
	"os/exec"

	"github.com/songgao/water"
	"go.uber.org/zap"
)

type linuxInterface struct {
	logger *zap.Logger
	iface  *water.Interface
	config *NetworkConfig
}

// NewLinuxInterface creates a new Linux network interface
func NewLinuxInterface(logger *zap.Logger, cfg *NetworkConfig) (Interface, error) {
	if err := validateInterfaceConfig(cfg); err != nil {
		return nil, fmt.Errorf("invalid interface config: %w", err)
	}

	iface := &linuxInterface{
		logger: logger,
		config: cfg,
	}

	if err := iface.create(); err != nil {
		return nil, fmt.Errorf("failed to create interface: %w", err)
	}

	return iface, nil
}

// create creates a new TUN interface
func (i *linuxInterface) create() error {
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
func (i *linuxInterface) configure() error {
	// Set interface up
	cmd := exec.Command("ip", "link", "set", i.iface.Name(), "up")
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("failed to set interface up: %s: %w", string(out), err)
	}

	// Set IP address
	cmd = exec.Command("ip", "addr", "add", i.config.Address+"/24", "dev", i.iface.Name())
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("failed to set interface address: %s: %w", string(out), err)
	}

	// Set MTU if specified
	if i.config.MTU > 0 {
		cmd = exec.Command("ip", "link", "set", i.iface.Name(), "mtu", fmt.Sprintf("%d", i.config.MTU))
		if out, err := cmd.CombinedOutput(); err != nil {
			return fmt.Errorf("failed to set interface MTU: %s: %w", string(out), err)
		}
	}

	return nil
}

// Read reads data from the interface
func (i *linuxInterface) Read(p []byte) (n int, err error) {
	return i.iface.Read(p)
}

// Write writes data to the interface
func (i *linuxInterface) Write(p []byte) (n int, err error) {
	return i.iface.Write(p)
}

// Close closes the interface
func (i *linuxInterface) Close() error {
	if i.iface != nil {
		// Remove interface configuration
		cmd := exec.Command("ip", "link", "set", i.iface.Name(), "down")
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
func (i *linuxInterface) GetFD() int {
	if i.iface == nil {
		return -1
	}
	return int(i.iface.ReadWriteCloser.(*os.File).Fd())
}

// GetName returns the interface name
func (i *linuxInterface) GetName() string {
	if i.iface == nil {
		return ""
	}
	return i.iface.Name()
}

// GetHardwareAddr returns the interface hardware address
func (i *linuxInterface) GetHardwareAddr() net.HardwareAddr {
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
func (i *linuxInterface) GetFlags() net.Flags {
	if i.iface == nil {
		return 0
	}

	iface, err := net.InterfaceByName(i.iface.Name())
	if err != nil {
		return 0
	}
	return iface.Flags
}
