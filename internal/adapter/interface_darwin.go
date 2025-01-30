//go:build darwin

package adapter

import (
	"fmt"
	"net"
	"os/exec"
	"strings"
)

type darwinInterface struct {
	name    string
	ip      net.IP
	netmask net.IPMask
	mtu     int
}

func newPlatformInterface(name string) (Interface, error) {
	return &darwinInterface{
		name: name,
		mtu:  1500,
	}, nil
}

func (i *darwinInterface) Create() error {
	// Create TUN interface using macOS built-in tools
	cmd := exec.Command("networksetup", "-createnetworkservice", i.name, "Ethernet")
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("failed to create TUN interface: %s: %w", strings.TrimSpace(string(output)), err)
	}
	return nil
}

func (i *darwinInterface) Configure(cfg *Config) error {
	// Parse IP address and netmask
	ip, mask, err := ParseCIDR(cfg.Address)
	if err != nil {
		return err
	}
	i.ip = ip
	i.netmask = mask
	i.mtu = cfg.MTU

	// Set IP address
	cmd := exec.Command("networksetup", "-setmanual", i.name, ip.String(), net.IP(mask).String())
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("failed to set IP address: %s: %w", strings.TrimSpace(string(output)), err)
	}

	// Set MTU
	cmd = exec.Command("ifconfig", i.name, "mtu", fmt.Sprint(i.mtu))
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("failed to set MTU: %s: %w", strings.TrimSpace(string(output)), err)
	}

	return nil
}

func (i *darwinInterface) Up() error {
	cmd := exec.Command("ifconfig", i.name, "up")
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("failed to bring interface up: %s: %w", strings.TrimSpace(string(output)), err)
	}
	return nil
}

func (i *darwinInterface) Down() error {
	cmd := exec.Command("ifconfig", i.name, "down")
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("failed to bring interface down: %s: %w", strings.TrimSpace(string(output)), err)
	}
	return nil
}

func (i *darwinInterface) Delete() error {
	cmd := exec.Command("networksetup", "-removenetworkservice", i.name)
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("failed to delete interface: %s: %w", strings.TrimSpace(string(output)), err)
	}
	return nil
}

func (i *darwinInterface) Name() string {
	return i.name
}

func (i *darwinInterface) MTU() int {
	return i.mtu
}

func (i *darwinInterface) Address() net.IP {
	return i.ip
}

func (i *darwinInterface) Netmask() net.IPMask {
	return i.netmask
}
