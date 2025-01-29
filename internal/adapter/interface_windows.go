package adapter

import (
	"fmt"
	"net"
	"os/exec"
	"strings"

	"github.com/songgao/water"
	"go.uber.org/zap"
)

type windowsInterface struct {
	name   string
	iface  *water.Interface
	logger *zap.Logger
}

func newWindowsInterface(name string) (Interface, error) {
	config := water.Config{
		DeviceType: water.TUN,
	}
	if name != "" {
		config.Name = name
	}

	iface, err := water.New(config)
	if err != nil {
		return nil, fmt.Errorf("failed to create TUN interface: %w", err)
	}

	return &windowsInterface{
		name:  iface.Name(),
		iface: iface,
	}, nil
}

func (i *windowsInterface) Name() string {
	return i.name
}

func (i *windowsInterface) Read(p []byte) (n int, err error) {
	return i.iface.Read(p)
}

func (i *windowsInterface) Write(p []byte) (n int, err error) {
	return i.iface.Write(p)
}

func (i *windowsInterface) Close() error {
	return i.iface.Close()
}

func (i *windowsInterface) SetMTU(mtu int) error {
	cmd := exec.Command("netsh", "interface", "ipv4", "set", "subinterface", i.name, fmt.Sprintf("mtu=%d", mtu))
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("failed to set MTU: %w (output: %s)", err, output)
	}
	return nil
}

func (i *windowsInterface) SetIPAddress(addr net.IP, mask net.IPMask) error {
	cmd := exec.Command("netsh", "interface", "ipv4", "set", "address", "name="+i.name, "static", addr.String(), net.IP(mask).String())
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("failed to set IP address: %w (output: %s)", err, output)
	}
	return nil
}

func (i *windowsInterface) GetIPAddress() (net.IP, net.IPMask, error) {
	cmd := exec.Command("netsh", "interface", "ipv4", "show", "addresses", i.name)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return nil, nil, fmt.Errorf("failed to get IP address: %w", err)
	}

	lines := strings.Split(string(output), "\n")
	var ip net.IP
	var mask net.IPMask

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "IP Address:") {
			fields := strings.Fields(line)
			if len(fields) >= 3 {
				ip = net.ParseIP(fields[2])
			}
		} else if strings.HasPrefix(line, "Subnet Prefix:") {
			fields := strings.Fields(line)
			if len(fields) >= 3 {
				parts := strings.Split(fields[2], "/")
				if len(parts) == 2 {
					prefixLen := 0
					fmt.Sscanf(parts[1], "%d", &prefixLen)
					mask = net.CIDRMask(prefixLen, 32)
				}
			}
		}
	}

	if ip == nil || mask == nil {
		return nil, nil, fmt.Errorf("no IP address found")
	}

	return ip, mask, nil
}

func (i *windowsInterface) Up() error {
	cmd := exec.Command("netsh", "interface", "set", "interface", i.name, "admin=enabled")
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("failed to bring interface up: %w (output: %s)", err, output)
	}
	return nil
}

func (i *windowsInterface) Down() error {
	cmd := exec.Command("netsh", "interface", "set", "interface", i.name, "admin=disabled")
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("failed to bring interface down: %w (output: %s)", err, output)
	}
	return nil
}
