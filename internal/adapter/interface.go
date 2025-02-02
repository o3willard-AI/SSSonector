package adapter

import (
	"fmt"
	"runtime"
)

// Interface represents a network interface
type Interface interface {
	Configure(cfg *Config) error
	Read(p []byte) (n int, err error)
	Write(p []byte) (n int, err error)
	Close() error
	GetName() string
	GetMTU() int
	GetAddress() string
	IsUp() bool
	Cleanup() error
}

// Config holds interface configuration
type Config struct {
	Name    string
	Address string
	MTU     int
}

// New creates a new interface based on the platform
func New(name string) (Interface, error) {
	switch runtime.GOOS {
	case "linux":
		return newLinuxInterface(name)
	case "darwin":
		return nil, fmt.Errorf("darwin platform not implemented")
	case "windows":
		return nil, fmt.Errorf("windows platform not implemented")
	default:
		return nil, fmt.Errorf("unsupported platform: %s", runtime.GOOS)
	}
}
