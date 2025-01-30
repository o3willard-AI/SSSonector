//go:build windows

package adapter

import (
	"fmt"
	"net"
	"os/exec"
	"strings"
)

type windowsInterface struct {
	name    string
	ip      net.IP
	netmask net.IPMask
	mtu     int
}

func newPlatformInterface(name string) (Interface, error) {
	return &windowsInterface{
		name: name,
		mtu:  1500,
	}, nil
}

func (i *windowsInterface) Create() error {
	// Create TAP adapter using Windows built-in tools
	cmd := exec.Command("netsh", "interface", "ip", "add", "interface", i.name, "type=tap")
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("failed to create TAP interface: %s: %w", strings.TrimSpace(string(output)), err)
	}
	return nil
}

func (i *windowsInterface) Configure(cfg *Config) error {
	// Parse IP address and netmask
	ip, mask, err := ParseCIDR(cfg.Address)
	if err != nil {
		return err
	}
	i.ip = ip
	i.netmask = mask
	i.mtu = cfg.MTU

	// Set IP address
	cmd := exec.Command("netsh", "interface", "ip", "set", "address", i.name, "static", ip.String(), net.IP(mask).String())
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("failed to set IP address: %s: %w", strings.TrimSpace(string(output)), err)
	}

	// Set MTU
	cmd = exec.Command("netsh", "interface", "ipv4", "set", "subinterface", i.name, fmt.Sprintf("mtu=%d", i.mtu))
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("failed to set MTU: %s: %w", strings.TrimSpace(string(output)), err)
	}

	return nil
}

func (i *windowsInterface) Up() error {
	cmd := exec.Command("netsh", "interface", "set", "interface", i.name, "admin=enabled")
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("failed to bring interface up: %s: %w", strings.TrimSpace(string(output)), err)
	}
	return nil
}

func (i *windowsInterface) Down() error {
	cmd := exec.Command("netsh", "interface", "set", "interface", i.name, "admin=disabled")
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("failed to bring interface down: %s: %w", strings.TrimSpace(string(output)), err)
	}
	return nil
}

func (i *windowsInterface) Delete() error {
	cmd := exec.Command("netsh", "interface", "delete", "interface", i.name)
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("failed to delete interface: %s: %w", strings.TrimSpace(string(output)), err)
	}
	return nil
}

func (i *windowsInterface) Name() string {
	return i.name
}

func (i *windowsInterface) MTU() int {
	return i.mtu
}

func (i *windowsInterface) Address() net.IP {
	return i.ip
}

func (i *windowsInterface) Netmask() net.IPMask {
	return i.netmask
}
