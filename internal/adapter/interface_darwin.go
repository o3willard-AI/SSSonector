package adapter

import (
"fmt"
"net"
"os"
)

type darwinInterface struct {
name    string
address net.IP
netmask net.IPMask
mtu     int
handle  *os.File
}

func newPlatformInterface(name string) (Interface, error) {
return &darwinInterface{
name: name,
mtu:  1500, // Default MTU
}, nil
}

func (i *darwinInterface) Create() error {
// macOS implementation would use /dev/tun{N} devices
handle, err := os.OpenFile("/dev/tun0", os.O_RDWR, 0)
if err != nil {
return fmt.Errorf("failed to open TUN device: %w", err)
}
i.handle = handle
return nil
}

func (i *darwinInterface) Configure(cfg *Config) error {
ip, mask, err := ParseCIDR(cfg.Address)
if err != nil {
return fmt.Errorf("failed to parse address: %w", err)
}

i.address = ip
i.netmask = mask
i.mtu = cfg.MTU

// macOS implementation would use system calls or ifconfig to configure the interface
return nil
}

func (i *darwinInterface) Up() error {
// macOS implementation would use ifconfig to bring interface up
return nil
}

func (i *darwinInterface) Down() error {
// macOS implementation would use ifconfig to bring interface down
return nil
}

func (i *darwinInterface) Delete() error {
// macOS implementation would clean up the TUN interface
return nil
}

func (i *darwinInterface) Name() string {
return i.name
}

func (i *darwinInterface) MTU() int {
return i.mtu
}

func (i *darwinInterface) Address() net.IP {
return i.address
}

func (i *darwinInterface) Netmask() net.IPMask {
return i.netmask
}

// Read implements io.Reader
func (i *darwinInterface) Read(p []byte) (n int, err error) {
if i.handle == nil {
return 0, fmt.Errorf("interface not initialized")
}
return i.handle.Read(p)
}

// Write implements io.Writer
func (i *darwinInterface) Write(p []byte) (n int, err error) {
if i.handle == nil {
return 0, fmt.Errorf("interface not initialized")
}
return i.handle.Write(p)
}

// Close implements io.Closer
func (i *darwinInterface) Close() error {
if i.handle != nil {
if err := i.handle.Close(); err != nil {
return fmt.Errorf("failed to close interface handle: %w", err)
}
i.handle = nil
}
return nil
}
