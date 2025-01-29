//go:build windows
// +build windows

package adapter

import (
	"fmt"
	"net"
	"os/exec"
	"strings"
	"syscall"

	"github.com/songgao/water"
	"SSSonector/internal/config"
	"go.uber.org/zap"
	"golang.org/x/sys/windows"
)

// windowsInterface implements the Interface for Windows systems
type windowsInterface struct {
	interfaceBase
	iface      *water.Interface
	logger     *zap.Logger
	persistent bool
}

// windowsManager implements the Manager interface for Windows systems
type windowsManager struct {
	logger *zap.Logger
}

func newManager(logger *zap.Logger) (Manager, error) {
	return &windowsManager{logger: logger}, nil
}

// Create implements Manager.Create for Windows
func (m *windowsManager) Create(cfg *config.InterfaceConfig) (Interface, error) {
	if err := validateInterfaceConfig(cfg); err != nil {
		return nil, fmt.Errorf("invalid interface configuration: %w", err)
	}

	// Create TUN/TAP device
	ifConfig := water.Config{
		DeviceType: water.TUN,
		PlatformSpecificParams: water.PlatformSpecificParams{
			ComponentID: "tap0901", // TAP-Windows Adapter V9
			Network:     cfg.IP + "/24",
		},
	}
	if cfg.Type == string(TAP) {
		ifConfig.DeviceType = water.TAP
	}
	if cfg.Name != "" {
		ifConfig.PlatformSpecificParams.InterfaceName = cfg.Name
	}

	iface, err := water.New(ifConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create interface: %w", err)
	}

	// Parse IP address
	ip := net.ParseIP(cfg.IP)
	if ip == nil {
		iface.Close()
		return nil, fmt.Errorf("invalid IP address: %s", cfg.IP)
	}

	// Create interface instance
	wi := &windowsInterface{
		interfaceBase: interfaceBase{
			name: iface.Name(),
			typ:  InterfaceType(cfg.Type),
			ip:   ip,
		},
		iface:      iface,
		logger:     m.logger,
		persistent: cfg.Persistent,
	}

	// Configure the interface
	if err := wi.configure(); err != nil {
		iface.Close()
		return nil, fmt.Errorf("failed to configure interface: %w", err)
	}

	m.logger.Info("Created network interface",
		zap.String("name", wi.name),
		zap.String("type", string(wi.typ)),
		zap.String("ip", ip.String()),
	)

	return wi, nil
}

// Remove implements Manager.Remove for Windows
func (m *windowsManager) Remove(name string) error {
	if name == "" {
		return fmt.Errorf("interface name is required")
	}

	// On Windows, we use devcon to remove the device
	cmd := exec.Command("devcon", "remove", fmt.Sprintf("@ROOT\\NET\\%s", name))
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("failed to remove interface: %s (%w)", string(output), err)
	}

	m.logger.Info("Removed network interface", zap.String("name", name))
	return nil
}

// Get implements Manager.Get for Windows
func (m *windowsManager) Get(name string) (Interface, error) {
	if name == "" {
		return nil, fmt.Errorf("interface name is required")
	}

	// Check if interface exists
	cmd := exec.Command("netsh", "interface", "show", "interface", name)
	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("interface not found: %s", name)
	}

	// Get interface IP
	cmd = exec.Command("netsh", "interface", "ip", "show", "addresses", name)
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to get interface address: %w", err)
	}

	// Parse IP address from output
	var ip net.IP
	lines := strings.Split(string(output), "\n")
	for _, line := range lines {
		if strings.Contains(line, "IP Address:") {
			fields := strings.Fields(line)
			if len(fields) >= 3 {
				ip = net.ParseIP(fields[2])
				break
			}
		}
	}

	if ip == nil {
		return nil, fmt.Errorf("interface has no IP address: %s", name)
	}

	// Open the existing interface
	ifConfig := water.Config{
		DeviceType: water.TUN,
		PlatformSpecificParams: water.PlatformSpecificParams{
			ComponentID:    "tap0901",
			InterfaceName: name,
			Network:       fmt.Sprintf("%s/24", ip.String()),
		},
	}

	iface, err := water.New(ifConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to open interface: %w", err)
	}

	return &windowsInterface{
		interfaceBase: interfaceBase{
			name: name,
			typ:  TUN, // Windows water package only supports TUN
			ip:   ip,
		},
		iface:  iface,
		logger: m.logger,
	}, nil
}

// Read implements Interface.Read
func (i *windowsInterface) Read(packet []byte) (int, error) {
	return i.iface.Read(packet)
}

// Write implements Interface.Write
func (i *windowsInterface) Write(packet []byte) (int, error) {
	return i.iface.Write(packet)
}

// Close implements Interface.Close
func (i *windowsInterface) Close() error {
	if !i.persistent {
		if err := i.Remove(i.name); err != nil {
			i.logger.Error("Failed to remove interface during close",
				zap.String("name", i.name),
				zap.Error(err),
			)
		}
	}
	return i.iface.Close()
}

// SetMTU implements Interface.SetMTU
func (i *windowsInterface) SetMTU(mtu int) error {
	cmd := exec.Command("netsh", "interface", "ipv4", "set", "subinterface",
		i.name, fmt.Sprintf("mtu=%d", mtu))
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("failed to set MTU: %s (%w)", string(output), err)
	}
	return nil
}

// SetFlags implements Interface.SetFlags
func (i *windowsInterface) SetFlags(flags int) error {
	// Windows doesn't support direct flag manipulation like Linux
	// Instead, we use netsh to enable/disable the interface
	if flags&syscall.IFF_UP != 0 {
		cmd := exec.Command("netsh", "interface", "set", "interface",
			i.name, "admin=enabled")
		if output, err := cmd.CombinedOutput(); err != nil {
			return fmt.Errorf("failed to enable interface: %s (%w)", string(output), err)
		}
	} else {
		cmd := exec.Command("netsh", "interface", "set", "interface",
			i.name, "admin=disabled")
		if output, err := cmd.CombinedOutput(); err != nil {
			return fmt.Errorf("failed to disable interface: %s (%w)", string(output), err)
		}
	}
	return nil
}

// configure sets up the interface IP address and brings it up
func (i *windowsInterface) configure() error {
	// Set IP address
	cmd := exec.Command("netsh", "interface", "ip", "set", "address",
		i.name, "static", i.ip.String(), "255.255.255.0")
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("failed to set IP address: %s (%w)", string(output), err)
	}

	// Enable the interface
	if err := i.SetFlags(syscall.IFF_UP); err != nil {
		return fmt.Errorf("failed to bring interface up: %w", err)
	}

	return nil
}
