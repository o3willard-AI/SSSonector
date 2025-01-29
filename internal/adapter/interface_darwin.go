//go:build darwin
// +build darwin

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
	"golang.org/x/sys/unix"
)

// darwinInterface implements the Interface for macOS systems
type darwinInterface struct {
	interfaceBase
	iface      *water.Interface
	logger     *zap.Logger
	persistent bool
}

// darwinManager implements the Manager interface for macOS systems
type darwinManager struct {
	logger *zap.Logger
}

func newManager(logger *zap.Logger) (Manager, error) {
	return &darwinManager{logger: logger}, nil
}

// Create implements Manager.Create for macOS
func (m *darwinManager) Create(cfg *config.InterfaceConfig) (Interface, error) {
	if err := validateInterfaceConfig(cfg); err != nil {
		return nil, fmt.Errorf("invalid interface configuration: %w", err)
	}

	// Create TUN/TAP device
	ifConfig := water.Config{
		DeviceType: water.TUN,
	}
	if cfg.Type == string(TAP) {
		return nil, fmt.Errorf("TAP interfaces are not supported on macOS")
	}
	if cfg.Name != "" {
		ifConfig.Name = cfg.Name
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
	di := &darwinInterface{
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
	if err := di.configure(); err != nil {
		iface.Close()
		return nil, fmt.Errorf("failed to configure interface: %w", err)
	}

	m.logger.Info("Created network interface",
		zap.String("name", di.name),
		zap.String("type", string(di.typ)),
		zap.String("ip", ip.String()),
	)

	return di, nil
}

// Remove implements Manager.Remove for macOS
func (m *darwinManager) Remove(name string) error {
	if name == "" {
		return fmt.Errorf("interface name is required")
	}

	// On macOS, we use ifconfig to remove the interface
	cmd := exec.Command("ifconfig", name, "destroy")
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("failed to remove interface: %s (%w)", string(output), err)
	}

	m.logger.Info("Removed network interface", zap.String("name", name))
	return nil
}

// Get implements Manager.Get for macOS
func (m *darwinManager) Get(name string) (Interface, error) {
	if name == "" {
		return nil, fmt.Errorf("interface name is required")
	}

	// Check if interface exists
	cmd := exec.Command("ifconfig", name)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("interface not found: %s", name)
	}

	// Parse interface type and IP from ifconfig output
	var ip net.IP
	lines := strings.Split(string(output), "\n")
	for _, line := range lines {
		if strings.Contains(line, "inet ") {
			fields := strings.Fields(line)
			if len(fields) >= 2 {
				ip = net.ParseIP(fields[1])
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
		Name:       name,
	}

	iface, err := water.New(ifConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to open interface: %w", err)
	}

	return &darwinInterface{
		interfaceBase: interfaceBase{
			name: name,
			typ:  TUN, // macOS only supports TUN
			ip:   ip,
		},
		iface:  iface,
		logger: m.logger,
	}, nil
}

// Read implements Interface.Read
func (i *darwinInterface) Read(packet []byte) (int, error) {
	return i.iface.Read(packet)
}

// Write implements Interface.Write
func (i *darwinInterface) Write(packet []byte) (int, error) {
	return i.iface.Write(packet)
}

// Close implements Interface.Close
func (i *darwinInterface) Close() error {
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
func (i *darwinInterface) SetMTU(mtu int) error {
	cmd := exec.Command("ifconfig", i.name, "mtu", fmt.Sprintf("%d", mtu))
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("failed to set MTU: %s (%w)", string(output), err)
	}
	return nil
}

// SetFlags implements Interface.SetFlags
func (i *darwinInterface) SetFlags(flags int) error {
	fd := i.iface.FD()
	_, _, errno := syscall.Syscall(syscall.SYS_IOCTL, uintptr(fd),
		uintptr(syscall.SIOCSIFFLAGS), uintptr(flags))
	if errno != 0 {
		return fmt.Errorf("failed to set interface flags: %w", errno)
	}
	return nil
}

// configure sets up the interface IP address and brings it up
func (i *darwinInterface) configure() error {
	// Set IP address
	cmd := exec.Command("ifconfig", i.name, "inet", i.ip.String(),
		i.ip.String(), "netmask", "255.255.255.0", "up")
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("failed to configure interface: %s (%w)", string(output), err)
	}

	// Set interface flags
	if err := i.SetFlags(unix.IFF_UP | unix.IFF_RUNNING); err != nil {
		return fmt.Errorf("failed to set interface flags: %w", err)
	}

	return nil
}
