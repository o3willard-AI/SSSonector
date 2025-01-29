package adapter

import (
	"fmt"
	"net"
	"os"
	"os/exec"

	"github.com/songgao/water"
	"go.uber.org/zap"
)

type windowsInterface struct {
	logger *zap.Logger
	iface  *water.Interface
	config *NetworkConfig
}

// NewWindowsInterface creates a new Windows network interface
func NewWindowsInterface(logger *zap.Logger, cfg *NetworkConfig) (Interface, error) {
	if err := validateInterfaceConfig(cfg); err != nil {
		return nil, fmt.Errorf("invalid interface config: %w", err)
	}

	iface := &windowsInterface{
		logger: logger,
		config: cfg,
	}

	if err := iface.create(); err != nil {
		return nil, fmt.Errorf("failed to create interface: %w", err)
	}

	return iface, nil
}

// create creates a new TUN interface
func (i *windowsInterface) create() error {
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
func (i *windowsInterface) configure() error {
	// Set interface up and configure IP using netsh
	cmd := exec.Command("netsh", "interface", "ip", "set", "address",
		fmt.Sprintf("name=%s", i.iface.Name()),
		"source=static",
		fmt.Sprintf("addr=%s", i.config.Address),
		"mask=255.255.255.0",
	)
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("failed to set interface address: %s: %w", string(out), err)
	}

	// Set MTU if specified
	if i.config.MTU > 0 {
		cmd = exec.Command("netsh", "interface", "ipv4", "set", "subinterface",
			fmt.Sprintf("\"%s\"", i.iface.Name()),
			fmt.Sprintf("mtu=%d", i.config.MTU),
		)
		if out, err := cmd.CombinedOutput(); err != nil {
			return fmt.Errorf("failed to set interface MTU: %s: %w", string(out), err)
		}
	}

	return nil
}

// Read reads data from the interface
func (i *windowsInterface) Read(p []byte) (n int, err error) {
	return i.iface.Read(p)
}

// Write writes data to the interface
func (i *windowsInterface) Write(p []byte) (n int, err error) {
	return i.iface.Write(p)
}

// Close closes the interface
func (i *windowsInterface) Close() error {
	if i.iface != nil {
		// Remove interface configuration
		cmd := exec.Command("netsh", "interface", "ip", "set", "address",
			fmt.Sprintf("name=%s", i.iface.Name()),
			"source=dhcp",
		)
		if out, err := cmd.CombinedOutput(); err != nil {
			i.logger.Error("Failed to reset interface configuration",
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
func (i *windowsInterface) GetFD() int {
	if i.iface == nil {
		return -1
	}
	return int(i.iface.ReadWriteCloser.(*os.File).Fd())
}

// GetName returns the interface name
func (i *windowsInterface) GetName() string {
	if i.iface == nil {
		return ""
	}
	return i.iface.Name()
}

// GetHardwareAddr returns the interface hardware address
func (i *windowsInterface) GetHardwareAddr() net.HardwareAddr {
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
func (i *windowsInterface) GetFlags() net.Flags {
	if i.iface == nil {
		return 0
	}

	iface, err := net.InterfaceByName(i.iface.Name())
	if err != nil {
		return 0
	}
	return iface.Flags
}
