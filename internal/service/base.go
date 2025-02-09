package service

import (
	"fmt"
	"os"
	"runtime"
	"time"

	"go.uber.org/zap"
)

// BaseDaemon provides base daemon functionality
type BaseDaemon struct {
	logger  *zap.Logger
	options ServiceOptions
	status  ServiceStatus
	metrics ServiceMetrics
}

// NewBaseDaemon creates a new base daemon instance
func NewBaseDaemon(opts ServiceOptions, logger *zap.Logger) (*BaseDaemon, error) {
	if opts.Name == "" {
		return nil, fmt.Errorf("service name is required")
	}

	if logger == nil {
		return nil, fmt.Errorf("logger is required")
	}

	d := &BaseDaemon{
		logger:  logger,
		options: opts,
		status: ServiceStatus{
			Name:      opts.Name,
			State:     StateStopped,
			StartTime: time.Now(),
		},
		metrics: ServiceMetrics{
			StartTime: time.Now(),
			Platform:  runtime.GOOS,
		},
	}

	return d, nil
}

// Start initializes the base daemon
func (d *BaseDaemon) Start() error {
	if d.status.State == StateRunning {
		return NewServiceError(ErrAlreadyRunning, "daemon already running", "")
	}

	d.status.State = StateStarting
	d.status.StartTime = time.Now()
	d.status.PID = os.Getpid()

	d.logger.Info("Starting daemon",
		zap.String("name", d.options.Name),
		zap.Int("pid", d.status.PID))

	d.status.State = StateRunning
	return nil
}

// Stop cleans up base daemon resources
func (d *BaseDaemon) Stop() error {
	if d.status.State != StateRunning {
		return NewServiceError(ErrNotRunning, "daemon not running", "")
	}

	d.status.State = StateStopping

	d.logger.Info("Stopping daemon",
		zap.String("name", d.options.Name))

	d.status.State = StateStopped
	return nil
}

// Reload reloads the daemon configuration
func (d *BaseDaemon) Reload() error {
	if d.status.State != StateRunning {
		return NewServiceError(ErrNotRunning, "daemon not running", "")
	}

	d.status.State = StateReloading
	d.status.LastReload = time.Now()
	d.metrics.LastReload = time.Now()

	d.logger.Info("Reloading daemon",
		zap.String("name", d.options.Name))

	d.status.State = StateRunning
	return nil
}

// Status returns the current daemon status
func (d *BaseDaemon) Status() (string, error) {
	if d.status.State == "" {
		return "", NewServiceError(ErrNotRunning, "daemon not initialized", "")
	}
	return d.status.State, nil
}

// GetPID returns the daemon process ID
func (d *BaseDaemon) GetPID() (int, error) {
	if d.status.State != StateRunning {
		return 0, NewServiceError(ErrNotRunning, "daemon not running", "")
	}
	return d.status.PID, nil
}

// IsRunning returns whether the daemon is running
func (d *BaseDaemon) IsRunning() (bool, error) {
	return d.status.State == StateRunning, nil
}

// GetMetrics returns daemon metrics
func (d *BaseDaemon) GetMetrics() (ServiceMetrics, error) {
	if d.status.State != StateRunning {
		return ServiceMetrics{}, NewServiceError(ErrNotRunning, "daemon not running", "")
	}

	d.metrics.UptimeSeconds = int64(time.Since(d.status.StartTime).Seconds())
	return d.metrics, nil
}

// Health checks daemon health
func (d *BaseDaemon) Health() error {
	if d.status.State != StateRunning {
		return NewServiceError(ErrNotRunning, "daemon not running", "")
	}
	return nil
}

// ExecuteCommand executes a daemon command
func (d *BaseDaemon) ExecuteCommand(cmd ServiceCommand) (string, error) {
	if d.status.State != StateRunning {
		return "", NewServiceError(ErrNotRunning, "daemon not running", "")
	}

	switch cmd.Command {
	case CmdStatus:
		return d.Status()
	case CmdReload:
		return "", d.Reload()
	case CmdMetrics:
		metrics, err := d.GetMetrics()
		if err != nil {
			return "", err
		}
		return fmt.Sprintf("%+v", metrics), nil
	case CmdHealth:
		return "", d.Health()
	default:
		return "", NewServiceError(ErrInvalidCommand, fmt.Sprintf("unknown command: %s", cmd.Command), "")
	}
}

// GetLogs returns daemon logs
func (d *BaseDaemon) GetLogs(lines int) ([]string, error) {
	if d.status.State != StateRunning {
		return nil, NewServiceError(ErrNotRunning, "daemon not running", "")
	}
	return nil, NewServiceError(ErrNotImplemented, "GetLogs not implemented", "")
}

// SendSignal sends a signal to the daemon
func (d *BaseDaemon) SendSignal(signal string) error {
	if d.status.State != StateRunning {
		return NewServiceError(ErrNotRunning, "daemon not running", "")
	}
	return NewServiceError(ErrNotImplemented, "SendSignal not implemented", "")
}

// Configure configures the daemon
func (d *BaseDaemon) Configure(opts ServiceOptions) error {
	d.options = opts
	return nil
}

// GetOptions returns daemon options
func (d *BaseDaemon) GetOptions() ServiceOptions {
	return d.options
}

// GetLogger returns the daemon logger
func (d *BaseDaemon) GetLogger() *zap.Logger {
	return d.logger
}

// SetLogger sets the daemon logger
func (d *BaseDaemon) SetLogger(logger *zap.Logger) {
	d.logger = logger
}

// GetStatus returns the daemon status
func (d *BaseDaemon) GetStatus() ServiceStatus {
	return d.status
}

// SetStatus sets the daemon status
func (d *BaseDaemon) SetStatus(status ServiceStatus) {
	d.status = status
}

// GetMetricsInternal returns internal metrics
func (d *BaseDaemon) GetMetricsInternal() ServiceMetrics {
	return d.metrics
}

// SetMetricsInternal sets internal metrics
func (d *BaseDaemon) SetMetricsInternal(metrics ServiceMetrics) {
	d.metrics = metrics
}
