//go:build windows
// +build windows

package adapter

import (
	"fmt"
	"sync/atomic"
)

// TUNAdapter represents a TUN network adapter for Windows
type TUNAdapter struct {
	name       string
	address    string
	state      State
	statistics Statistics
	lastError  error
}

// NewTUNAdapter creates a new TUN adapter
// Note: Windows implementation is currently a stub
func NewTUNAdapter(opts *Options) (AdapterInterface, error) {
	return &TUNAdapter{
		name:      opts.Name,
		address:   opts.Address,
		state:     StateError,
		lastError: fmt.Errorf("TUN adapter not implemented for Windows"),
	}, nil
}

func (t *TUNAdapter) Read(p []byte) (n int, err error) {
	atomic.AddUint64(&t.statistics.Errors, 1)
	return 0, fmt.Errorf("not implemented")
}

func (t *TUNAdapter) Write(p []byte) (n int, err error) {
	atomic.AddUint64(&t.statistics.Errors, 1)
	return 0, fmt.Errorf("not implemented")
}

func (t *TUNAdapter) Close() error {
	t.state = StateStopped
	return nil
}

func (t *TUNAdapter) GetAddress() string {
	return t.address
}

func (t *TUNAdapter) GetState() State {
	return t.state
}

func (t *TUNAdapter) GetStatus() Status {
	return Status{
		State:      t.state,
		Statistics: t.statistics,
		LastError:  t.lastError,
	}
}

func (t *TUNAdapter) GetStatistics() Statistics {
	return t.statistics
}

func (t *TUNAdapter) Configure(opts *Options) error {
	t.name = opts.Name
	t.address = opts.Address
	return nil
}

func (t *TUNAdapter) Cleanup() error {
	return nil
}
