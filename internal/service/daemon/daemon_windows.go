//go:build windows
// +build windows

package daemon

import (
	"context"
	"fmt"
	"sync"
	"time"

	"go.uber.org/zap"
	"golang.org/x/sys/windows"
	"golang.org/x/sys/windows/svc"
	"golang.org/x/sys/windows/svc/mgr"
)

// WindowsDaemon represents a Windows service daemon
type WindowsDaemon struct {
	*Daemon
	jobObject *JobObject
	service   *mgr.Service
	status    svc.Status
	stopCh    chan struct{}
	mu        sync.RWMutex
}

// NewWindowsDaemon creates a new Windows daemon
func NewWindowsDaemon(logger *zap.Logger) (*WindowsDaemon, error) {
	daemon, err := NewDaemon(logger)
	if err != nil {
		return nil, err
	}

	// Create Job Object for process isolation
	jobObject, err := NewJobObject(ServiceName)
	if err != nil {
		return nil, fmt.Errorf("failed to create job object: %w", err)
	}

	// Set basic limits
	basicLimits := &JobObjectBasicLimitInformation{
		LimitFlags:         JobObjectLimitKillOnJobClose,
		ActiveProcessLimit: 1,
		PriorityClass:      uint32(windows.NORMAL_PRIORITY_CLASS),
	}
	if err := jobObject.SetBasicLimits(basicLimits); err != nil {
		jobObject.Close()
		return nil, fmt.Errorf("failed to set basic limits: %w", err)
	}

	return &WindowsDaemon{
		Daemon:    daemon,
		jobObject: jobObject,
		stopCh:    make(chan struct{}),
	}, nil
}

// Start starts the Windows daemon
func (d *WindowsDaemon) Start(ctx context.Context) error {
	d.logger.Info("Starting Windows daemon")

	// Connect to Windows Service Manager
	m, err := mgr.Connect()
	if err != nil {
		return fmt.Errorf("failed to connect to service manager: %w", err)
	}
	defer m.Disconnect()

	// Open service
	service, err := m.OpenService(ServiceName)
	if err != nil {
		return fmt.Errorf("failed to open service: %w", err)
	}
	d.service = service

	// Update service status
	d.mu.Lock()
	d.status = svc.Status{
		State:   svc.Running,
		Accepts: svc.AcceptStop | svc.AcceptShutdown,
	}
	d.mu.Unlock()

	// Start base daemon
	if err := d.Daemon.Start(ctx); err != nil {
		return err
	}

	// Assign current process to Job Object
	if err := d.jobObject.AssignProcess(uint32(windows.GetCurrentProcessId())); err != nil {
		return fmt.Errorf("failed to assign process to job object: %w", err)
	}

	// Start monitoring goroutine
	d.wg.Add(1)
	go func() {
		defer d.wg.Done()
		d.monitor(ctx)
	}()

	return nil
}

// Stop stops the Windows daemon
func (d *WindowsDaemon) Stop() error {
	d.logger.Info("Stopping Windows daemon")

	// Signal monitoring goroutine to stop
	close(d.stopCh)

	// Wait for goroutines to finish
	d.wg.Wait()

	// Update service status
	d.mu.Lock()
	d.status = svc.Status{
		State: svc.Stopped,
	}
	d.mu.Unlock()

	// Clean up Job Object
	if err := d.jobObject.Close(); err != nil {
		d.logger.Error("Failed to close job object", zap.Error(err))
	}

	// Stop base daemon
	return d.Daemon.Stop()
}

// Execute handles Windows service control requests
func (d *WindowsDaemon) Execute(args []string, r <-chan svc.ChangeRequest, changes chan<- svc.Status) (bool, uint32) {
	const cmdsAccepted = svc.AcceptStop | svc.AcceptShutdown | svc.AcceptPauseAndContinue

	changes <- svc.Status{State: svc.StartPending}

	// Create context for daemon
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Start daemon
	if err := d.Start(ctx); err != nil {
		d.logger.Error("Failed to start daemon", zap.Error(err))
		return false, 1
	}

	changes <- svc.Status{State: svc.Running, Accepts: cmdsAccepted}

	// Handle service control requests
	for {
		select {
		case c := <-r:
			switch c.Cmd {
			case svc.Interrogate:
				d.mu.RLock()
				changes <- d.status
				d.mu.RUnlock()
			case svc.Stop, svc.Shutdown:
				changes <- svc.Status{State: svc.StopPending}
				cancel()
				return false, 0
			case svc.Pause:
				changes <- svc.Status{State: svc.Paused, Accepts: cmdsAccepted}
			case svc.Continue:
				changes <- svc.Status{State: svc.Running, Accepts: cmdsAccepted}
			default:
				d.logger.Error("Unexpected control request", zap.Int("control", int(c.Cmd)))
			}
		}
	}
}

// monitor monitors daemon health and resources
func (d *WindowsDaemon) monitor(ctx context.Context) {
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-d.stopCh:
			return
		case <-ticker.C:
			// Get Job Object statistics
			info, err := d.jobObject.GetExtendedAccountingInformation()
			if err != nil {
				d.logger.Error("Failed to get job object stats", zap.Error(err))
				continue
			}

			// Log resource usage
			d.logger.Debug("Resource usage",
				zap.Uint64("process_time", info.BasicLimitInformation.PerProcessUserTimeLimit),
				zap.Uint64("job_time", info.BasicLimitInformation.PerJobUserTimeLimit),
				zap.Uint32("active_processes", info.BasicLimitInformation.ActiveProcessLimit),
				zap.Uint64("read_bytes", info.IoInfo.ReadTransferCount),
				zap.Uint64("write_bytes", info.IoInfo.WriteTransferCount),
			)
		}
	}
}
