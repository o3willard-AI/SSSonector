package service

import "time"

// ServiceCommand represents a service control command
type ServiceCommand string

const (
	// Service commands
	CmdStatus  ServiceCommand = "status"
	CmdMetrics ServiceCommand = "metrics"
	CmdHealth  ServiceCommand = "health"
	CmdStart   ServiceCommand = "start"
	CmdStop    ServiceCommand = "stop"
	CmdReload  ServiceCommand = "reload"
)

// ServiceState represents a service state
type ServiceState string

const (
	// Service states
	StateStarting  ServiceState = "starting"
	StateRunning   ServiceState = "running"
	StateStopping  ServiceState = "stopping"
	StateStopped   ServiceState = "stopped"
	StateReloading ServiceState = "reloading"
	StateFailed    ServiceState = "failed"
)

// ServiceError represents a service error
type ServiceError struct {
	Code    ErrorCode
	Message string
}

// ErrorCode represents a service error code
type ErrorCode string

const (
	// Error codes
	ErrNotRunning     ErrorCode = "not_running"
	ErrAlreadyRunning ErrorCode = "already_running"
	ErrInvalidCommand ErrorCode = "invalid_command"
	ErrInvalidConfig  ErrorCode = "invalid_config"
	ErrInternal       ErrorCode = "internal_error"
)

// NewServiceError creates a new service error
func NewServiceError(code ErrorCode, message string) error {
	return &ServiceError{
		Code:    code,
		Message: message,
	}
}

// Error implements the error interface
func (e *ServiceError) Error() string {
	return e.Message
}

// ServiceStatus represents service status information
type ServiceStatus struct {
	Name       string       // Service name
	State      ServiceState // Current state
	Mode       string       // Operating mode
	Version    string       // Service version
	StartTime  time.Time    // Service start time
	LastReload time.Time    // Last config reload time
	PID        int          // Process ID
}

// ServiceMetrics represents service metrics
type ServiceMetrics struct {
	Platform      string  // Operating system platform
	UptimeSeconds int64   // Uptime in seconds
	CPUPercent    float64 // CPU usage percentage
	MemoryBytes   uint64  // Memory usage in bytes
	OpenFiles     int     // Number of open files
	Connections   int     // Number of active connections
	BytesIn       uint64  // Total bytes received
	BytesOut      uint64  // Total bytes sent
	ErrorCount    int     // Total error count
}

// ServiceResponse represents a service command response
type ServiceResponse struct {
	Success bool        // Success flag
	Message string      // Response message
	Data    interface{} // Response data
}

// ServiceOptions represents service configuration options
type ServiceOptions struct {
	Name      string // Service name
	ConfigDir string // Configuration directory
	DataDir   string // Data directory
	LogDir    string // Log directory
}

// Service defines the interface for service operations
type Service interface {
	// Core operations
	Start() error
	Stop() error
	Reload() error

	// Status and monitoring
	Status() (*ServiceStatus, error)
	Metrics() (*ServiceMetrics, error)
	Health() error
}
