package service

import "time"

// ServiceState represents the state of a service
type ServiceState string

const (
	// StateStopped indicates the service is stopped
	StateStopped ServiceState = "stopped"
	// StateStarting indicates the service is starting
	StateStarting ServiceState = "starting"
	// StateRunning indicates the service is running
	StateRunning ServiceState = "running"
	// StateStopping indicates the service is stopping
	StateStopping ServiceState = "stopping"
	// StateReloading indicates the service is reloading
	StateReloading ServiceState = "reloading"
)

// ServiceCommand represents a command that can be executed on a service
type ServiceCommand string

const (
	// CmdStart starts the service
	CmdStart ServiceCommand = "start"
	// CmdStop stops the service
	CmdStop ServiceCommand = "stop"
	// CmdStatus returns the current service status
	CmdStatus ServiceCommand = "status"
	// CmdReload reloads the service configuration
	CmdReload ServiceCommand = "reload"
	// CmdMetrics returns service metrics
	CmdMetrics ServiceCommand = "metrics"
	// CmdHealth performs a health check
	CmdHealth ServiceCommand = "health"
)

// ServiceStatus represents the current status of a service
type ServiceStatus struct {
	Name       string       `json:"name"`
	State      ServiceState `json:"state"`
	Mode       string       `json:"mode"`
	Version    string       `json:"version"`
	PID        int          `json:"pid"`
	StartTime  time.Time    `json:"start_time"`
	LastReload time.Time    `json:"last_reload,omitempty"`
}

// ServiceMetrics represents service metrics
type ServiceMetrics struct {
	Platform      string `json:"platform"`
	UptimeSeconds int64  `json:"uptime_seconds"`
}

// ServiceResponse represents a response from a service command
type ServiceResponse struct {
	Success bool        `json:"success"`
	Message string      `json:"message,omitempty"`
	Data    interface{} `json:"data,omitempty"`
}

// ServiceOptions represents service options
type ServiceOptions struct {
	Name      string
	ConfigDir string
	DataDir   string
	LogDir    string
}

// Service defines the interface for service operations
type Service interface {
	// Start starts the service
	Start() error
	// Stop stops the service
	Stop() error
	// Reload reloads the service configuration
	Reload() error
	// Status returns the current service status
	Status() (*ServiceStatus, error)
	// Metrics returns service metrics
	Metrics() (*ServiceMetrics, error)
	// Health performs a health check
	Health() error
	// ExecuteCommand executes a service command
	ExecuteCommand(cmd ServiceCommand, args map[string]interface{}) (*ServiceResponse, error)
}

// ServiceError represents a service error
type ServiceError struct {
	Code    ErrorCode
	Message string
}

// ErrorCode represents a service error code
type ErrorCode int

const (
	// ErrUnknown represents an unknown error
	ErrUnknown ErrorCode = iota
	// ErrNotFound represents a not found error
	ErrNotFound
	// ErrInvalidCommand represents an invalid command error
	ErrInvalidCommand
	// ErrAlreadyRunning represents an already running error
	ErrAlreadyRunning
	// ErrNotRunning represents a not running error
	ErrNotRunning
)

// Error returns the error message
func (e *ServiceError) Error() string {
	return e.Message
}

// NewServiceError creates a new service error
func NewServiceError(code ErrorCode, message string) *ServiceError {
	return &ServiceError{
		Code:    code,
		Message: message,
	}
}
