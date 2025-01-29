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

// InterfaceType represents the type of network interface
type InterfaceType int

const (
	// TUN interface type
	TUN InterfaceType = iota
	// TAP interface type
	TAP
)

// InterfaceConfig holds configuration for network interfaces
type InterfaceConfig struct {
	Type      InterfaceType
	Name      string
	MTU       int
	IPAddress net.IP
	Netmask   net.IPMask
}
