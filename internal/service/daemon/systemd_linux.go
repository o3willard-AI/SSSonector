//go:build linux

package daemon

import (
	"fmt"
	"os"
	"sync"
	"time"

	"github.com/coreos/go-systemd/v22/daemon"
	"github.com/coreos/go-systemd/v22/journal"
	"go.uber.org/zap"
)

// SystemdManager handles systemd-specific service operations
type SystemdManager struct {
	logger       *zap.Logger
	notifySocket string
	watchdogCh   chan struct{}
	stopCh       chan struct{}
	wg           sync.WaitGroup
	mu           sync.RWMutex
	status       string
}

// NewSystemdManager creates a new systemd service manager
func NewSystemdManager(logger *zap.Logger) *SystemdManager {
	return &SystemdManager{
		logger:       logger,
		notifySocket: os.Getenv("NOTIFY_SOCKET"),
		watchdogCh:   make(chan struct{}),
		stopCh:       make(chan struct{}),
	}
}

// Start initializes systemd service management
func (m *SystemdManager) Start() error {
	// Check if running under systemd
	if m.notifySocket == "" {
		return nil
	}

	// Start watchdog if enabled
	interval, err := daemon.SdWatchdogEnabled(false)
	if err == nil && interval > 0 {
		m.startWatchdog(time.Duration(interval) * time.Microsecond)
	}

	return nil
}

// Stop cleans up systemd service management
func (m *SystemdManager) Stop() error {
	if m.notifySocket == "" {
		return nil
	}

	// Stop watchdog
	close(m.stopCh)
	m.wg.Wait()

	// Notify stopping
	if _, err := daemon.SdNotify(false, daemon.SdNotifyStopping); err != nil {
		return fmt.Errorf("failed to notify stopping: %w", err)
	}

	return nil
}

// UpdateStatus updates the service status in systemd
func (m *SystemdManager) UpdateStatus(status string) error {
	m.mu.Lock()
	m.status = status
	m.mu.Unlock()

	if m.notifySocket == "" {
		return nil
	}

	msg := fmt.Sprintf("STATUS=%s", status)
	if _, err := daemon.SdNotify(false, msg); err != nil {
		return fmt.Errorf("failed to update status: %w", err)
	}

	return nil
}

// LogError logs an error to the systemd journal
func (m *SystemdManager) LogError(msg string, err error) {
	if journal.Enabled() {
		journal.Send(fmt.Sprintf("%s: %v", msg, err), journal.PriErr, nil)
	}
	m.logger.Error(msg, zap.Error(err))
}

// LogInfo logs an info message to the systemd journal
func (m *SystemdManager) LogInfo(msg string) {
	if journal.Enabled() {
		journal.Send(msg, journal.PriInfo, nil)
	}
	m.logger.Info(msg)
}

// startWatchdog starts the systemd watchdog
func (m *SystemdManager) startWatchdog(interval time.Duration) {
	m.wg.Add(1)
	go func() {
		defer m.wg.Done()

		ticker := time.NewTicker(interval / 2)
		defer ticker.Stop()

		for {
			select {
			case <-m.stopCh:
				return
			case <-ticker.C:
				if _, err := daemon.SdNotify(false, daemon.SdNotifyWatchdog); err != nil {
					m.logger.Error("Failed to send watchdog notification",
						zap.Error(err))
				}
			}
		}
	}()
}

// NotifyReady notifies systemd that the service is ready
func (m *SystemdManager) NotifyReady() error {
	if m.notifySocket == "" {
		return nil
	}

	if _, err := daemon.SdNotify(false, daemon.SdNotifyReady); err != nil {
		return fmt.Errorf("failed to notify ready: %w", err)
	}

	return nil
}

// NotifyReloading notifies systemd that the service is reloading
func (m *SystemdManager) NotifyReloading() error {
	if m.notifySocket == "" {
		return nil
	}

	if _, err := daemon.SdNotify(false, daemon.SdNotifyReloading); err != nil {
		return fmt.Errorf("failed to notify reloading: %w", err)
	}

	return nil
}

// GetStatus returns the current service status
func (m *SystemdManager) GetStatus() string {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.status
}

// IsSystemd returns true if running under systemd
func (m *SystemdManager) IsSystemd() bool {
	return m.notifySocket != ""
}

// GetWatchdogInterval returns the systemd watchdog interval
func (m *SystemdManager) GetWatchdogInterval() (time.Duration, error) {
	if m.notifySocket == "" {
		return 0, fmt.Errorf("not running under systemd")
	}

	usec, err := daemon.SdWatchdogEnabled(false)
	if err != nil || usec == 0 {
		return 0, fmt.Errorf("watchdog not enabled: %w", err)
	}

	return time.Duration(usec) * time.Microsecond, nil
}
