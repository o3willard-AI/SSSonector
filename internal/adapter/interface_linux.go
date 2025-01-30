package adapter

import (
	"fmt"
	"net"
	"os/exec"
	"strings"
	"syscall"
	"unsafe"
)

const (
	IFF_TUN   = 0x0001
	IFF_NO_PI = 0x1000
	TUNSETIFF = 0x400454ca
)

type linuxInterface struct {
	name    string
	fd      int
	ip      net.IP
	netmask net.IPMask
	mtu     int
}

func newPlatformInterface(name string) (Interface, error) {
	return &linuxInterface{
		name: name,
		mtu:  1500,
	}, nil
}

func (i *linuxInterface) Create() error {
	// Open TUN device
	fd, err := syscall.Open("/dev/net/tun", syscall.O_RDWR, 0)
	if err != nil {
		return fmt.Errorf("failed to open /dev/net/tun: %w", err)
	}

	// Create interface request
	var ifr struct {
		name  [16]byte
		flags uint16
		_     [22]byte
	}
	copy(ifr.name[:], i.name)
	ifr.flags = IFF_TUN | IFF_NO_PI

	// Configure TUN interface
	_, _, errno := syscall.Syscall(syscall.SYS_IOCTL, uintptr(fd), uintptr(TUNSETIFF), uintptr(unsafe.Pointer(&ifr)))
	if errno != 0 {
		syscall.Close(fd)
		return fmt.Errorf("failed to configure TUN interface: %w", errno)
	}

	i.fd = fd
	return nil
}

func (i *linuxInterface) Configure(cfg *Config) error {
	// Parse IP address and netmask
	ip, mask, err := ParseCIDR(cfg.Address)
	if err != nil {
		return err
	}
	i.ip = ip
	i.netmask = mask
	i.mtu = cfg.MTU

	// Set IP address
	cmd := exec.Command("ip", "addr", "add", cfg.Address, "dev", i.name)
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("failed to set IP address: %s: %w", strings.TrimSpace(string(output)), err)
	}

	// Set MTU
	cmd = exec.Command("ip", "link", "set", "mtu", fmt.Sprint(i.mtu), "dev", i.name)
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("failed to set MTU: %s: %w", strings.TrimSpace(string(output)), err)
	}

	return nil
}

func (i *linuxInterface) Up() error {
	cmd := exec.Command("ip", "link", "set", "dev", i.name, "up")
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("failed to bring interface up: %s: %w", strings.TrimSpace(string(output)), err)
	}
	return nil
}

func (i *linuxInterface) Down() error {
	cmd := exec.Command("ip", "link", "set", "dev", i.name, "down")
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("failed to bring interface down: %s: %w", strings.TrimSpace(string(output)), err)
	}
	return nil
}

func (i *linuxInterface) Delete() error {
	if i.fd != 0 {
		if err := syscall.Close(i.fd); err != nil {
			return fmt.Errorf("failed to close interface: %w", err)
		}
		i.fd = 0
	}

	cmd := exec.Command("ip", "link", "delete", "dev", i.name)
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("failed to delete interface: %s: %w", strings.TrimSpace(string(output)), err)
	}
	return nil
}

func (i *linuxInterface) Name() string {
	return i.name
}

func (i *linuxInterface) MTU() int {
	return i.mtu
}

func (i *linuxInterface) Address() net.IP {
	return i.ip
}

func (i *linuxInterface) Netmask() net.IPMask {
	return i.netmask
}
