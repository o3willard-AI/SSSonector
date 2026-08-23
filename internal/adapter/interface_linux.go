//go:build linux
// +build linux

package adapter

import (
	"fmt"
	"os"
	"sync"
	"sync/atomic"
	"syscall"
	"time"
	"unsafe"

	"github.com/vishvananda/netlink"
	"go.uber.org/zap"
)

const (
	TUNSETIFF = 0x400454ca
	IFF_TUN   = 0x0001
	IFF_NO_PI = 0x1000
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
	if opts.Logger == nil {
		opts.Logger = zap.NewNop()
	}
	if err := ValidateInterfaceName(name); err != nil {
		return nil, err
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

// initialize opens /dev/net/tun and creates the interface natively via the
// TUNSETIFF ioctl. Requires the process to run as root or hold
// CAP_NET_ADMIN; no external commands are involved. A stale interface with
// the requested name is torn down first so the name can be reused.
func (i *linuxInterface) initialize() error {
	if !i.transitionState(StateUninitialized, StateInitializing) {
		return ErrInvalidStateTransition
	}
	i.opts.Logger.Info("Initializing TUN interface", zap.String("name", i.name))

	i.removeStaleLink()

	file, err := os.OpenFile("/dev/net/tun", os.O_RDWR, 0)
	if err != nil {
		i.opts.Logger.Error("Failed to open TUN device file",
			zap.String("path", "/dev/net/tun"),
			zap.Error(err),
		)
		i.setState(StateError)
		return fmt.Errorf("failed to open /dev/net/tun: %w", err)
	}

	ifreq, err := createIfreq(i.name)
	if err != nil {
		file.Close()
		i.setState(StateError)
		return err
	}

	// #nosec G103 -- TUN ioctl requires an ifreq struct; the layout is fixed
	// by the kernel ABI and the interface name is validated in New().
	// #nosec G103 -- TUN ioctl requires an ifreq struct; the layout is fixed
	// by the kernel ABI and the interface name is validated in New().
	_, _, errno := syscall.Syscall(syscall.SYS_IOCTL, file.Fd(), uintptr(TUNSETIFF), uintptr(unsafe.Pointer(&ifreq[0])))
	if errno != 0 {
		i.opts.Logger.Error("TUNSETIFF ioctl failed",
			zap.String("name", i.name),
			zap.Uint("errno", uint(errno)),
			zap.String("errno_name", errnoName(errno)),
		)
		file.Close()
		i.setState(StateError)
		return fmt.Errorf("failed to create TUN interface: %v", errno)
	}
	i.file = file
	i.mtu = 1500 // Default MTU
	i.setState(StateReady)
	i.opts.Logger.Info("TUN interface initialized", zap.String("name", i.name))
	return nil
}

// removeStaleLink tears down a leftover interface with the requested name
// (e.g. after an unclean shutdown). Absent links are not an error.
func (i *linuxInterface) removeStaleLink() {
	link, err := netlink.LinkByName(i.name)
	if err != nil {
		return
	}
	i.opts.Logger.Info("Removing pre-existing interface", zap.String("name", i.name))
	if err := netlink.LinkDel(link); err != nil {
		i.opts.Logger.Warn("Failed to remove existing interface",
			zap.String("name", i.name),
			zap.Error(err),
		)
	}
	time.Sleep(time.Second) // Allow kernel teardown to settle
}

// errnoName maps a syscall.Errno to its symbolic name for log readability
func errnoName(errno syscall.Errno) string {
	if s, ok := map[syscall.Errno]string{
		syscall.EPERM:  "EPERM",
		syscall.ENOENT: "ENOENT",
		syscall.EBUSY:  "EBUSY",
		syscall.EINVAL: "EINVAL",
		syscall.ENOTTY: "ENOTTY",
	}[errno]; ok {
		return s
	}
	return errno.Error()
}

func createIfreq(name string) ([]byte, error) {
	var ifreq [40]byte
	copy(ifreq[:16], []byte(name))
	// #nosec G103 -- kernel ABI: flags field lives at offset 16 of ifreq
	*(*uint16)(unsafe.Pointer(&ifreq[16])) = IFF_TUN | IFF_NO_PI
	return ifreq[:], nil
}

func (i *linuxInterface) Configure(cfg *Config) error {
	if state := i.getState(); state != StateReady {
		return fmt.Errorf("%w: current state %s", ErrInterfaceNotReady, state)
	}

	i.opts.Logger.Info("Configuring TUN interface",
		zap.String("name", cfg.Name),
		zap.String("address", cfg.Address),
		zap.Int("mtu", cfg.MTU),
	)

	// Parse IP address and network to validate format
	addr, err := netlink.ParseAddr(cfg.Address)
	if err != nil {
		i.opts.Logger.Error("Invalid TUN address format",
			zap.String("address", cfg.Address),
			zap.Error(err),
		)
		i.setState(StateError)
		return fmt.Errorf("invalid address format: %w", err)
	}

	configured := false
	var lastErr error
	for attempt := 0; attempt < i.opts.RetryAttempts; attempt++ {
		if err := i.applyConfiguration(cfg, addr); err == nil {
			configured = true
			break
		} else {
			lastErr = err
			i.opts.Logger.Warn("TUN configuration attempt failed",
				zap.Int("attempt", attempt+1),
				zap.Int("max_attempts", i.opts.RetryAttempts),
				zap.Error(err),
			)
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

	i.opts.Logger.Info("TUN interface configured",
		zap.String("name", i.name),
		zap.String("address", i.address),
		zap.Int("mtu", i.mtu),
	)
	return nil
}

func (i *linuxInterface) applyConfiguration(cfg *Config, addr *netlink.Addr) error {
	link, err := netlink.LinkByName(i.name)
	if err != nil {
		return fmt.Errorf("interface does not exist: %w", err)
	}

	if err := netlink.LinkSetMTU(link, cfg.MTU); err != nil {
		return fmt.Errorf("failed to set MTU: %w", err)
	}

	if err := netlink.AddrAdd(link, addr); err != nil && !os.IsExist(err) {
		return fmt.Errorf("failed to set IP address: %w", err)
	}

	if err := netlink.LinkSetUp(link); err != nil {
		return fmt.Errorf("failed to bring interface up: %w", err)
	}

	// Verify configuration
	if i.opts.ValidateState {
		addrs, err := netlink.AddrList(link, netlink.FAMILY_V4)
		if err != nil {
			return fmt.Errorf("failed to validate interface configuration: %w", err)
		}
		found := false
		for _, a := range addrs {
			if a.String() == addr.String() {
				found = true
				break
			}
		}
		if !found {
			return fmt.Errorf("configured address %s not present after setup", addr.String())
		}
		i.opts.Logger.Debug("Interface configuration verified",
			zap.String("name", i.name),
			zap.Strings("addresses", addrStrings(addrs)),
		)
	}

	return nil
}

func addrStrings(addrs []netlink.Addr) []string {
	out := make([]string, 0, len(addrs))
	for _, a := range addrs {
		out = append(out, a.String())
	}
	return out
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
	i.opts.Logger.Info("Cleaning up TUN interface", zap.String("name", i.name))

	done := make(chan error, 1)
	go func() {
		done <- i.performCleanup()
	}()

	select {
	case err := <-done:
		if err != nil {
			i.opts.Logger.Error("TUN interface cleanup failed",
				zap.String("name", i.name),
				zap.Error(err),
			)
			i.setState(StateError)
			return err
		}
		i.setState(StateStopped)
		i.opts.Logger.Info("TUN interface cleaned up", zap.String("name", i.name))
		return nil
	case <-time.After(time.Duration(i.opts.CleanupTimeout) * time.Millisecond):
		i.opts.Logger.Error("TUN interface cleanup timed out",
			zap.String("name", i.name),
			zap.Int("timeout_ms", i.opts.CleanupTimeout),
		)
		i.setState(StateError)
		return ErrCleanupTimeout
	}
}

// performCleanup withdraws the address and brings the link down; closing
// the file descriptor destroys the unpersisted TUN device.
func (i *linuxInterface) performCleanup() error {
	if i.isUp {
		link, err := netlink.LinkByName(i.name)
		if err == nil {
			_ = netlink.LinkSetDown(link)
			if addr, err := netlink.ParseAddr(i.address); err == nil {
				_ = netlink.AddrDel(link, addr)
			}
		}
		i.isUp = false
	}

	return i.Close()
}
