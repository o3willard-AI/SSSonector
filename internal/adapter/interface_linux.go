//go:build linux
// +build linux

package adapter

import (
	"fmt"
	"net"
	"os"
	"os/exec"
	"sync"
	"sync/atomic"
	"syscall"
	"time"
	"unsafe"
)

const (
	TUNSETIFF = 0x400454ca
	IFF_TUN   = 0x0001
	IFF_NO_PI = 0x1000
	IFF_UP    = 0x1
)

type linuxInterface struct {
	name    string
	file    *os.File
	address string
	mtu     int
	isUp    bool
	state   atomic.Value // Holds InterfaceState
	stateMu sync.Mutex   // Protects state transitions
	opts    *Options
}

// transitionState attempts to transition the interface from one state to another
func (i *linuxInterface) transitionState(from, to InterfaceState) bool {
	i.stateMu.Lock()
	defer i.stateMu.Unlock()

	if current := i.state.Load().(InterfaceState); current != from {
		return false
	}

	i.state.Store(to)
	return true
}

// setState sets the interface state directly
func (i *linuxInterface) setState(state InterfaceState) {
	i.stateMu.Lock()
	defer i.stateMu.Unlock()
	i.state.Store(state)
}

// getState returns the current interface state
func (i *linuxInterface) getState() InterfaceState {
	return i.state.Load().(InterfaceState)
}

func New(name string, opts *Options) (Interface, error) {
	if opts == nil {
		opts = DefaultOptions()
	}
	return newLinuxInterface(name, opts)
}

func newLinuxInterface(name string, opts *Options) (Interface, error) {
	iface := &linuxInterface{
		name: name,
		opts: opts,
	}
	iface.state.Store(StateUninitialized)

	if err := iface.initialize(); err != nil {
		return nil, err
	}

	return iface, nil
}

func (i *linuxInterface) initialize() error {
	if !i.transitionState(StateUninitialized, StateInitializing) {
		return ErrInvalidStateTransition
	}

	file, err := os.OpenFile("/dev/net/tun", os.O_RDWR, 0)
	if err != nil {
		i.setState(StateError)
		return fmt.Errorf("failed to open /dev/net/tun: %w", err)
	}

	ifreq, err := createIfreq(i.name)
	if err != nil {
		file.Close()
		i.setState(StateError)
		return err
	}

	_, _, errno := syscall.Syscall(syscall.SYS_IOCTL, file.Fd(), uintptr(TUNSETIFF), uintptr(unsafe.Pointer(&ifreq[0])))
	if errno != 0 {
		file.Close()
		i.setState(StateError)
		return fmt.Errorf("failed to create TUN interface: %v", errno)
	}

	// Wait for interface to be ready with configurable retry
	ready := false
	for attempt := 0; attempt < i.opts.RetryAttempts; attempt++ {
		if _, err := os.Stat(fmt.Sprintf("/sys/class/net/%s", i.name)); err == nil {
			ready = true
			break
		}
		time.Sleep(time.Duration(i.opts.RetryDelay) * time.Millisecond)
	}

	if !ready {
		file.Close()
		i.setState(StateError)
		return fmt.Errorf("interface failed to become ready after %d attempts", i.opts.RetryAttempts)
	}

	i.file = file
	i.mtu = 1500 // Default MTU
	i.setState(StateReady)
	return nil
}

func createIfreq(name string) ([]byte, error) {
	var ifreq [40]byte
	copy(ifreq[:16], []byte(name))
	*(*uint16)(unsafe.Pointer(&ifreq[16])) = IFF_TUN | IFF_NO_PI | IFF_UP
	return ifreq[:], nil
}

func (i *linuxInterface) Configure(cfg *Config) error {
	if state := i.getState(); state != StateReady {
		return fmt.Errorf("%w: current state %s", ErrInterfaceNotReady, state)
	}

	// Parse IP address and network to validate format
	if _, _, err := net.ParseCIDR(cfg.Address); err != nil {
		i.setState(StateError)
		return fmt.Errorf("invalid address format: %w", err)
	}

	// Configure with retries
	configured := false
	for attempt := 0; attempt < i.opts.RetryAttempts; attempt++ {
		if err := i.applyConfiguration(cfg); err == nil {
			configured = true
			break
		}
		time.Sleep(time.Duration(i.opts.RetryDelay) * time.Millisecond)
	}

	if !configured {
		i.setState(StateError)
		return ErrConfigurationFailed
	}

	i.address = cfg.Address
	i.mtu = cfg.MTU
	i.isUp = true

	return nil
}

func (i *linuxInterface) applyConfiguration(cfg *Config) error {
	// Set IP address
	if err := exec.Command("ip", "addr", "add", cfg.Address, "dev", i.name).Run(); err != nil {
		return fmt.Errorf("failed to set IP address: %w", err)
	}

	// Set MTU
	if err := exec.Command("ip", "link", "set", "mtu", fmt.Sprintf("%d", cfg.MTU), "dev", i.name).Run(); err != nil {
		return fmt.Errorf("failed to set MTU: %w", err)
	}

	// Bring interface up
	if err := exec.Command("ip", "link", "set", "up", "dev", i.name).Run(); err != nil {
		return fmt.Errorf("failed to bring interface up: %w", err)
	}

	// Verify configuration
	if i.opts.ValidateState {
		if err := exec.Command("ip", "addr", "show", "dev", i.name).Run(); err != nil {
			return fmt.Errorf("failed to validate interface configuration: %w", err)
		}
	}

	return nil
}

func (i *linuxInterface) Read(p []byte) (int, error) {
	return i.file.Read(p)
}

func (i *linuxInterface) Write(p []byte) (int, error) {
	return i.file.Write(p)
}

func (i *linuxInterface) Close() error {
	if i.file != nil {
		return i.file.Close()
	}
	return nil
}

func (i *linuxInterface) GetName() string {
	return i.name
}

func (i *linuxInterface) GetMTU() int {
	return i.mtu
}

func (i *linuxInterface) GetAddress() string {
	return i.address
}

func (i *linuxInterface) IsUp() bool {
	return i.isUp
}

func (i *linuxInterface) Cleanup() error {
	if !i.transitionState(StateReady, StateStopping) {
		return ErrInvalidStateTransition
	}

	// Create cleanup channel with timeout
	done := make(chan error, 1)
	go func() {
		done <- i.performCleanup()
	}()

	// Wait for cleanup with timeout
	select {
	case err := <-done:
		if err != nil {
			i.setState(StateError)
			return err
		}
		i.setState(StateStopped)
		return nil
	case <-time.After(time.Duration(i.opts.CleanupTimeout) * time.Millisecond):
		i.setState(StateError)
		return ErrCleanupTimeout
	}
}

func (i *linuxInterface) performCleanup() error {
	if i.isUp {
		// Bring interface down
		if err := exec.Command("ip", "link", "set", "down", "dev", i.name).Run(); err != nil {
			return fmt.Errorf("failed to bring interface down: %w", err)
		}

		// Remove IP address with retries
		for attempt := 0; attempt < i.opts.RetryAttempts; attempt++ {
			if err := exec.Command("ip", "addr", "del", i.address, "dev", i.name).Run(); err == nil {
				break
			}
			if attempt == i.opts.RetryAttempts-1 {
				return fmt.Errorf("failed to remove IP address after %d attempts", i.opts.RetryAttempts)
			}
			time.Sleep(time.Duration(i.opts.RetryDelay) * time.Millisecond)
		}

		i.isUp = false
	}

	return i.Close()
}
