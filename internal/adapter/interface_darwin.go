package adapter

import (
"fmt"
"net"
"os"
"os/exec"
"sync"
"syscall"
"unsafe"
)

const (
DARWIN_UTUN_CONTROL_NAME = "com.apple.net.utun_control"
CTLIOCGINFO             = (0x40000000 | 0x80000000) | ((100 & 0x1fff) << 16) | uint32(78) << 8 | 3
)

type darwinInterface struct {
name    string
address net.IP
netmask net.IPMask
mtu     int
fd      *os.File
mutex   sync.Mutex
}

func newPlatformInterface(name string) (Interface, error) {
return &darwinInterface{
name: name,
mtu:  1500,
}, nil
}

func (i *darwinInterface) Create() error {
i.mutex.Lock()
defer i.mutex.Unlock()

// Create UTUN interface
fd, err := syscall.Socket(syscall.AF_SYSTEM, syscall.SOCK_DGRAM, 2)
if err != nil {
return fmt.Errorf("failed to create UTUN socket: %w", err)
}

var ctlInfo = &struct {
ctlID   uint32
ctlName [96]byte
}{}

copy(ctlInfo.ctlName[:], DARWIN_UTUN_CONTROL_NAME)

_, _, errno := syscall.Syscall(
syscall.SYS_IOCTL,
uintptr(fd),
uintptr(CTLIOCGINFO),
uintptr(unsafe.Pointer(ctlInfo)),
)
if errno != 0 {
syscall.Close(fd)
return fmt.Errorf("failed to get UTUN control info: %w", errno)
}

sc, err := syscall.Socket(syscall.AF_SYSTEM, syscall.SOCK_DGRAM, int(ctlInfo.ctlID))
if err != nil {
return fmt.Errorf("failed to create UTUN control socket: %w", err)
}

i.fd = os.NewFile(uintptr(sc), i.name)

// Set non-blocking mode
err = syscall.SetNonblock(sc, true)
if err != nil {
i.Close()
return fmt.Errorf("failed to set non-blocking mode: %w", err)
}

return nil
}

func (i *darwinInterface) Configure(cfg *Config) error {
i.mutex.Lock()
defer i.mutex.Unlock()

ip, mask, err := ParseCIDR(cfg.Address)
if err != nil {
return fmt.Errorf("failed to parse address: %w", err)
}

i.address = ip
i.netmask = mask
i.mtu = cfg.MTU

// Configure interface using ifconfig
cmd := exec.Command("ifconfig", i.name, "inet", cfg.Address, ip.String())
if err := cmd.Run(); err != nil {
return fmt.Errorf("failed to set interface address: %w", err)
}

// Set MTU
cmd = exec.Command("ifconfig", i.name, "mtu", fmt.Sprintf("%d", cfg.MTU))
if err := cmd.Run(); err != nil {
return fmt.Errorf("failed to set interface MTU: %w", err)
}

// Enable interface
cmd = exec.Command("ifconfig", i.name, "up")
if err := cmd.Run(); err != nil {
return fmt.Errorf("failed to bring interface up: %w", err)
}

// Enable IP forwarding
cmd = exec.Command("sysctl", "-w", "net.inet.ip.forwarding=1")
if err := cmd.Run(); err != nil {
return fmt.Errorf("failed to enable IP forwarding: %w", err)
}

return nil
}

func (i *darwinInterface) Up() error {
i.mutex.Lock()
defer i.mutex.Unlock()

cmd := exec.Command("ifconfig", i.name, "up")
if err := cmd.Run(); err != nil {
return fmt.Errorf("failed to bring interface up: %w", err)
}
return nil
}

func (i *darwinInterface) Down() error {
i.mutex.Lock()
defer i.mutex.Unlock()

cmd := exec.Command("ifconfig", i.name, "down")
if err := cmd.Run(); err != nil {
return fmt.Errorf("failed to bring interface down: %w", err)
}
return nil
}

func (i *darwinInterface) Delete() error {
i.mutex.Lock()
defer i.mutex.Unlock()

cmd := exec.Command("ifconfig", i.name, "destroy")
if err := cmd.Run(); err != nil {
return fmt.Errorf("failed to delete interface: %w", err)
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
return i.address
}

func (i *darwinInterface) Netmask() net.IPMask {
return i.netmask
}

func (i *darwinInterface) Read(p []byte) (n int, err error) {
i.mutex.Lock()
defer i.mutex.Unlock()

if i.fd == nil {
return 0, fmt.Errorf("interface not initialized")
}

// Read packet
n, err = i.fd.Read(p)
if err != nil {
return 0, err
}

// Skip 4 bytes of header on macOS
if n > 4 {
copy(p, p[4:])
n -= 4
}

return n, nil
}

func (i *darwinInterface) Write(p []byte) (n int, err error) {
i.mutex.Lock()
defer i.mutex.Unlock()

if i.fd == nil {
return 0, fmt.Errorf("interface not initialized")
}

// Prepend 4 bytes of header required by macOS
header := make([]byte, 4)
header[0] = 0x00
header[1] = 0x00
header[2] = 0x00
header[3] = 0x02 // AF_INET

packet := append(header, p...)
return i.fd.Write(packet)
}

func (i *darwinInterface) Close() error {
i.mutex.Lock()
defer i.mutex.Unlock()

if i.fd != nil {
if err := i.fd.Close(); err != nil {
return fmt.Errorf("failed to close interface file: %w", err)
}
i.fd = nil
}

return nil
}
