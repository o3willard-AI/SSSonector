package daemon

import (
	"fmt"

	"github.com/o3willard-AI/SSSonector/internal/service"
	"go.uber.org/zap"
)

// Daemon provides base daemon functionality
type Daemon struct {
	logger  *zap.Logger
	options service.ServiceOptions
}

// New creates a new base daemon instance
func New(opts service.ServiceOptions, logger *zap.Logger) (*Daemon, error) {
	if opts.Name == "" {
		return nil, fmt.Errorf("service name is required")
	}

	if logger == nil {
		return nil, fmt.Errorf("logger is required")
	}

	return &Daemon{
		logger:  logger,
		options: opts,
	}, nil
}

// Start initializes the base daemon
func (d *Daemon) Start() error {
	d.logger.Info("Starting daemon",
		zap.String("name", d.options.Name))
	return nil
}

// Stop cleans up base daemon resources
func (d *Daemon) Stop() error {
	d.logger.Info("Stopping daemon",
		zap.String("name", d.options.Name))
	return nil
}

// GetLogger returns the daemon logger
func (d *Daemon) GetLogger() *zap.Logger {
	return d.logger
}

// GetOptions returns daemon options
func (d *Daemon) GetOptions() service.ServiceOptions {
	return d.options
}

// SetOptions updates daemon options
func (d *Daemon) SetOptions(opts service.ServiceOptions) {
	d.options = opts
}

// GetName returns daemon name
func (d *Daemon) GetName() string {
	return d.options.Name
}

// GetWorkingDir returns daemon working directory
func (d *Daemon) GetWorkingDir() string {
	return d.options.WorkingDir
}

// GetUser returns daemon user
func (d *Daemon) GetUser() string {
	return d.options.User
}

// GetGroup returns daemon group
func (d *Daemon) GetGroup() string {
	return d.options.Group
}

// GetEnvironment returns daemon environment variables
func (d *Daemon) GetEnvironment() []string {
	return d.options.Environment
}

// GetConfigPath returns daemon config path
func (d *Daemon) GetConfigPath() string {
	return d.options.ConfigPath
}

// GetPidFile returns daemon PID file path
func (d *Daemon) GetPidFile() string {
	return d.options.PIDFile
}

// GetLogFile returns daemon log file path
func (d *Daemon) GetLogFile() string {
	return d.options.LogFile
}
