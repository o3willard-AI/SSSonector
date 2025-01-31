package adapter

import (
"fmt"
"net"
"os"
"sync"
"unsafe"

"golang.org/x/sys/windows"
)

const (
FILE_DEVICE_UNKNOWN = 0x00000022
METHOD_BUFFERED    = 0
FILE_ANY_ACCESS    = 0

TAP_IOCTL_GET_MAC               = (FILE_DEVICE_UNKNOWN << 16) | (FILE_ANY_ACCESS << 14) | (1 << 2) | METHOD_BUFFERED
TAP_IOCTL_GET_VERSION           = (FILE_DEVICE_UNKNOWN << 16) | (FILE_ANY_ACCESS << 14) | (2 << 2) | METHOD_BUFFERED
TAP_IOCTL_SET_MEDIA_STATUS     = (FILE_DEVICE_UNKNOWN << 16) | (FILE_ANY_ACCESS << 14) | (6 << 2) | METHOD_BUFFERED
TAP_IOCTL_CONFIG_POINT_TO_POINT = (FILE_DEVICE_UNKNOWN << 16) | (FILE_ANY_ACCESS << 14) | (7 << 2) | METHOD_BUFFERED
TAP_IOCTL_CONFIG_DHCP_MASQ      = (FILE_DEVICE_UNKNOWN << 16) | (FILE_ANY_ACCESS << 14) | (8 << 2) | METHOD_BUFFERED
TAP_IOCTL_GET_LOG_LINE          = (FILE_DEVICE_UNKNOWN << 16) | (FILE_ANY_ACCESS << 14) | (9 << 2) | METHOD_BUFFERED
)

type windowsInterface struct {
name    string
address net.IP
netmask net.IPMask
mtu     int
handle  windows.Handle
fd      *os.File
mutex   sync.Mutex
}

func newPlatformInterface(name string) (Interface, error) {
return &windowsInterface{
name: name,
mtu:  1500,
}, nil
}

func (i *windowsInterface) Create() error {
i.mutex.Lock()
defer i.mutex.Unlock()

// Open TAP device
devicePath := fmt.Sprintf(`\\.\Global\%s.tap`, i.name)
pathPtr, err := windows.UTF16PtrFromString(devicePath)
if err != nil {
return fmt.Errorf("failed to convert device path: %w", err)
}

handle, err := windows.CreateFile(
pathPtr,
windows.GENERIC_READ|windows.GENERIC_WRITE,
windows.FILE_SHARE_READ|windows.FILE_SHARE_WRITE,
nil,
windows.OPEN_EXISTING,
windows.FILE_ATTRIBUTE_SYSTEM|windows.FILE_FLAG_OVERLAPPED,
0,
)
if err != nil {
return fmt.Errorf("failed to open TAP device: %w", err)
}

i.handle = handle
i.fd = os.NewFile(uintptr(handle), devicePath)

// Set media status to connected
var statusWord uint32 = 1
var bytesReturned uint32
err = windows.DeviceIoControl(
i.handle,
TAP_IOCTL_SET_MEDIA_STATUS,
(*byte)(unsafe.Pointer(&statusWord)),
uint32(unsafe.Sizeof(statusWord)),
(*byte)(unsafe.Pointer(&statusWord)),
uint32(unsafe.Sizeof(statusWord)),
&bytesReturned,
nil,
)
if err != nil {
i.Close()
return fmt.Errorf("failed to set media status: %w", err)
}

return nil
}

func (i *windowsInterface) Configure(cfg *Config) error {
i.mutex.Lock()
defer i.mutex.Unlock()

ip, mask, err := ParseCIDR(cfg.Address)
if err != nil {
return fmt.Errorf("failed to parse address: %w", err)
}

i.address = ip
i.netmask = mask
i.mtu = cfg.MTU

// Configure point-to-point mode
addresses := make([]byte, 12)
copy(addresses[0:4], ip.To4())
copy(addresses[4:8], mask)
copy(addresses[8:12], ip.To4())

var bytesReturned uint32
err = windows.DeviceIoControl(
i.handle,
TAP_IOCTL_CONFIG_POINT_TO_POINT,
&addresses[0],
uint32(len(addresses)),
&addresses[0],
uint32(len(addresses)),
&bytesReturned,
nil,
)
if err != nil {
return fmt.Errorf("failed to configure point-to-point: %w", err)
}

return nil
}

func (i *windowsInterface) Up() error {
i.mutex.Lock()
defer i.mutex.Unlock()

var statusWord uint32 = 1
var bytesReturned uint32
err := windows.DeviceIoControl(
i.handle,
TAP_IOCTL_SET_MEDIA_STATUS,
(*byte)(unsafe.Pointer(&statusWord)),
uint32(unsafe.Sizeof(statusWord)),
(*byte)(unsafe.Pointer(&statusWord)),
uint32(unsafe.Sizeof(statusWord)),
&bytesReturned,
nil,
)
if err != nil {
return fmt.Errorf("failed to bring interface up: %w", err)
}

return nil
}

func (i *windowsInterface) Down() error {
i.mutex.Lock()
defer i.mutex.Unlock()

var statusWord uint32 = 0
var bytesReturned uint32
err := windows.DeviceIoControl(
i.handle,
TAP_IOCTL_SET_MEDIA_STATUS,
(*byte)(unsafe.Pointer(&statusWord)),
uint32(unsafe.Sizeof(statusWord)),
(*byte)(unsafe.Pointer(&statusWord)),
uint32(unsafe.Sizeof(statusWord)),
&bytesReturned,
nil,
)
if err != nil {
return fmt.Errorf("failed to bring interface down: %w", err)
}

return nil
}

func (i *windowsInterface) Delete() error {
return i.Close()
}

func (i *windowsInterface) Name() string {
return i.name
}

func (i *windowsInterface) MTU() int {
return i.mtu
}

func (i *windowsInterface) Address() net.IP {
return i.address
}

func (i *windowsInterface) Netmask() net.IPMask {
return i.netmask
}

func (i *windowsInterface) Read(p []byte) (n int, err error) {
i.mutex.Lock()
defer i.mutex.Unlock()

if i.fd == nil {
return 0, fmt.Errorf("interface not initialized")
}
return i.fd.Read(p)
}

func (i *windowsInterface) Write(p []byte) (n int, err error) {
i.mutex.Lock()
defer i.mutex.Unlock()

if i.fd == nil {
return 0, fmt.Errorf("interface not initialized")
}
return i.fd.Write(p)
}

func (i *windowsInterface) Close() error {
i.mutex.Lock()
defer i.mutex.Unlock()

if i.fd != nil {
if err := i.fd.Close(); err != nil {
return fmt.Errorf("failed to close interface file: %w", err)
}
i.fd = nil
}

if i.handle != 0 {
if err := windows.CloseHandle(i.handle); err != nil {
return fmt.Errorf("failed to close handle: %w", err)
}
i.handle = 0
}

return nil
}
