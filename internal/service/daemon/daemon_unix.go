package daemon

import (
	"fmt"
	"io/ioutil"
	"os"
	"strconv"
	"syscall"

	"github.com/o3willard-AI/SSSonector/internal/service"
	"go.uber.org/zap"
)

// UnixDaemon extends base daemon with Unix-specific functionality
type UnixDaemon struct {
	*Daemon
	logger *zap.Logger
}

// NewUnix creates a new Unix daemon instance
func NewUnix(opts service.ServiceOptions, logger *zap.Logger) (*UnixDaemon, error) {
	baseDaemon, err := New(opts, logger)
	if err != nil {
		return nil, err
	}

	return &UnixDaemon{
		Daemon: baseDaemon,
		logger: logger,
	}, nil
}

// StartUnix initializes the Unix daemon process
func (d *UnixDaemon) StartUnix() error {
	// Call base daemon start
	if err := d.Daemon.Start(); err != nil {
		return err
	}

	// Write PID file
	pidFile := d.GetOptions().PIDFile
	if pidFile != "" {
		pid := os.Getpid()
		if err := ioutil.WriteFile(pidFile, []byte(strconv.Itoa(pid)), 0644); err != nil {
			return fmt.Errorf("failed to write PID file: %w", err)
		}
	}

	return nil
}

// StopUnix cleans up Unix daemon resources
func (d *UnixDaemon) StopUnix() error {
	// Call base daemon stop
	if err := d.Daemon.Stop(); err != nil {
		return err
	}

	// Remove PID file
	pidFile := d.GetOptions().PIDFile
	if pidFile != "" {
		if err := os.Remove(pidFile); err != nil && !os.IsNotExist(err) {
			return fmt.Errorf("failed to remove PID file: %w", err)
		}
	}

	return nil
}

// GetPID returns the daemon process ID from PID file
func (d *UnixDaemon) GetPID() (int, error) {
	pidFile := d.GetOptions().PIDFile
	if pidFile == "" {
		return 0, fmt.Errorf("PID file not configured")
	}

	data, err := ioutil.ReadFile(pidFile)
	if err != nil {
		return 0, fmt.Errorf("failed to read PID file: %w", err)
	}

	pid, err := strconv.Atoi(string(data))
	if err != nil {
		return 0, fmt.Errorf("invalid PID file content: %w", err)
	}

	return pid, nil
}

// IsRunning checks if daemon process is running
func (d *UnixDaemon) IsRunning() (bool, error) {
	pid, err := d.GetPID()
	if err != nil {
		return false, err
	}

	// Check if process exists
	process, err := os.FindProcess(pid)
	if err != nil {
		return false, nil
	}

	// Send signal 0 to check if process is running
	err = process.Signal(syscall.Signal(0))
	if err != nil {
		return false, nil
	}

	return true, nil
}
