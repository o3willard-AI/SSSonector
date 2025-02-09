package adapter

import "fmt"

// InterfaceState represents the possible states of a network interface
type InterfaceState int

const (
	StateUninitialized InterfaceState = iota
	StateInitializing
	StateReady
	StateStopping
	StateStopped
	StateError
)

// String provides a human-readable representation of interface states
func (s InterfaceState) String() string {
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
		return fmt.Sprintf("Unknown(%d)", s)
	}
}

// Interface errors
var (
	ErrInvalidStateTransition = fmt.Errorf("invalid state transition")
	ErrInterfaceNotReady      = fmt.Errorf("interface not in ready state")
	ErrCleanupTimeout         = fmt.Errorf("interface cleanup timed out")
	ErrConfigurationFailed    = fmt.Errorf("interface configuration failed")
)

// Statistics holds interface performance metrics
type Statistics struct {
	BytesReceived  uint64
	BytesSent      uint64
	PacketsDropped uint64
	Errors         uint64
}

// Status represents the current interface status
type Status struct {
	State      InterfaceState
	IsUp       bool
	MTU        int
	Address    string
	LastError  error
	Statistics *Statistics
}

// Options contains configurable interface parameters
type Options struct {
	RetryAttempts  int
	RetryDelay     int // milliseconds
	CleanupTimeout int // milliseconds
	ValidateState  bool
}

// DefaultOptions provides sensible defaults for interface options
func DefaultOptions() *Options {
	return &Options{
		RetryAttempts:  3,
		RetryDelay:     100,
		CleanupTimeout: 5000,
		ValidateState:  true,
	}
}
