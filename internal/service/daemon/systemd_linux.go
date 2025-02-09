package daemon

import (
	"fmt"
	"os"
	"syscall"

	"go.uber.org/zap"
)

const (
	// Systemd notification messages
	notifyReady    = "READY=1"
	notifyReload   = "RELOADING=1"
	notifyStopping = "STOPPING=1"
	notifyStatus   = "STATUS=%s"

	// Systemd environment variables
	envNotifySocket = "NOTIFY_SOCKET"
)

// SystemdManager manages systemd integration
type SystemdManager struct {
	logger     *zap.Logger
	isSystemd  bool
	notifyFd   int
	notifyAddr string
}

// NewSystemdManager creates a new systemd manager
func NewSystemdManager(logger *zap.Logger) *SystemdManager {
	m := &SystemdManager{
		logger: logger,
	}

	// Check if running under systemd
	m.isSystemd = os.Getenv(envNotifySocket) != ""
	if m.isSystemd {
		m.notifyAddr = os.Getenv(envNotifySocket)
		m.logger.Info("Running under systemd",
			zap.String("notify_socket", m.notifyAddr))
	}

	return m
}

// Start initializes systemd integration
func (m *SystemdManager) Start() error {
	if !m.isSystemd {
		return nil
	}

	// Open notify socket
	fd, err := syscall.Socket(syscall.AF_UNIX, syscall.SOCK_DGRAM|syscall.SOCK_CLOEXEC, 0)
	if err != nil {
		return fmt.Errorf("failed to create notify socket: %w", err)
	}
	m.notifyFd = fd

	return nil
}

// Stop cleans up systemd integration
func (m *SystemdManager) Stop() error {
	if !m.isSystemd {
		return nil
	}

	if m.notifyFd > 0 {
		if err := syscall.Close(m.notifyFd); err != nil {
			return fmt.Errorf("failed to close notify socket: %w", err)
		}
		m.notifyFd = 0
	}

	return nil
}

// NotifyReady notifies systemd that service is ready
func (m *SystemdManager) NotifyReady() error {
	return m.notify(notifyReady)
}

// NotifyReloading notifies systemd that service is reloading
func (m *SystemdManager) NotifyReloading() error {
	return m.notify(notifyReload)
}

// NotifyStopping notifies systemd that service is stopping
func (m *SystemdManager) NotifyStopping() error {
	return m.notify(notifyStopping)
}

// NotifyStatus updates systemd service status
func (m *SystemdManager) NotifyStatus(status string) error {
	return m.notify(fmt.Sprintf(notifyStatus, status))
}

// IsSystemd returns whether running under systemd
func (m *SystemdManager) IsSystemd() bool {
	return m.isSystemd
}

// notify sends a notification to systemd
func (m *SystemdManager) notify(state string) error {
	if !m.isSystemd {
		return nil
	}

	if m.notifyFd <= 0 {
		return fmt.Errorf("notify socket not initialized")
	}

	addr := &syscall.SockaddrUnix{
		Name: m.notifyAddr,
	}

	if err := syscall.Sendto(m.notifyFd, []byte(state), 0, addr); err != nil {
		return fmt.Errorf("failed to send notification: %w", err)
	}

	return nil
}
