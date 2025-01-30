package adapter

import "net"

// Interface represents a virtual network interface
type Interface interface {
	// Create creates the virtual interface
	Create() error

	// Configure sets up the interface with the given configuration
	Configure(cfg *Config) error

	// Up brings the interface up
	Up() error

	// Down brings the interface down
	Down() error

	// Delete removes the interface
	Delete() error

	// Name returns the interface name
	Name() string

	// MTU returns the interface MTU
	MTU() int

	// Address returns the interface IP address
	Address() net.IP

	// Netmask returns the interface netmask
	Netmask() net.IPMask
}

// Config represents interface configuration
type Config struct {
	Name    string
	Address string
	MTU     int
}

// ParseCIDR parses a CIDR string into IP and netmask
func ParseCIDR(cidr string) (net.IP, net.IPMask, error) {
	ip, ipnet, err := net.ParseCIDR(cidr)
	if err != nil {
		return nil, nil, err
	}
	return ip, ipnet.Mask, nil
}
