package adapter

import (
	"context"
	"fmt"
	"os"
	"sync"
	"sync/atomic"
	"syscall"
	"unsafe"

	"go.uber.org/zap"
)

const (
	tunDevice = "/dev/net/tun"
	ifReqSize = 40
	TUNSETIFF = 0x400454ca
)

// tunAdapter represents a TUN interface adapter
type tunAdapter struct {
	opts      *Options
	logger    *zap.Logger
	file      *os.File
	state     State
	lastError error
	stats     Statistics
	stateMu   sync.RWMutex
	closeChan chan struct{}
	closeOnce sync.Once
}

// newTUNAdapter creates a new TUN adapter
func newTUNAdapter(opts *Options) (AdapterInterface, error) {
	if opts == nil {
		opts = DefaultOptions()
	}

	adapter := &tunAdapter{
		opts:      opts,
		state:     StateUninitialized,
		closeChan: make(chan struct{}),
	}

	if err := adapter.initialize(); err != nil {
		return nil, err
	}

	return adapter, nil
}

// Configure configures the adapter with new options
func (i *tunAdapter) Configure(opts *Options) error {
	if i.GetState() != StateStopped {
		return fmt.Errorf("adapter must be stopped before reconfiguring")
	}

	i.opts = opts
	return i.initialize()
}

// Read implements io.Reader
func (i *tunAdapter) Read(p []byte) (n int, err error) {
	if i.GetState() != StateReady {
		return 0, ErrInterfaceNotReady
	}

	n, err = i.file.Read(p)
	if err != nil {
		atomic.AddUint64(&i.stats.Errors, 1)
		return n, err
	}

	atomic.AddUint64(&i.stats.BytesReceived, uint64(n))
	return n, nil
}

// Write implements io.Writer
func (i *tunAdapter) Write(p []byte) (n int, err error) {
	if i.GetState() != StateReady {
		return 0, ErrInterfaceNotReady
	}

	n, err = i.file.Write(p)
	if err != nil {
		atomic.AddUint64(&i.stats.Errors, 1)
		return n, err
	}

	atomic.AddUint64(&i.stats.BytesSent, uint64(n))
	return n, nil
}

// Close implements io.Closer
func (i *tunAdapter) Close() error {
	var err error
	i.closeOnce.Do(func() {
		i.setState(StateStopping)

		// Close file descriptor
		if i.file != nil {
			err = i.file.Close()
			if err != nil {
				i.setError(fmt.Errorf("failed to close file: %w", err))
			}
		}

		close(i.closeChan)
		i.setState(StateStopped)
	})
	return err
}

// Cleanup performs cleanup operations
func (i *tunAdapter) Cleanup() error {
	// Set cleanup state
	i.setState(StateStopping)

	// Create cleanup timeout context
	ctx, cancel := context.WithTimeout(context.Background(), i.opts.CleanupTimeout)
	defer cancel()

	// Create done channel
	done := make(chan error, 1)

	// Run cleanup in goroutine
	go func() {
		// Close adapter
		err := i.Close()
		if err != nil {
			done <- fmt.Errorf("failed to close adapter: %w", err)
			return
		}

		// Delete interface
		if err := deleteInterface(i.opts.Name); err != nil {
			done <- fmt.Errorf("failed to delete interface: %w", err)
			return
		}

		done <- nil
	}()

	// Wait for cleanup or timeout
	select {
	case err := <-done:
		if err != nil {
			i.setError(err)
			i.setState(StateError)
			return err
		}
		i.setState(StateStopped)
		return nil
	case <-ctx.Done():
		i.setError(ErrCleanupTimeout)
		i.setState(StateError)
		return ErrCleanupTimeout
	}
}

// GetAddress returns the interface address
func (i *tunAdapter) GetAddress() string {
	return i.opts.Address
}

// GetState returns the current state
func (i *tunAdapter) GetState() State {
	i.stateMu.RLock()
	defer i.stateMu.RUnlock()
	return i.state
}

// GetStatus returns the current status
func (i *tunAdapter) GetStatus() Status {
	return Status{
		State:      i.GetState(),
		Statistics: i.GetStatistics(),
		LastError:  i.lastError,
	}
}

// GetStatistics returns the current statistics
func (i *tunAdapter) GetStatistics() Statistics {
	return Statistics{
		BytesReceived:  atomic.LoadUint64(&i.stats.BytesReceived),
		BytesSent:      atomic.LoadUint64(&i.stats.BytesSent),
		PacketsDropped: atomic.LoadUint64(&i.stats.PacketsDropped),
		Errors:         atomic.LoadUint64(&i.stats.Errors),
	}
}

// setState sets the adapter state
func (i *tunAdapter) setState(state State) {
	i.stateMu.Lock()
	defer i.stateMu.Unlock()
	i.state = state
}

// setError sets the last error
func (i *tunAdapter) setError(err error) {
	i.stateMu.Lock()
	defer i.stateMu.Unlock()
	i.lastError = err
}

// createInterface creates the TUN interface
func (i *tunAdapter) createInterface() error {
	// Open TUN device
	file, err := os.OpenFile(tunDevice, os.O_RDWR, 0)
	if err != nil {
		return fmt.Errorf("failed to open TUN device: %w", err)
	}

	// Create interface request
	var ifr [ifReqSize]byte
	copy(ifr[:], i.opts.Name)
	*(*uint16)(unsafe.Pointer(&ifr[16])) = syscall.IFF_TUN | syscall.IFF_NO_PI

	// Set interface flags
	_, _, errno := syscall.Syscall(syscall.SYS_IOCTL, file.Fd(), uintptr(TUNSETIFF), uintptr(unsafe.Pointer(&ifr[0])))
	if errno != 0 {
		file.Close()
		return fmt.Errorf("failed to set interface flags: %w", errno)
	}

	i.file = file
	return nil
}

// deleteInterface deletes the network interface
func deleteInterface(name string) error {
	// Create socket
	sock, err := syscall.Socket(syscall.AF_INET, syscall.SOCK_DGRAM, 0)
	if err != nil {
		return fmt.Errorf("failed to create socket: %w", err)
	}
	defer syscall.Close(sock)

	// Create interface request
	var ifr [ifReqSize]byte
	copy(ifr[:], name)

	// Delete interface
	_, _, errno := syscall.Syscall(syscall.SYS_IOCTL, uintptr(sock), syscall.SIOCDIFADDR, uintptr(unsafe.Pointer(&ifr[0])))
	if errno != 0 {
		return fmt.Errorf("failed to delete interface: %w", errno)
	}

	return nil
}

// initialize initializes the adapter
func (i *tunAdapter) initialize() error {
	i.setState(StateInitializing)

	// Create interface
	if err := i.createInterface(); err != nil {
		i.setError(err)
		i.setState(StateError)
		return err
	}

	i.setState(StateReady)
	return nil
}
