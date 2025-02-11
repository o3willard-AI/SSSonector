package adapter

import (
	"fmt"
	"time"
)

// State represents the adapter state
type State int

const (
	// StateUninitialized indicates the adapter is not initialized
	StateUninitialized State = iota
	// StateInitializing indicates the adapter is initializing
	StateInitializing
	// StateReady indicates the adapter is ready
	StateReady
	// StateStopping indicates the adapter is stopping
	StateStopping
	// StateStopped indicates the adapter is stopped
	StateStopped
	// StateError indicates the adapter is in error state
	StateError
)

// Error definitions
var (
	ErrInvalidStateTransition = fmt.Errorf("invalid state transition")
	ErrInterfaceNotReady      = fmt.Errorf("interface not ready")
	ErrCleanupTimeout         = fmt.Errorf("cleanup timeout")
)

// Options represents adapter configuration options
type Options struct {
	Name           string
	MTU            int
	Address        string
	RetryAttempts  int
	RetryDelay     time.Duration
	CleanupTimeout time.Duration
}

// DefaultOptions returns default adapter options
func DefaultOptions() *Options {
	return &Options{
		MTU:            1500,
		RetryAttempts:  3,
		RetryDelay:     time.Second * 5,
		CleanupTimeout: time.Second * 30,
	}
}

// Statistics represents adapter statistics
type Statistics struct {
	BytesReceived  uint64
	BytesSent      uint64
	PacketsDropped uint64
	Errors         uint64
}

// Status represents adapter status
type Status struct {
	State      State
	Statistics Statistics
	LastError  error
}

// AdapterInterface represents a network adapter interface
type AdapterInterface interface {
	// Basic operations
	Read([]byte) (int, error)
	Write([]byte) (int, error)
	Close() error
	GetAddress() string

	// State management
	GetState() State
	GetStatus() Status
	GetStatistics() Statistics

	// Configuration
	Configure(opts *Options) error
	Cleanup() error
}

// FromConfig creates a new adapter from configuration
func FromConfig(cfg interface{}) (AdapterInterface, error) {
	// Extract network configuration
	type networkConfig struct {
		Name      string `yaml:"name"`
		Interface string `yaml:"interface"`
		MTU       int    `yaml:"mtu"`
		Address   string `yaml:"address"`
	}

	type config struct {
		Config struct {
			Network networkConfig `yaml:"network"`
		} `yaml:"config"`
	}

	c, ok := cfg.(*config)
	if !ok {
		return nil, fmt.Errorf("invalid configuration type")
	}

	opts := DefaultOptions()
	opts.Name = c.Config.Network.Name
	opts.MTU = c.Config.Network.MTU
	opts.Address = c.Config.Network.Address

	return newTUNAdapter(opts)
}
