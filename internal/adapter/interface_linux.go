package adapter

import (
	"fmt"
	"net"
	"os/exec"
	"strings"

	"github.com/songgao/water"
	"go.uber.org/zap"
)

type linuxInterface struct {
	name   string
	iface  *water.Interface
	logger *zap.Logger
}

func newLinuxInterface(name string) (Interface, error) {
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

	return &linuxInterface{
		name:  iface.Name(),
		iface: iface,
	}, nil
}

func (i *linuxInterface) Name() string {
	return i.name
}

func (i *linuxInterface) Read(p []byte) (n int, err error) {
	return i.iface.Read(p)
}

func (i *linuxInterface) Write(p []byte) (n int, err error) {
	return i.iface.Write(p)
}

func (i *linuxInterface) Close() error {
	return i.iface.Close()
}

func (i *linuxInterface) SetMTU(mtu int) error {
	cmd := exec.Command("ip", "link", "set", i.name, "mtu", fmt.Sprintf("%d", mtu))
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("failed to set MTU: %w (output: %s)", err, output)
	}
	return nil
}

func (i *linuxInterface) SetIPAddress(addr net.IP, mask net.IPMask) error {
	ones, _ := mask.Size()
	cidr := fmt.Sprintf("%s/%d", addr.String(), ones)
	cmd := exec.Command("ip", "addr", "add", cidr, "dev", i.name)
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("failed to set IP address: %w (output: %s)", err, output)
	}
	return nil
}

func (i *linuxInterface) GetIPAddress() (net.IP, net.IPMask, error) {
	cmd := exec.Command("ip", "addr", "show", i.name)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return nil, nil, fmt.Errorf("failed to get IP address: %w", err)
	}

	lines := strings.Split(string(output), "\n")
	for _, line := range lines {
		if strings.Contains(line, "inet ") {
			fields := strings.Fields(line)
			if len(fields) < 2 {
				continue
			}
			ipNet, err := parseIPNet(fields[1])
			if err != nil {
				continue
			}
			return ipNet.IP, ipNet.Mask, nil
		}
	}
	return nil, nil, fmt.Errorf("no IP address found")
}

func (i *linuxInterface) Up() error {
	cmd := exec.Command("ip", "link", "set", i.name, "up")
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("failed to bring interface up: %w (output: %s)", err, output)
	}
	return nil
}

func (i *linuxInterface) Down() error {
	cmd := exec.Command("ip", "link", "set", i.name, "down")
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("failed to bring interface down: %w (output: %s)", err, output)
	}
	return nil
}

func parseIPNet(s string) (*net.IPNet, error) {
	if !strings.Contains(s, "/") {
		s = s + "/32"
	}
	ip, ipNet, err := net.ParseCIDR(s)
	if err != nil {
		return nil, err
	}
	ipNet.IP = ip
	return ipNet, nil
}
