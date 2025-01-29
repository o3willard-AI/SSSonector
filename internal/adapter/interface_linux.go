//go:build linux
// +build linux

package adapter

import (
	"fmt"
	"net"
	"os/exec"
	"strings"
	"syscall"

	"SSSonector/internal/config"

	"github.com/songgao/water"
	"go.uber.org/zap"
	"golang.org/x/sys/unix"
)

// linuxInterface implements the Interface for Linux systems
type linuxInterface struct {
	interfaceBase
	iface      *water.Interface
	logger     *zap.Logger
	persistent bool
}

// linuxManager implements the Manager interface for Linux systems
type linuxManager struct {
	logger *zap.Logger
}

func newManager(logger *zap.Logger) (Manager, error) {
	return &linuxManager{logger: logger}, nil
}

// Create implements Manager.Create for Linux
func (m *linuxManager) Create(cfg *config.NetworkConfig) (Interface, error) {
	if err := validateNetworkConfig(cfg); err != nil {
		return nil, fmt.Errorf("invalid network configuration: %w", err)
	}

	m.logger.Debug("Creating TUN device",
		zap.String("name", cfg.Interface),
		zap.String("address", cfg.Address))

	// Create TUN device
	m.logger.Debug("Opening /dev/net/tun")
	ifConfig := water.Config{
		DeviceType: water.TUN, // We only support TUN devices
		PlatformSpecificParams: water.PlatformSpecificParams{
			Name:        cfg.Interface,
			Persist:     false, // Non-persistent by default
			MultiQueue:  false,
			Permissions: nil, // Use default permissions
		},
	}

	m.logger.Debug("Creating interface with config",
		zap.String("type", "tun"),
		zap.String("name", cfg.Interface))
	iface, err := water.New(ifConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create interface: %w", err)
	}

	// Parse IP address
	ip := net.ParseIP(cfg.Address)
	if ip == nil {
		iface.Close()
		return nil, fmt.Errorf("invalid IP address: %s", cfg.Address)
	}

	// Create interface instance
	li := &linuxInterface{
		interfaceBase: interfaceBase{
			name: iface.Name(),
			typ:  TUN,
			ip:   ip,
		},
		iface:      iface,
		logger:     m.logger,
		persistent: false,
	}

	// Configure the interface
	m.logger.Debug("Configuring interface",
		zap.String("name", li.name),
		zap.String("ip", li.ip.String()))
	if err := li.configure(); err != nil {
		iface.Close()
		return nil, fmt.Errorf("failed to configure interface: %w", err)
	}
	m.logger.Debug("Interface configured successfully")

	m.logger.Info("Created network interface",
		zap.String("name", li.name),
		zap.String("type", string(li.typ)),
		zap.String("ip", ip.String()),
	)

	return li, nil
}

// Remove implements Manager.Remove for Linux
func (m *linuxManager) Remove(name string) error {
	if name == "" {
		return fmt.Errorf("interface name is required")
	}

	cmd := exec.Command("ip", "link", "delete", name)
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("failed to remove interface: %s (%w)", string(output), err)
	}

	m.logger.Info("Removed network interface", zap.String("name", name))
	return nil
}

// Get implements Manager.Get for Linux
func (m *linuxManager) Get(name string) (Interface, error) {
	if name == "" {
		return nil, fmt.Errorf("interface name is required")
	}

	// Check if interface exists
	cmd := exec.Command("ip", "link", "show", name)
	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("interface not found: %s", name)
	}

	// Get interface type
	cmd = exec.Command("ip", "link", "show", name)
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to get interface info: %w", err)
	}

	var ifType InterfaceType
	if strings.Contains(string(output), "tun") {
		ifType = TUN
	} else if strings.Contains(string(output), "tap") {
		ifType = TAP
	} else {
		return nil, fmt.Errorf("interface is not TUN/TAP: %s", name)
	}

	// Get interface IP
	cmd = exec.Command("ip", "addr", "show", name)
	output, err = cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to get interface address: %w", err)
	}

	// Parse IP address from output
	var ip net.IP
	lines := strings.Split(string(output), "\n")
	for _, line := range lines {
		if strings.Contains(line, "inet ") {
			fields := strings.Fields(line)
			if len(fields) >= 2 {
				addr := strings.Split(fields[1], "/")[0]
				ip = net.ParseIP(addr)
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
	}
	if ifType == TAP {
		ifConfig.DeviceType = water.TAP
	}

	iface, err := water.New(ifConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to open interface: %w", err)
	}

	return &linuxInterface{
		interfaceBase: interfaceBase{
			name: name,
			typ:  ifType,
			ip:   ip,
		},
		iface:  iface,
		logger: m.logger,
	}, nil
}

// Read implements Interface.Read
func (i *linuxInterface) Read(packet []byte) (int, error) {
	return i.iface.Read(packet)
}

// Write implements Interface.Write
func (i *linuxInterface) Write(packet []byte) (int, error) {
	return i.iface.Write(packet)
}

// Close implements Interface.Close
func (i *linuxInterface) Close() error {
	if !i.persistent {
		if err := (&linuxManager{logger: i.logger}).Remove(i.name); err != nil {
			i.logger.Error("Failed to remove interface during close",
				zap.String("name", i.name),
				zap.Error(err),
			)
		}
	}
	return i.iface.Close()
}

// SetMTU implements Interface.SetMTU
func (i *linuxInterface) SetMTU(mtu int) error {
	cmd := exec.Command("ip", "link", "set", i.name, "mtu", fmt.Sprintf("%d", mtu))
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("failed to set MTU: %s (%w)", string(output), err)
	}
	return nil
}

// SetFlags implements Interface.SetFlags
func (i *linuxInterface) SetFlags(flags int) error {
	// Get the file descriptor using syscall.Socket
	fd, err := syscall.Socket(syscall.AF_INET, syscall.SOCK_DGRAM, 0)
	if err != nil {
		return fmt.Errorf("failed to create socket: %w", err)
	}
	defer syscall.Close(fd)
	_, _, errno := syscall.Syscall(syscall.SYS_IOCTL, uintptr(fd),
		uintptr(syscall.SIOCSIFFLAGS), uintptr(flags))
	if errno != 0 {
		return fmt.Errorf("failed to set interface flags: %w", errno)
	}
	return nil
}

// configure sets up the interface IP address and brings it up
func (i *linuxInterface) configure() error {
	i.logger.Debug("Setting IP address",
		zap.String("ip", i.ip.String()),
		zap.String("interface", i.name))

	// Set IP address
	cmd := exec.Command("ip", "addr", "add", fmt.Sprintf("%s/24", i.ip.String()), "dev", i.name)
	if output, err := cmd.CombinedOutput(); err != nil {
		i.logger.Error("Failed to set IP address",
			zap.String("output", string(output)),
			zap.Error(err))
		return fmt.Errorf("failed to set IP address: %s (%w)", string(output), err)
	}

	i.logger.Debug("Bringing interface up")
	// Bring interface up
	cmd = exec.Command("ip", "link", "set", i.name, "up")
	if output, err := cmd.CombinedOutput(); err != nil {
		i.logger.Error("Failed to bring interface up",
			zap.String("output", string(output)),
			zap.Error(err))
		return fmt.Errorf("failed to bring interface up: %s (%w)", string(output), err)
	}

	i.logger.Debug("Setting interface flags")
	// Set interface flags
	if err := i.SetFlags(unix.IFF_UP | unix.IFF_RUNNING); err != nil {
		i.logger.Error("Failed to set interface flags", zap.Error(err))
		return fmt.Errorf("failed to set interface flags: %w", err)
	}

	i.logger.Debug("Interface configuration completed successfully")
	return nil
}
