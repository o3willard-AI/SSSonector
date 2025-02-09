package daemon

import (
	"fmt"
	"os"
	"time"

	"github.com/o3willard-AI/SSSonector/internal/security"
	"github.com/o3willard-AI/SSSonector/internal/service"
	"go.uber.org/zap"
)

// LinuxDaemon extends the base daemon with Linux-specific functionality
type LinuxDaemon struct {
	*Daemon
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
	baseDaemon, err := New(opts, logger)
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
		Daemon:   baseDaemon,
		systemd:  systemd,
		cgroup:   cgroup,
		ns:       ns,
		security: securityMgr,
		logger:   logger,
		options:  opts,
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

	// Stop base daemon
	if err := d.Daemon.Stop(); err != nil {
		return fmt.Errorf("failed to stop base daemon: %w", err)
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

// Status returns the current daemon status
func (d *LinuxDaemon) Status() (string, error) {
	return d.status.State, nil
}

// GetPID returns the daemon process ID
func (d *LinuxDaemon) GetPID() (int, error) {
	return d.status.PID, nil
}

// IsRunning returns whether the daemon is running
func (d *LinuxDaemon) IsRunning() (bool, error) {
	return d.status.State == service.StateRunning, nil
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

// ExecuteCommand executes a daemon command
func (d *LinuxDaemon) ExecuteCommand(cmd service.ServiceCommand) (string, error) {
	switch cmd.Command {
	case service.CmdStatus:
		return d.Status()
	case service.CmdReload:
		return "", d.Reload()
	case service.CmdMetrics:
		metrics, err := d.GetMetrics()
		if err != nil {
			return "", err
		}
		return fmt.Sprintf("%+v", metrics), nil
	case service.CmdHealth:
		return "", d.Health()
	default:
		return "", fmt.Errorf("unknown command: %s", cmd.Command)
	}
}

// GetLogs returns daemon logs
func (d *LinuxDaemon) GetLogs(lines int) ([]string, error) {
	// Not implemented
	return nil, nil
}

// SendSignal sends a signal to the daemon
func (d *LinuxDaemon) SendSignal(signal string) error {
	// Not implemented
	return nil
}

// Configure configures the daemon
func (d *LinuxDaemon) Configure(opts service.ServiceOptions) error {
	// Not implemented
	return nil
}

// GetOptions returns daemon options
func (d *LinuxDaemon) GetOptions() service.ServiceOptions {
	return d.options
}
