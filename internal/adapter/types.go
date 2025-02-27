package adapter

import (
	"fmt"
	"time"

	"github.com/o3willard-AI/SSSonector/internal/config/types"
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

// String returns the string representation of the state
func (s State) String() string {
	switch s {
	case StateUninitialized:
		return "Uninitialized"
	case StateInitializing:
		return "Initializing"
	case StateReady:
		return "Ready"
	case StateStopping:
		return "Stopping"
	case StateStopped:
		return "Stopped"
	case StateError:
		return "Error"
	default:
		return "Unknown"
	}
}

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
	appConfig, ok := cfg.(*types.AppConfig)
	if !ok {
		return nil, fmt.Errorf("invalid configuration type")
	}

	if appConfig.Config == nil || appConfig.Config.Network == nil {
		return nil, fmt.Errorf("missing network configuration")
	}

	opts := DefaultOptions()
	opts.Name = appConfig.Config.Network.Interface
	opts.MTU = appConfig.Config.Network.MTU
	opts.Address = appConfig.Config.Network.Address

	if appConfig.Adapter != nil {
		opts.RetryAttempts = appConfig.Adapter.RetryAttempts
		opts.RetryDelay = appConfig.Adapter.RetryDelay.Duration
		opts.CleanupTimeout = appConfig.Adapter.CleanupTimeout.Duration
	}

	return NewTUNAdapter(opts)
}
