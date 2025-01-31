package adapter

import (
"fmt"
"net"
"os"
"os/exec"
"syscall"
"unsafe"
)

const (
TUNSETIFF = 0x400454ca
IFF_TUN   = 0x0001
IFF_NO_PI = 0x1000
IFF_UP    = 0x1
)

type ifReq struct {
Name  [0x10]byte
Flags uint16
pad   [0x28 - 0x10 - 2]byte
}

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
fd, err := os.OpenFile("/dev/net/tun", os.O_RDWR, 0)
if err != nil {
return fmt.Errorf("failed to open TUN device: %w", err)
}

var req ifReq
copy(req.Name[:], i.name)
req.Flags = IFF_TUN | IFF_NO_PI
_, _, errno := syscall.Syscall(syscall.SYS_IOCTL, fd.Fd(), uintptr(TUNSETIFF), uintptr(unsafe.Pointer(&req)))
if errno != 0 {
fd.Close()
return fmt.Errorf("failed to create TUN interface: %v", errno)
}

i.fd = fd

// Set interface flags
cmd := exec.Command("ip", "link", "set", "dev", i.name, "up")
if err := cmd.Run(); err != nil {
return fmt.Errorf("failed to set interface up: %w", err)
}

// Disable multicast and enable point-to-point mode
cmd = exec.Command("ip", "link", "set", "dev", i.name, "multicast", "off")
if err := cmd.Run(); err != nil {
return fmt.Errorf("failed to disable multicast: %w", err)
}

// Enable ICMP forwarding
cmd = exec.Command("sysctl", "-w", fmt.Sprintf("net.ipv4.conf.%s.forwarding=1", i.name))
if err := cmd.Run(); err != nil {
return fmt.Errorf("failed to enable forwarding: %w", err)
}

// Disable reverse path filtering
cmd = exec.Command("sysctl", "-w", fmt.Sprintf("net.ipv4.conf.%s.rp_filter=0", i.name))
if err := cmd.Run(); err != nil {
return fmt.Errorf("failed to disable rp_filter: %w", err)
}

// Accept local packets
cmd = exec.Command("sysctl", "-w", fmt.Sprintf("net.ipv4.conf.%s.accept_local=1", i.name))
if err := cmd.Run(); err != nil {
return fmt.Errorf("failed to enable accept_local: %w", err)
}

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
