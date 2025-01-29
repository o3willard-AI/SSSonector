package adapter

import "net"

// Interface represents a network interface
type Interface interface {
	// Read reads data from the interface
	Read(p []byte) (n int, err error)

	// Write writes data to the interface
	Write(p []byte) (n int, err error)

	// Close closes the interface
	Close() error

	// GetFD returns the file descriptor for the interface
	GetFD() int

	// GetName returns the interface name
	GetName() string

	// GetHardwareAddr returns the interface hardware address
	GetHardwareAddr() net.HardwareAddr

	// GetFlags returns the interface flags
	GetFlags() net.Flags
}
