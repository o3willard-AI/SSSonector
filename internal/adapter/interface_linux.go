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

	// First, check if interface already exists and remove it
	if _, err := os.Stat(fmt.Sprintf("/sys/class/net/%s", i.name)); err == nil {
		if out, err := exec.Command("sudo", "ip", "tuntap", "del", "dev", i.name, "mode", "tun").CombinedOutput(); err != nil {
			fmt.Printf("Warning: Failed to remove existing interface: %v (output: %s)\n", err, string(out))
		}
		time.Sleep(time.Second) // Wait for interface to be removed
	}

	// Create TUN device
	if out, err := exec.Command("sudo", "ip", "tuntap", "add", "dev", i.name, "mode", "tun").CombinedOutput(); err != nil {
		i.setState(StateError)
		return fmt.Errorf("failed to create TUN device: %w (output: %s)", err, string(out))
	}

	// Set ownership and permissions
	if out, err := exec.Command("sudo", "chown", "root:sssonector", fmt.Sprintf("/sys/class/net/%s", i.name)).CombinedOutput(); err != nil {
		i.setState(StateError)
		return fmt.Errorf("failed to set interface ownership: %w (output: %s)", err, string(out))
	}

	if out, err := exec.Command("sudo", "chmod", "0660", fmt.Sprintf("/sys/class/net/%s", i.name)).CombinedOutput(); err != nil {
		i.setState(StateError)
		return fmt.Errorf("failed to set interface permissions: %w (output: %s)", err, string(out))
	}

	// Wait for interface to be created
	ready := false
	for attempt := 0; attempt < i.opts.RetryAttempts; attempt++ {
		if _, err := os.Stat(fmt.Sprintf("/sys/class/net/%s", i.name)); err == nil {
			ready = true
			break
		}
		time.Sleep(time.Duration(i.opts.RetryDelay) * time.Millisecond)
	}

	if !ready {
		i.setState(StateError)
		return fmt.Errorf("interface failed to appear after creation")
	}

	// Open TUN device
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

	// Ensure interface is down before configuration
	if out, err := exec.Command("sudo", "ip", "link", "set", "dev", i.name, "down").CombinedOutput(); err != nil {
		file.Close()
		i.setState(StateError)
		return fmt.Errorf("failed to bring interface down: %w (output: %s)", err, string(out))
	}

	// Set interface ownership
	if out, err := exec.Command("sudo", "chown", "sssonector:sssonector", fmt.Sprintf("/dev/net/tun")).CombinedOutput(); err != nil {
		file.Close()
		i.setState(StateError)
		return fmt.Errorf("failed to set TUN device ownership: %w (output: %s)", err, string(out))
	}

	i.file = file
	i.mtu = 1500 // Default MTU
	i.setState(StateReady)
	return nil
}

func createIfreq(name string) ([]byte, error) {
	var ifreq [40]byte
	copy(ifreq[:16], []byte(name))
	*(*uint16)(unsafe.Pointer(&ifreq[16])) = IFF_TUN | IFF_NO_PI
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
	var lastErr error
	for attempt := 0; attempt < i.opts.RetryAttempts; attempt++ {
		if err := i.applyConfiguration(cfg); err == nil {
			configured = true
			break
		} else {
			lastErr = err
			time.Sleep(time.Duration(i.opts.RetryDelay) * time.Millisecond)
		}
	}

	if !configured {
		i.setState(StateError)
		return fmt.Errorf("configuration failed after %d attempts: %v", i.opts.RetryAttempts, lastErr)
	}

	i.address = cfg.Address
	i.mtu = cfg.MTU
	i.isUp = true

	return nil
}

func (i *linuxInterface) applyConfiguration(cfg *Config) error {
	// Verify interface exists
	if _, err := os.Stat(fmt.Sprintf("/sys/class/net/%s", i.name)); err != nil {
		return fmt.Errorf("interface does not exist: %w", err)
	}

	// Set MTU first
	if out, err := exec.Command("sudo", "ip", "link", "set", "dev", i.name, "mtu", fmt.Sprintf("%d", cfg.MTU)).CombinedOutput(); err != nil {
		return fmt.Errorf("failed to set MTU: %w (output: %s)", err, string(out))
	}

	// Set IP address
	if out, err := exec.Command("sudo", "ip", "addr", "add", cfg.Address, "dev", i.name).CombinedOutput(); err != nil {
		return fmt.Errorf("failed to set IP address: %w (output: %s)", err, string(out))
	}

	// Bring interface up last
	if out, err := exec.Command("sudo", "ip", "link", "set", "dev", i.name, "up").CombinedOutput(); err != nil {
		return fmt.Errorf("failed to bring interface up: %w (output: %s)", err, string(out))
	}

	// Verify configuration
	if i.opts.ValidateState {
		out, err := exec.Command("sudo", "ip", "addr", "show", "dev", i.name).CombinedOutput()
		if err != nil {
			return fmt.Errorf("failed to validate interface configuration: %w (output: %s)", err, string(out))
		}
		fmt.Printf("Interface %s configuration: %s\n", i.name, string(out))
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
		if out, err := exec.Command("sudo", "ip", "link", "set", "dev", i.name, "down").CombinedOutput(); err != nil {
			return fmt.Errorf("failed to bring interface down: %w (output: %s)", err, string(out))
		}

		// Remove IP address with retries
		for attempt := 0; attempt < i.opts.RetryAttempts; attempt++ {
			out, err := exec.Command("sudo", "ip", "addr", "del", i.address, "dev", i.name).CombinedOutput()
			if err == nil {
				break
			} else if attempt == i.opts.RetryAttempts-1 {
				return fmt.Errorf("failed to remove IP address after %d attempts: %w (output: %s)", i.opts.RetryAttempts, err, string(out))
			}
			time.Sleep(time.Duration(i.opts.RetryDelay) * time.Millisecond)
		}

		// Delete the TUN interface
		if out, err := exec.Command("sudo", "ip", "tuntap", "del", "dev", i.name, "mode", "tun").CombinedOutput(); err != nil {
			return fmt.Errorf("failed to delete TUN interface: %w (output: %s)", err, string(out))
		}

		i.isUp = false
	}

	return i.Close()
}
