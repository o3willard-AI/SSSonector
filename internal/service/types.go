package service

import "time"

// Service states
const (
	StateStarting  = "starting"
	StateRunning   = "running"
	StateStopping  = "stopping"
	StateStopped   = "stopped"
	StateReloading = "reloading"
	StateFailed    = "failed"
)

// Service commands
const (
	CmdStart        = "start"
	CmdStop         = "stop"
	CmdReload       = "reload"
	CmdStatus       = "status"
	CmdMetrics      = "metrics"
	CmdHealth       = "health"
	CmdUpdateConfig = "update-config"
	CmdRotateCerts  = "rotate-certs"
)

// Service error codes
const (
	ErrNotFound       = 404
	ErrUnauthorized   = 401
	ErrForbidden      = 403
	ErrInvalidInput   = 400
	ErrInternal       = 500
	ErrNotSupported   = 501
	ErrUnavailable    = 503
	ErrTimeout        = 504
	ErrAlreadyExists  = 409
	ErrAlreadyRunning = 400
	ErrNotRunning     = 400
	ErrInvalidCommand = 400
	ErrNotImplemented = 501
)

// ServiceOptions represents service configuration options
type ServiceOptions struct {
	Name        string   // Service name
	ConfigPath  string   // Path to config file
	PIDFile     string   // Path to PID file
	LogFile     string   // Path to log file
	User        string   // User to run as
	Group       string   // Group to run as
	WorkingDir  string   // Working directory
	Environment []string // Environment variables
}

// ServiceStatus represents service status information
type ServiceStatus struct {
	Name        string    // Service name
	State       string    // Current state
	PID         int       // Process ID
	StartTime   time.Time // Start time
	LastReload  time.Time // Last reload time
	Error       string    // Last error message
	Uptime      string    // Uptime duration
	Memory      uint64    // Memory usage in bytes
	CPU         float64   // CPU usage percentage
	Restarts    int       // Number of restarts
	Connections int       // Number of active connections
	Platform    string    // Platform (linux, windows, etc)
}

// ServiceMetrics represents service metrics
type ServiceMetrics struct {
	StartTime       time.Time // Service start time
	Platform        string    // Platform (linux, windows, etc)
	LastError       string    // Last error message
	CPUUsage        float64   // CPU usage percentage
	MemoryUsage     uint64    // Memory usage in bytes
	UptimeSeconds   int64     // Uptime in seconds
	LastReload      time.Time // Last reload time
	ConnectionCount int       // Number of active connections
	BytesReceived   uint64    // Total bytes received
	BytesSent       uint64    // Total bytes sent
	ErrorCount      int       // Total error count
}

// ServiceCommand represents a command to be executed by the service
type ServiceCommand struct {
	Command   string // Command to execute
	Type      string // Command type
	RequestID string // Request ID for tracking
	Data      []byte // Optional command data
	Payload   []byte // Optional payload data
}

// ServiceError represents a service error
type ServiceError struct {
	Code    int    // Error code
	Message string // Error message
	Details string // Error details
}

// Error returns the error message
func (e *ServiceError) Error() string {
	return e.Message
}

// NewServiceError creates a new service error
func NewServiceError(code int, message string, details string) *ServiceError {
	return &ServiceError{
		Code:    code,
		Message: message,
		Details: details,
	}
}

// Service interface defines the methods that must be implemented by a service
type Service interface {
	// Start starts the service
	Start() error

	// Stop stops the service
	Stop() error

	// Reload reloads the service configuration
	Reload() error

	// Status returns the current service status
	Status() (string, error)

	// GetPID returns the service process ID
	GetPID() (int, error)

	// IsRunning returns whether the service is running
	IsRunning() (bool, error)

	// GetMetrics returns service metrics
	GetMetrics() (ServiceMetrics, error)

	// Health checks service health
	Health() error

	// ExecuteCommand executes a service command
	ExecuteCommand(cmd ServiceCommand) (string, error)

	// GetLogs returns service logs
	GetLogs(lines int) ([]string, error)

	// SendSignal sends a signal to the service
	SendSignal(signal string) error

	// Configure configures the service
	Configure(opts ServiceOptions) error

	// GetOptions returns service options
	GetOptions() ServiceOptions
}
