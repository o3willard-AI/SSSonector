package adapter

import (
"fmt"
"net"
"os"
"os/exec"
)

type linuxInterface struct {
name    string
address net.IP
netmask net.IPMask
mtu     int
fd      *os.File
}

func newPlatformInterface(name string) (Interface, error) {
return &linuxInterface{
name: name,
mtu:  1500, // Default MTU
}, nil
}

func (i *linuxInterface) Create() error {
cmd := exec.Command("ip", "tuntap", "add", i.name, "mode", "tun")
if err := cmd.Run(); err != nil {
return fmt.Errorf("failed to create tun interface: %w", err)
}

// Open the TUN device file
fd, err := os.OpenFile("/dev/net/tun", os.O_RDWR, 0)
if err != nil {
return fmt.Errorf("failed to open TUN device: %w", err)
}
i.fd = fd

return nil
}

func (i *linuxInterface) Configure(cfg *Config) error {
ip, mask, err := ParseCIDR(cfg.Address)
if err != nil {
return fmt.Errorf("failed to parse address: %w", err)
}

i.address = ip
i.netmask = mask
i.mtu = cfg.MTU

// Set IP address
cmd := exec.Command("ip", "addr", "add", cfg.Address, "dev", i.name)
if err := cmd.Run(); err != nil {
return fmt.Errorf("failed to set interface address: %w", err)
}

// Set MTU
cmd = exec.Command("ip", "link", "set", i.name, "mtu", fmt.Sprintf("%d", cfg.MTU))
if err := cmd.Run(); err != nil {
return fmt.Errorf("failed to set interface MTU: %w", err)
}

return nil
}

func (i *linuxInterface) Up() error {
cmd := exec.Command("ip", "link", "set", i.name, "up")
if err := cmd.Run(); err != nil {
return fmt.Errorf("failed to bring interface up: %w", err)
}
return nil
}

func (i *linuxInterface) Down() error {
cmd := exec.Command("ip", "link", "set", i.name, "down")
if err := cmd.Run(); err != nil {
return fmt.Errorf("failed to bring interface down: %w", err)
}
return nil
}

func (i *linuxInterface) Delete() error {
cmd := exec.Command("ip", "link", "delete", i.name)
if err := cmd.Run(); err != nil {
return fmt.Errorf("failed to delete interface: %w", err)
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
return i.address
}

func (i *linuxInterface) Netmask() net.IPMask {
return i.netmask
}

// Read implements io.Reader
func (i *linuxInterface) Read(p []byte) (n int, err error) {
if i.fd == nil {
return 0, fmt.Errorf("interface not initialized")
}
return i.fd.Read(p)
}

// Write implements io.Writer
func (i *linuxInterface) Write(p []byte) (n int, err error) {
if i.fd == nil {
return 0, fmt.Errorf("interface not initialized")
}
return i.fd.Write(p)
}

// Close implements io.Closer
func (i *linuxInterface) Close() error {
if i.fd != nil {
if err := i.fd.Close(); err != nil {
return fmt.Errorf("failed to close interface file: %w", err)
}
i.fd = nil
}
return nil
}
