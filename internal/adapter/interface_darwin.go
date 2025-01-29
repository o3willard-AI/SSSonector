package adapter

import (
	"fmt"
	"net"
	"os/exec"
	"strings"

	"github.com/songgao/water"
	"go.uber.org/zap"
)

type darwinInterface struct {
	name   string
	iface  *water.Interface
	logger *zap.Logger
}

func newDarwinInterface(name string) (Interface, error) {
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

	return &darwinInterface{
		name:  iface.Name(),
		iface: iface,
	}, nil
}

func (i *darwinInterface) Name() string {
	return i.name
}

func (i *darwinInterface) Read(p []byte) (n int, err error) {
	return i.iface.Read(p)
}

func (i *darwinInterface) Write(p []byte) (n int, err error) {
	return i.iface.Write(p)
}

func (i *darwinInterface) Close() error {
	return i.iface.Close()
}

func (i *darwinInterface) SetMTU(mtu int) error {
	cmd := exec.Command("ifconfig", i.name, "mtu", fmt.Sprintf("%d", mtu))
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("failed to set MTU: %w (output: %s)", err, output)
	}
	return nil
}

func (i *darwinInterface) SetIPAddress(addr net.IP, mask net.IPMask) error {
	ones, _ := mask.Size()
	cidr := fmt.Sprintf("%s/%d", addr.String(), ones)
	cmd := exec.Command("ifconfig", i.name, "inet", addr.String(), addr.String(), "netmask", net.IP(mask).String())
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("failed to set IP address: %w (output: %s)", err, output)
	}
	return nil
}

func (i *darwinInterface) GetIPAddress() (net.IP, net.IPMask, error) {
	cmd := exec.Command("ifconfig", i.name)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return nil, nil, fmt.Errorf("failed to get IP address: %w", err)
	}

	lines := strings.Split(string(output), "\n")
	for _, line := range lines {
		if strings.Contains(line, "inet ") {
			fields := strings.Fields(line)
			if len(fields) < 4 {
				continue
			}
			ip := net.ParseIP(fields[1])
			if ip == nil {
				continue
			}
			mask := parseNetmask(fields[3])
			if mask == nil {
				continue
			}
			return ip, mask, nil
		}
	}
	return nil, nil, fmt.Errorf("no IP address found")
}

func (i *darwinInterface) Up() error {
	cmd := exec.Command("ifconfig", i.name, "up")
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("failed to bring interface up: %w (output: %s)", err, output)
	}
	return nil
}

func (i *darwinInterface) Down() error {
	cmd := exec.Command("ifconfig", i.name, "down")
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("failed to bring interface down: %w (output: %s)", err, output)
	}
	return nil
}

func parseNetmask(s string) net.IPMask {
	// Convert hex string (e.g., 0xffffff00) to IP mask
	s = strings.TrimPrefix(s, "0x")
	if len(s) != 8 {
		return nil
	}
	var mask [4]byte
	for i := 0; i < 4; i++ {
		b, err := parseHexByte(s[i*2 : i*2+2])
		if err != nil {
			return nil
		}
		mask[i] = b
	}
	return net.IPv4Mask(mask[0], mask[1], mask[2], mask[3])
}

func parseHexByte(s string) (byte, error) {
	var b byte
	_, err := fmt.Sscanf(s, "%02x", &b)
	return b, err
}
