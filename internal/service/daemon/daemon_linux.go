package daemon

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"go.uber.org/zap"
)

// LinuxDaemon represents a Linux daemon
type LinuxDaemon struct {
	*UnixDaemon
	systemd bool
}

// NewLinuxDaemon creates a new Linux daemon
func NewLinuxDaemon(logger *zap.Logger, pidFile string) (*LinuxDaemon, error) {
	// Create Unix daemon
	unixDaemon, err := NewUnixDaemon(logger, pidFile)
	if err != nil {
		return nil, err
	}

	// Check if running under systemd
	systemd := os.Getenv("INVOCATION_ID") != "" || os.Getenv("JOURNAL_STREAM") != ""

	return &LinuxDaemon{
		UnixDaemon: unixDaemon,
		systemd:    systemd,
	}, nil
}

// Start starts the Linux daemon
func (d *LinuxDaemon) Start() error {
	// Create context with signal handling
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Set up signal handling
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan,
		syscall.SIGINT,
		syscall.SIGTERM,
		syscall.SIGHUP,
		syscall.SIGUSR1,
		syscall.SIGUSR2,
	)

	// Handle signals
	go func() {
		for sig := range sigChan {
			switch sig {
			case syscall.SIGINT, syscall.SIGTERM:
				d.logger.Info("Received shutdown signal", zap.String("signal", sig.String()))
				cancel()
			case syscall.SIGHUP:
				d.logger.Info("Received reload signal")
				if err := d.Reload(); err != nil {
					d.logger.Error("Failed to reload", zap.Error(err))
				}
			case syscall.SIGUSR1:
				d.logger.Info("Received status dump signal")
				status, err := d.Status()
				if err != nil {
					d.logger.Error("Failed to get status", zap.Error(err))
					continue
				}
				d.logger.Info("Daemon status",
					zap.Bool("running", status.Running),
					zap.Int("pid", status.Pid),
					zap.Any("stats", status.Stats),
				)
			}
		}
	}()

	// Write PID file
	if err := d.writePIDFile(); err != nil {
		return err
	}

	// Start daemon
	if err := d.Daemon.Start(ctx); err != nil {
		return err
	}

	// Wait for context cancellation
	<-ctx.Done()

	// Stop daemon
	return d.Stop()
}

// Stop stops the Linux daemon
func (d *LinuxDaemon) Stop() error {
	// Remove PID file
	if err := d.removePIDFile(); err != nil {
		return err
	}

	// Stop daemon
	return d.Daemon.Stop()
}

// Reload reloads the daemon configuration
func (d *LinuxDaemon) Reload() error {
	// Reload cgroup manager
	if err := d.cgManager.Reload(); err != nil {
		return fmt.Errorf("failed to reload cgroup manager: %w", err)
	}

	return nil
}

// IsSystemd returns whether the daemon is running under systemd
func (d *LinuxDaemon) IsSystemd() bool {
	return d.systemd
}
