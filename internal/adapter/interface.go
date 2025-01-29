package adapter

import (
	"io"
	"net"
)

// Interface represents a network interface
type Interface interface {
	io.ReadWriteCloser
	Name() string
	SetMTU(mtu int) error
	SetIPAddress(addr net.IP, mask net.IPMask) error
	GetIPAddress() (net.IP, net.IPMask, error)
	Up() error
	Down() error
}

// NewLinuxInterface creates a new Linux TUN interface
func NewLinuxInterface(name string) (Interface, error) {
	return newLinuxInterface(name)
}

// NewDarwinInterface creates a new macOS TUN interface
func NewDarwinInterface(name string) (Interface, error) {
	return newDarwinInterface(name)
}

// NewWindowsInterface creates a new Windows TUN interface
func NewWindowsInterface(name string) (Interface, error) {
	return newWindowsInterface(name)
}
