package adapter

import (
	"context"
	"fmt"
	"net"
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

// NewTUNAdapter creates a new TUN adapter
func NewTUNAdapter(opts *Options) (AdapterInterface, error) {
	if opts == nil {
		opts = DefaultOptions()
	}

	logger, err := zap.NewProduction()
	if err != nil {
		return nil, fmt.Errorf("failed to create logger: %w", err)
	}

	adapter := &tunAdapter{
		opts:      opts,
		logger:    logger,
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
			// Ignore "no such device" errors during cleanup
			if !os.IsNotExist(err) {
				done <- fmt.Errorf("failed to delete interface: %w", err)
				return
			}
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
	i.logger.Info(fmt.Sprintf("State: %s", state.String()))
}

// setError sets the last error
func (i *tunAdapter) setError(err error) {
	i.stateMu.Lock()
	defer i.stateMu.Unlock()
	i.lastError = err
	if err != nil {
		i.logger.Error("Adapter error", zap.Error(err))
	}
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

	// Set interface up and configure address
	if err := configureInterface(i.opts.Name, i.opts.Address, i.opts.MTU); err != nil {
		file.Close()
		return fmt.Errorf("failed to configure interface: %w", err)
	}

	return nil
}

// configureInterface sets up the network interface
func configureInterface(name, address string, mtu int) error {
	// Create socket
	sock, err := syscall.Socket(syscall.AF_INET, syscall.SOCK_DGRAM, 0)
	if err != nil {
		return fmt.Errorf("failed to create socket: %w", err)
	}
	defer syscall.Close(sock)

	// Create interface request
	var ifr [ifReqSize]byte
	copy(ifr[:], name)

	// Set interface up
	*(*uint16)(unsafe.Pointer(&ifr[16])) = syscall.IFF_UP | syscall.IFF_RUNNING
	_, _, errno := syscall.Syscall(syscall.SYS_IOCTL, uintptr(sock), syscall.SIOCSIFFLAGS, uintptr(unsafe.Pointer(&ifr[0])))
	if errno != 0 {
		return fmt.Errorf("failed to set interface up: %w", errno)
	}

	// Set MTU
	*(*int32)(unsafe.Pointer(&ifr[16])) = int32(mtu)
	_, _, errno = syscall.Syscall(syscall.SYS_IOCTL, uintptr(sock), syscall.SIOCSIFMTU, uintptr(unsafe.Pointer(&ifr[0])))
	if errno != 0 {
		return fmt.Errorf("failed to set interface MTU: %w", errno)
	}

	// Parse IP address and netmask
	ip, ipNet, err := net.ParseCIDR(address)
	if err != nil {
		return fmt.Errorf("failed to parse address: %w", err)
	}

	// Set IP address
	var sockaddr syscall.RawSockaddrInet4
	sockaddr.Family = syscall.AF_INET
	copy(sockaddr.Addr[:], ip.To4())

	*(*syscall.RawSockaddrInet4)(unsafe.Pointer(&ifr[16])) = sockaddr
	_, _, errno = syscall.Syscall(syscall.SYS_IOCTL, uintptr(sock), syscall.SIOCSIFADDR, uintptr(unsafe.Pointer(&ifr[0])))
	if errno != 0 {
		return fmt.Errorf("failed to set interface address: %w", errno)
	}

	// Set netmask
	copy(sockaddr.Addr[:], ipNet.Mask)
	*(*syscall.RawSockaddrInet4)(unsafe.Pointer(&ifr[16])) = sockaddr
	_, _, errno = syscall.Syscall(syscall.SYS_IOCTL, uintptr(sock), syscall.SIOCSIFNETMASK, uintptr(unsafe.Pointer(&ifr[0])))
	if errno != 0 {
		return fmt.Errorf("failed to set interface netmask: %w", errno)
	}

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

	// Get current interface flags
	_, _, errno := syscall.Syscall(syscall.SYS_IOCTL, uintptr(sock), syscall.SIOCGIFFLAGS, uintptr(unsafe.Pointer(&ifr[0])))
	if errno != 0 {
		if errno == syscall.ENODEV {
			// Interface doesn't exist, which is fine during cleanup
			return nil
		}
		return fmt.Errorf("failed to get interface flags: %w", errno)
	}

	// Clear IFF_UP flag to bring interface down
	flags := *(*uint16)(unsafe.Pointer(&ifr[16]))
	flags &^= syscall.IFF_UP
	*(*uint16)(unsafe.Pointer(&ifr[16])) = flags

	// Set new interface flags
	_, _, errno = syscall.Syscall(syscall.SYS_IOCTL, uintptr(sock), syscall.SIOCSIFFLAGS, uintptr(unsafe.Pointer(&ifr[0])))
	if errno != 0 {
		return fmt.Errorf("failed to set interface flags: %w", errno)
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
