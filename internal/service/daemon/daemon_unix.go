package daemon

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"go.uber.org/zap"
)

// UnixDaemon represents a Unix daemon
type UnixDaemon struct {
	*Daemon
	pidFile string
}

// NewUnixDaemon creates a new Unix daemon
func NewUnixDaemon(logger *zap.Logger, pidFile string) (*UnixDaemon, error) {
	daemon, err := NewDaemon(logger)
	if err != nil {
		return nil, err
	}

	return &UnixDaemon{
		Daemon:  daemon,
		pidFile: pidFile,
	}, nil
}

// Start starts the Unix daemon
func (d *UnixDaemon) Start() error {
	// Create context with signal handling
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Set up signal handling
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-sigChan
		cancel()
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

// Stop stops the Unix daemon
func (d *UnixDaemon) Stop() error {
	// Remove PID file
	if err := d.removePIDFile(); err != nil {
		return err
	}

	// Stop daemon
	return d.Daemon.Stop()
}

// Reload reloads the daemon configuration
func (d *UnixDaemon) Reload() error {
	// Reload cgroup manager
	if err := d.cgManager.Reload(); err != nil {
		return fmt.Errorf("failed to reload cgroup manager: %w", err)
	}

	return nil
}

// writePIDFile writes the daemon's PID to a file
func (d *UnixDaemon) writePIDFile() error {
	if d.pidFile == "" {
		return nil
	}

	pid := os.Getpid()
	return os.WriteFile(d.pidFile, []byte(fmt.Sprintf("%d", pid)), 0644)
}

// removePIDFile removes the daemon's PID file
func (d *UnixDaemon) removePIDFile() error {
	if d.pidFile == "" {
		return nil
	}

	if err := os.Remove(d.pidFile); err != nil && !os.IsNotExist(err) {
		return err
	}

	return nil
}
