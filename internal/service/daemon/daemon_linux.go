package daemon

import (
	"fmt"
	"os"
	"time"

	"github.com/o3willard-AI/SSSonector/internal/security"
	"github.com/o3willard-AI/SSSonector/internal/service"
	"go.uber.org/zap"
)

// LinuxDaemon extends the Unix daemon with Linux-specific functionality
type LinuxDaemon struct {
	*UnixDaemon
	systemd  *SystemdManager
	cgroup   *CgroupManager
	ns       *NamespaceManager
	security *security.SecurityManager
	status   service.ServiceStatus
	logger   *zap.Logger
	options  service.ServiceOptions
}

// NewLinux creates a new Linux daemon instance
func NewLinux(opts service.ServiceOptions, logger *zap.Logger) (*LinuxDaemon, error) {
	unixDaemon, err := NewUnix(opts, logger)
	if err != nil {
		return nil, err
	}

	// Initialize systemd manager
	systemd := NewSystemdManager(logger)

	// Initialize cgroup manager
	cgroup, err := NewCgroupManager(logger, "sssonector")
	if err != nil {
		return nil, fmt.Errorf("failed to create cgroup manager: %w", err)
	}

	// Initialize namespace manager
	ns := NewNamespaceManager(logger)

	// Initialize security manager
	securityOpts := security.GetDefaultOptions()
	securityMgr, err := security.NewSecurityManager(securityOpts)
	if err != nil {
		return nil, fmt.Errorf("failed to create security manager: %w", err)
	}

	d := &LinuxDaemon{
		UnixDaemon: unixDaemon,
		systemd:    systemd,
		cgroup:     cgroup,
		ns:         ns,
		security:   securityMgr,
		logger:     logger,
		options:    opts,
		status: service.ServiceStatus{
			Name:      opts.Name,
			State:     service.StateStopped,
			StartTime: time.Now(),
		},
	}

	return d, nil
}

// Start initializes the Linux daemon process
func (d *LinuxDaemon) Start() error {
	// Initialize Unix daemon
	if err := d.UnixDaemon.StartUnix(); err != nil {
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

	// Initialize cgroup with default configuration
	if err := d.cgroup.Start(GetDefaultConfig()); err != nil {
		return fmt.Errorf("failed to initialize cgroup: %w", err)
	}

	// Configure and setup namespaces
	if err := d.ns.Configure(GetDefaultNamespaceConfig()); err != nil {
		return fmt.Errorf("failed to configure namespaces: %w", err)
	}
	if err := d.ns.SetupNamespaces(); err != nil {
		return fmt.Errorf("failed to setup namespaces: %w", err)
	}

	// Apply security hardening
	if err := d.security.Apply(); err != nil {
		return fmt.Errorf("failed to apply security hardening: %w", err)
	}

	// Notify systemd we're ready
	if err := d.systemd.NotifyReady(); err != nil {
		return fmt.Errorf("failed to notify systemd: %w", err)
	}

	d.status.State = service.StateRunning
	d.status.PID = os.Getpid()
	return nil
}

// Stop cleans up Linux daemon resources
func (d *LinuxDaemon) Stop() error {
	// Stop cgroup
	if err := d.cgroup.Stop(); err != nil {
		d.logger.Error("Failed to stop cgroup",
			zap.Error(err))
	}

	// Stop systemd management
	if err := d.systemd.Stop(); err != nil {
		d.logger.Error("Failed to stop systemd management",
			zap.Error(err))
	}

	// Stop Unix daemon
	if err := d.UnixDaemon.StopUnix(); err != nil {
		return fmt.Errorf("failed to stop Unix daemon: %w", err)
	}

	d.status.State = service.StateStopped
	d.logger.Info("Linux daemon stopped successfully")
	return nil
}

// Reload reloads the daemon configuration
func (d *LinuxDaemon) Reload() error {
	d.status.State = service.StateReloading

	// Notify systemd we're reloading
	if err := d.systemd.NotifyReloading(); err != nil {
		return fmt.Errorf("failed to notify systemd of reload: %w", err)
	}

	// Reload security settings
	if err := d.security.Apply(); err != nil {
		return fmt.Errorf("failed to reload security settings: %w", err)
	}

	// Reload cgroup settings
	if err := d.cgroup.Reload(); err != nil {
		return fmt.Errorf("failed to reload cgroup settings: %w", err)
	}

	// Notify systemd we're ready
	if err := d.systemd.NotifyReady(); err != nil {
		return fmt.Errorf("failed to notify systemd after reload: %w", err)
	}

	d.status.State = service.StateRunning
	d.status.LastReload = time.Now()
	return nil
}

// GetMetrics returns daemon metrics
func (d *LinuxDaemon) GetMetrics() (service.ServiceMetrics, error) {
	stats, err := d.cgroup.GetStats()
	if err != nil {
		return service.ServiceMetrics{}, fmt.Errorf("failed to get cgroup stats: %w", err)
	}

	metrics := service.ServiceMetrics{
		StartTime:   d.status.StartTime,
		Platform:    "linux",
		LastError:   d.status.Error,
		CPUUsage:    stats.CPU.Usage,
		MemoryUsage: stats.Memory.Usage,
	}
	return metrics, nil
}

// Health checks daemon health
func (d *LinuxDaemon) Health() error {
	if !d.systemd.IsSystemd() {
		return fmt.Errorf("daemon not running under systemd")
	}
	return nil
}
