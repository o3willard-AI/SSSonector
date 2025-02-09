//go:build linux

package daemon

import (
	"fmt"

	"github.com/o3willard-AI/SSSonector/internal/service"
	"go.uber.org/zap"
)

// LinuxDaemon extends the base daemon with Linux-specific functionality
type LinuxDaemon struct {
	*Daemon
	systemd *SystemdManager
}

// NewLinux creates a new Linux daemon instance
func NewLinux(opts service.ServiceOptions, logger *zap.Logger) (*LinuxDaemon, error) {
	baseDaemon, err := New(opts, logger)
	if err != nil {
		return nil, err
	}

	systemd := NewSystemdManager(logger)

	return &LinuxDaemon{
		Daemon:  baseDaemon,
		systemd: systemd,
	}, nil
}

// Start initializes the Linux daemon process
func (d *LinuxDaemon) Start() error {
	// Initialize base daemon
	if err := d.Daemon.Start(); err != nil {
		return err
	}

	// Set up Linux-specific resource limits
	if err := setResourceLimits(); err != nil {
		return fmt.Errorf("failed to set Linux limits: %w", err)
	}

	// Start systemd management
	if err := d.systemd.Start(); err != nil {
		return fmt.Errorf("failed to start systemd management: %w", err)
	}

	// Notify systemd we're ready
	if err := d.systemd.NotifyReady(); err != nil {
		return fmt.Errorf("failed to notify systemd: %w", err)
	}

	return nil
}

// Stop cleans up Linux daemon resources
func (d *LinuxDaemon) Stop() error {
	// Stop systemd management
	if err := d.systemd.Stop(); err != nil {
		d.logger.Error("Failed to stop systemd management",
			zap.Error(err))
	}

	// Stop base daemon
	if err := d.Daemon.Stop(); err != nil {
		return fmt.Errorf("failed to stop base daemon: %w", err)
	}

	return nil
}

// Reload handles configuration reload
func (d *LinuxDaemon) Reload() error {
	// Notify systemd we're reloading
	if err := d.systemd.NotifyReloading(); err != nil {
		return fmt.Errorf("failed to notify systemd of reload: %w", err)
	}

	// Perform reload logic here
	// ...

	// Notify systemd we're ready again
	if err := d.systemd.NotifyReady(); err != nil {
		return fmt.Errorf("failed to notify systemd after reload: %w", err)
	}

	return nil
}

// UpdateStatus sends a status update to systemd
func (d *LinuxDaemon) UpdateStatus(status string) error {
	return d.systemd.UpdateStatus(status)
}

// LogError logs an error to the systemd journal
func (d *LinuxDaemon) LogError(msg string, err error) {
	d.systemd.LogError(msg, err)
}

// LogInfo logs an info message to the systemd journal
func (d *LinuxDaemon) LogInfo(msg string) {
	d.systemd.LogInfo(msg)
}

// IsSystemd returns true if running under systemd
func (d *LinuxDaemon) IsSystemd() bool {
	return d.systemd.IsSystemd()
}
