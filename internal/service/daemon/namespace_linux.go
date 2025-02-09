//go:build linux

package daemon

import (
	"fmt"
	"os"
	"strings"

	"github.com/vishvananda/netlink"
	"go.uber.org/zap"
	"golang.org/x/sys/unix"
)

// NamespaceConfig represents namespace configuration options
type NamespaceConfig struct {
	// Network namespace options
	NetworkNS     bool   // Create new network namespace
	NetworkVeth   string // Virtual ethernet device name
	NetworkBridge string // Bridge device name
	NetworkIP     string // IP address for veth device
	NetworkMask   string // Network mask for veth device

	// Mount namespace options
	MountNS       bool     // Create new mount namespace
	MountProc     bool     // Mount procfs
	MountSys      bool     // Mount sysfs
	MountTmp      bool     // Mount tmpfs
	MountBinds    []string // Bind mounts (src:dst)
	MountRootfs   string   // Root filesystem path
	MountReadonly bool     // Make rootfs readonly

	// PID namespace options
	PidNS bool // Create new PID namespace

	// IPC namespace options
	IpcNS bool // Create new IPC namespace

	// UTS namespace options
	UtsNS       bool   // Create new UTS namespace
	UtsHostname string // Hostname in new UTS namespace

	// User namespace options
	UserNS       bool     // Create new user namespace
	UserMap      []string // UID mappings (container:host:size)
	UserMapGroup []string // GID mappings (container:host:size)
}

// NamespaceManager handles Linux namespace management
type NamespaceManager struct {
	logger  *zap.Logger
	config  NamespaceConfig
	nsFlags uintptr
}

// NewNamespaceManager creates a new namespace manager
func NewNamespaceManager(logger *zap.Logger) *NamespaceManager {
	return &NamespaceManager{
		logger: logger,
	}
}

// Configure sets up namespace configuration
func (m *NamespaceManager) Configure(config NamespaceConfig) error {
	m.config = config
	m.nsFlags = 0

	// Set namespace flags based on configuration
	if config.NetworkNS {
		m.nsFlags |= unix.CLONE_NEWNET
	}
	if config.MountNS {
		m.nsFlags |= unix.CLONE_NEWNS
	}
	if config.PidNS {
		m.nsFlags |= unix.CLONE_NEWPID
	}
	if config.IpcNS {
		m.nsFlags |= unix.CLONE_NEWIPC
	}
	if config.UtsNS {
		m.nsFlags |= unix.CLONE_NEWUTS
	}
	if config.UserNS {
		m.nsFlags |= unix.CLONE_NEWUSER
	}

	return nil
}

// SetupNamespaces sets up configured namespaces
func (m *NamespaceManager) SetupNamespaces() error {
	if m.nsFlags == 0 {
		return nil // No namespaces configured
	}

	// Unshare namespaces
	if err := unix.Unshare(int(m.nsFlags)); err != nil {
		return fmt.Errorf("failed to unshare namespaces: %w", err)
	}

	// Set hostname in UTS namespace
	if m.config.UtsNS && m.config.UtsHostname != "" {
		if err := unix.Sethostname([]byte(m.config.UtsHostname)); err != nil {
			return fmt.Errorf("failed to set hostname: %w", err)
		}
	}

	// Setup mount namespace
	if m.config.MountNS {
		if err := m.setupMountNamespace(); err != nil {
			return fmt.Errorf("failed to setup mount namespace: %w", err)
		}
	}

	// Setup network namespace
	if m.config.NetworkNS {
		if err := m.setupNetworkNamespace(); err != nil {
			return fmt.Errorf("failed to setup network namespace: %w", err)
		}
	}

	// Setup user namespace mappings
	if m.config.UserNS {
		if err := m.setupUserNamespace(); err != nil {
			return fmt.Errorf("failed to setup user namespace: %w", err)
		}
	}

	return nil
}

// setupMountNamespace configures the mount namespace
func (m *NamespaceManager) setupMountNamespace() error {
	// Make current mount namespace private
	if err := unix.Mount("none", "/", "", unix.MS_REC|unix.MS_PRIVATE, ""); err != nil {
		return fmt.Errorf("failed to make mounts private: %w", err)
	}

	if m.config.MountRootfs != "" {
		// Mount new root filesystem
		if err := unix.Mount(m.config.MountRootfs, m.config.MountRootfs, "", unix.MS_BIND|unix.MS_REC, ""); err != nil {
			return fmt.Errorf("failed to bind mount rootfs: %w", err)
		}

		// Make readonly if configured
		if m.config.MountReadonly {
			if err := unix.Mount("none", m.config.MountRootfs, "", unix.MS_REMOUNT|unix.MS_BIND|unix.MS_RDONLY, ""); err != nil {
				return fmt.Errorf("failed to make rootfs readonly: %w", err)
			}
		}

		// Change root
		if err := unix.Chroot(m.config.MountRootfs); err != nil {
			return fmt.Errorf("failed to chroot: %w", err)
		}
		if err := os.Chdir("/"); err != nil {
			return fmt.Errorf("failed to chdir to new root: %w", err)
		}
	}

	// Mount pseudo filesystems
	if m.config.MountProc {
		if err := unix.Mount("proc", "/proc", "proc", 0, ""); err != nil {
			return fmt.Errorf("failed to mount proc: %w", err)
		}
	}
	if m.config.MountSys {
		if err := unix.Mount("sysfs", "/sys", "sysfs", 0, ""); err != nil {
			return fmt.Errorf("failed to mount sysfs: %w", err)
		}
	}
	if m.config.MountTmp {
		if err := unix.Mount("tmpfs", "/tmp", "tmpfs", 0, ""); err != nil {
			return fmt.Errorf("failed to mount tmpfs: %w", err)
		}
	}

	// Setup bind mounts
	for _, bind := range m.config.MountBinds {
		src, dst, found := strings.Cut(bind, ":")
		if !found {
			return fmt.Errorf("invalid bind mount specification: %s", bind)
		}

		if err := os.MkdirAll(dst, 0755); err != nil {
			return fmt.Errorf("failed to create bind mount target: %w", err)
		}

		if err := unix.Mount(src, dst, "", unix.MS_BIND, ""); err != nil {
			return fmt.Errorf("failed to create bind mount %s -> %s: %w", src, dst, err)
		}
	}

	return nil
}

// setupNetworkNamespace configures the network namespace
func (m *NamespaceManager) setupNetworkNamespace() error {
	if m.config.NetworkVeth == "" {
		return nil // No network setup needed
	}

	// Create veth pair
	if err := m.createVethPair(); err != nil {
		return fmt.Errorf("failed to create veth pair: %w", err)
	}

	// Setup IP address
	if m.config.NetworkIP != "" {
		if err := m.setupIPAddress(); err != nil {
			return fmt.Errorf("failed to setup IP address: %w", err)
		}
	}

	return nil
}

// setupUserNamespace configures the user namespace
func (m *NamespaceManager) setupUserNamespace() error {
	// Write UID mappings
	uidMap := "/proc/self/uid_map"
	if err := m.writeIDMapping(uidMap, m.config.UserMap); err != nil {
		return fmt.Errorf("failed to write UID mapping: %w", err)
	}

	// Write GID mappings
	gidMap := "/proc/self/gid_map"
	if err := m.writeIDMapping(gidMap, m.config.UserMapGroup); err != nil {
		return fmt.Errorf("failed to write GID mapping: %w", err)
	}

	return nil
}

// writeIDMapping writes ID mappings to a proc file
func (m *NamespaceManager) writeIDMapping(path string, mappings []string) error {
	f, err := os.OpenFile(path, os.O_WRONLY, 0)
	if err != nil {
		return err
	}
	defer f.Close()

	for _, mapping := range mappings {
		if _, err := f.WriteString(mapping + "\n"); err != nil {
			return err
		}
	}

	return nil
}

// createVethPair creates a veth pair for network namespace
func (m *NamespaceManager) createVethPair() error {
	// Create veth pair
	veth := &netlink.Veth{
		LinkAttrs: netlink.LinkAttrs{Name: m.config.NetworkVeth},
		PeerName:  m.config.NetworkVeth + "-peer",
	}

	if err := netlink.LinkAdd(veth); err != nil {
		return fmt.Errorf("failed to create veth pair: %w", err)
	}

	// Get veth link
	link, err := netlink.LinkByName(m.config.NetworkVeth)
	if err != nil {
		return fmt.Errorf("failed to get veth link: %w", err)
	}

	// Set veth up
	if err := netlink.LinkSetUp(link); err != nil {
		return fmt.Errorf("failed to set veth up: %w", err)
	}

	// Add to bridge if configured
	if m.config.NetworkBridge != "" {
		bridge, err := netlink.LinkByName(m.config.NetworkBridge)
		if err != nil {
			return fmt.Errorf("failed to get bridge: %w", err)
		}

		if err := netlink.LinkSetMaster(link, bridge); err != nil {
			return fmt.Errorf("failed to add veth to bridge: %w", err)
		}
	}

	return nil
}

// setupIPAddress configures IP address on veth device
func (m *NamespaceManager) setupIPAddress() error {
	// Get veth link
	link, err := netlink.LinkByName(m.config.NetworkVeth)
	if err != nil {
		return fmt.Errorf("failed to get veth link: %w", err)
	}

	// Parse IP address
	addr, err := netlink.ParseAddr(m.config.NetworkIP + "/" + m.config.NetworkMask)
	if err != nil {
		return fmt.Errorf("failed to parse IP address: %w", err)
	}

	// Add IP address to veth
	if err := netlink.AddrAdd(link, addr); err != nil {
		return fmt.Errorf("failed to add IP address: %w", err)
	}

	return nil
}

// GetDefaultConfig returns default namespace configuration
func GetDefaultNamespaceConfig() NamespaceConfig {
	return NamespaceConfig{
		NetworkNS: true,
		MountNS:   true,
		PidNS:     true,
		IpcNS:     true,
		UtsNS:     true,
		UserNS:    false, // Disabled by default as it requires additional setup
	}
}
