package daemon

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"golang.org/x/sys/unix"
)

// setupTmpfsMount sets up a tmpfs mount
func (m *NamespaceManager) setupTmpfsMount(mount *TmpfsMount) error {
	// Create target directory if it doesn't exist
	if err := os.MkdirAll(mount.Target, 0755); err != nil {
		return fmt.Errorf("failed to create target directory: %w", err)
	}

	// Parse size option
	sizeOpt := fmt.Sprintf("size=%s", mount.Size)
	modeOpt := fmt.Sprintf("mode=%o", mount.Mode)
	data := strings.Join([]string{sizeOpt, modeOpt}, ",")

	// Mount tmpfs
	if err := unix.Mount("tmpfs", mount.Target, "tmpfs", uintptr(0), data); err != nil {
		return fmt.Errorf("failed to mount tmpfs: %w", err)
	}

	return nil
}

// setupBindMount sets up a bind mount
func (m *NamespaceManager) setupBindMount(mount *BindMount) error {
	// Create target directory if it doesn't exist
	if err := os.MkdirAll(mount.Target, 0755); err != nil {
		return fmt.Errorf("failed to create target directory: %w", err)
	}

	// Perform bind mount
	flags := uintptr(unix.MS_BIND)
	if mount.RO {
		flags |= uintptr(unix.MS_RDONLY)
	}

	if err := unix.Mount(mount.Source, mount.Target, "", flags, ""); err != nil {
		return fmt.Errorf("failed to bind mount: %w", err)
	}

	// Make mount read-only if requested
	if mount.RO {
		flags |= uintptr(unix.MS_REMOUNT)
		if err := unix.Mount("", mount.Target, "", flags, ""); err != nil {
			return fmt.Errorf("failed to make mount read-only: %w", err)
		}
	}

	return nil
}

// setupMount sets up a regular mount
func (m *NamespaceManager) setupMount(mount *MountConfig) error {
	// Create target directory if it doesn't exist
	if err := os.MkdirAll(mount.Target, 0755); err != nil {
		return fmt.Errorf("failed to create target directory: %w", err)
	}

	// Build mount flags
	flags := uintptr(mount.Flags)
	if mount.Private {
		flags |= uintptr(unix.MS_PRIVATE)
	}
	if mount.Slave {
		flags |= uintptr(unix.MS_SLAVE)
	}
	if mount.Shared {
		flags |= uintptr(unix.MS_SHARED)
	}
	if mount.Unbindable {
		flags |= uintptr(unix.MS_UNBINDABLE)
	}

	// Perform mount
	if err := unix.Mount(mount.Source, mount.Target, mount.Type, flags, mount.Data); err != nil {
		return fmt.Errorf("failed to mount %s: %w", mount.Type, err)
	}

	return nil
}

// setupProcMount sets up proc filesystem
func (m *NamespaceManager) setupProcMount(target string) error {
	// Create target directory if it doesn't exist
	if err := os.MkdirAll(target, 0755); err != nil {
		return fmt.Errorf("failed to create proc directory: %w", err)
	}

	// Mount proc
	if err := unix.Mount("proc", target, "proc", uintptr(0), ""); err != nil {
		return fmt.Errorf("failed to mount proc: %w", err)
	}

	return nil
}

// setupSysMount sets up sysfs filesystem
func (m *NamespaceManager) setupSysMount(target string) error {
	// Create target directory if it doesn't exist
	if err := os.MkdirAll(target, 0755); err != nil {
		return fmt.Errorf("failed to create sys directory: %w", err)
	}

	// Mount sysfs
	if err := unix.Mount("sysfs", target, "sysfs", uintptr(0), ""); err != nil {
		return fmt.Errorf("failed to mount sysfs: %w", err)
	}

	return nil
}

// setupDevMount sets up dev filesystem
func (m *NamespaceManager) setupDevMount(target string) error {
	// Create target directory if it doesn't exist
	if err := os.MkdirAll(target, 0755); err != nil {
		return fmt.Errorf("failed to create dev directory: %w", err)
	}

	// Mount devtmpfs
	if err := unix.Mount("devtmpfs", target, "devtmpfs", uintptr(0), ""); err != nil {
		return fmt.Errorf("failed to mount devtmpfs: %w", err)
	}

	// Create required device nodes
	devices := []struct {
		path string
		mode uint32
		dev  uint64
	}{
		{filepath.Join(target, "null"), 0666, unix.Mkdev(1, 3)},
		{filepath.Join(target, "zero"), 0666, unix.Mkdev(1, 5)},
		{filepath.Join(target, "full"), 0666, unix.Mkdev(1, 7)},
		{filepath.Join(target, "random"), 0666, unix.Mkdev(1, 8)},
		{filepath.Join(target, "urandom"), 0666, unix.Mkdev(1, 9)},
		{filepath.Join(target, "tty"), 0666, unix.Mkdev(5, 0)},
	}

	for _, device := range devices {
		if err := unix.Mknod(device.path, unix.S_IFCHR|device.mode, int(device.dev)); err != nil && !os.IsExist(err) {
			return fmt.Errorf("failed to create device node %s: %w", device.path, err)
		}
	}

	return nil
}

// setupCgroupMount sets up cgroup filesystem
func (m *NamespaceManager) setupCgroupMount(target string) error {
	// Create target directory if it doesn't exist
	if err := os.MkdirAll(target, 0755); err != nil {
		return fmt.Errorf("failed to create cgroup directory: %w", err)
	}

	// Mount cgroup2 if available
	if err := unix.Mount("cgroup2", target, "cgroup2", uintptr(0), ""); err == nil {
		return nil
	}

	// Fall back to cgroup v1
	return unix.Mount("cgroup", target, "cgroup", uintptr(0), "")
}
