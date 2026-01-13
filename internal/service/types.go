package service

import (
	"context"
	"time"

	"github.com/o3willard-AI/SSSonector/internal/config"
)

// ServiceState represents the state of a service
type ServiceState int

const (
	ServiceStateCreated ServiceState = iota
	ServiceStateStarting
	ServiceStateRunning
	ServiceStateStopping
	ServiceStateStopped
	ServiceStateError
	ServiceStateReloading
)

// ComponentHealth represents the health status of a component
type ComponentHealth int

const (
	ComponentHealthUnknown ComponentHealth = iota
	ComponentHealthHealthy
	ComponentHealthDegraded
	ComponentHealthUnhealthy
	ComponentHealthOffline
)

// MigrationStatus represents the status of a configuration migration
type MigrationStatus int

const (
	MigrationStatusIdle MigrationStatus = iota
	MigrationStatusInProgress
	MigrationStatusCompleted
	MigrationStatusFailed
	MigrationStatusRolledBack
)

// VersionInfo represents version information for a component
type VersionInfo struct {
	Component      string    `json:"component"`
	Version        string    `json:"version"`
	SchemaVersion  string    `json:"schema_version"`
	BuildNumber    string    `json:"build_number,omitempty"`
	CommitHash     string    `json:"commit_hash,omitempty"`
	BuildDate      time.Time `json:"build_date"`
	StartedAt      time.Time `json:"started_at"`
}

// ResourceMetrics represents resource usage metrics
type ResourceMetrics struct {
	CPUUsage        float64       `json:"cpu_usage_percent"`
	MemoryUsage     uint64        `json:"memory_usage_bytes"`
	MemoryLimit     uint64        `json:"memory_limit_bytes"`
	OpenFiles       int           `json:"open_files"`
	OpenConnections int           `json:"open_connections"`
	GoroutineCount  int           `json:"goroutine_count"`
	GCPauseTime     time.Duration `json:"gc_pause_time"`
	LastUpdate      time.Time     `json:"last_update"`
}

// SecurityMetrics represents security-related metrics
type SecurityMetrics struct {
	ActiveSessions     int64     `json:"active_sessions"`
	RevokedSessions    int64     `json:"revoked_sessions"`
	FailedAuthAttempts int64     `json:"failed_auth_attempts"`
	BlockedIPs         int64     `json:"blocked_ips"`
	CertificateExpiry  time.Time `json:"certificate_expiry"`
	LastSecurityCheck  time.Time `json:"last_security_check"`
}

// ComponentStatus represents the status of a single component
type ComponentStatus struct {
	Name          string          `json:"name"`
	State         ServiceState    `json:"state"`
	Health        ComponentHealth `json:"health"`
	Error         string          `json:"error,omitempty"`
	StartedAt     time.Time       `json:"started_at"`
	LastCheck     time.Time       `json:"last_check"`
	ResourceUsage ResourceMetrics `json:"resource_usage,omitempty"`
}

// ServiceStatus represents the overall status of the service
type ServiceStatus struct {
	OverallHealth    ComponentHealth   `json:"overall_health"`
	Version          VersionInfo       `json:"version"`
	Components       []ComponentStatus `json:"components"`
	MigrationStatus  MigrationStatus   `json:"migration_status,omitempty"`
	MigrationMessage string            `json:"migration_message,omitempty"`
	ResourceMetrics  ResourceMetrics   `json:"resource_metrics"`
	SecurityMetrics  SecurityMetrics   `json:"security_metrics"`
	Uptime           time.Duration     `json:"uptime"`
	CurrentConfig    *config.AppConfig `json:"current_config,omitempty"`
}

// LifecycleHook is a function type for lifecycle hooks
type LifecycleHook func(ctx context.Context) error

// ServiceConfig represents service configuration
type ServiceConfig struct {
	Name          string
	Version       string
	SchemaVersion string
	BuildNumber   string
	CommitHash    string
	BuildDate     time.Time
}

// TypedError represents a typed error with context
type TypedError struct {
	Type    string `json:"type"`
	Message string `json:"message"`
	Code    int    `json:"code"`
	Cause   error  `json:"cause,omitempty"`
}

func (e *TypedError) Error() string {
	return e.Message
}

// Common error types
var (
	ErrServiceNotRunning     = &TypedError{Type: "service_error", Message: "service is not running", Code: 1001}
	ErrServiceAlreadyRunning  = &TypedError{Type: "service_error", Message: "service is already running", Code: 1002}
	ErrConfigInvalid         = &TypedError{Type: "config_error", Message: "configuration is invalid", Code: 2001}
	ErrConfigVersionMismatch = &TypedError{Type: "config_error", Message: "configuration version mismatch", Code: 2002}
	ErrMigrationInProgress   = &TypedError{Type: "migration_error", Message: "migration is in progress", Code: 3001}
	ErrMigrationFailed       = &TypedError{Type: "migration_error", Message: "migration failed", Code: 3002}
	ErrComponentNotFound     = &TypedError{Type: "component_error", Message: "component not found", Code: 4001}
	ErrHealthCheckFailed     = &TypedError{Type: "health_error", Message: "health check failed", Code: 5001}
)

// Service interface defines the core service operations
type Service interface {
	// Lifecycle methods
	Start(ctx context.Context) error
	Stop(ctx context.Context) error
	Reload(ctx context.Context) error

	// Status methods
	Status(ctx context.Context) (*ServiceStatus, error)
	Health(ctx context.Context) ComponentHealth

	// Version methods
	GetVersion() VersionInfo
	GetSchemaVersion() string

	// Migration methods
	StartMigration(ctx context.Context) error
	AbortMigration(ctx context.Context) error
	GetMigrationStatus() (MigrationStatus, string)

	// Component management
	RegisterComponent(name string, component Service) error
	UnregisterComponent(name string) error
	GetComponent(name string) (Service, bool)
}
</parameter>
<task_progress>
- [x] Clone and analyze SSSonector repository
- [x] Review existing Issues.md and codebase
- [x] Identify improvement areas
- [x] Create prioritized atomic task list (96 tasks)
- [x] Create sprint/iteration breakdown (11 sprints, 22 weeks)
- [x] Implement Sprint 1 tasks (100% complete - 16/16 tasks)
- [x] Implement Sprint 2 tasks (100% complete - 10/10 tasks)
- [ ] Implement Sprint 3 tasks (SERVICE-001 through SERVICE-010)
- [ ] Implement Sprint 4-5 tasks (PERF-001 through PERF-015)
- [ ] Implement Sprint 6-7 tasks (REL-001 through REL-020)
- [ ] Implement Sprint 8-9 tasks (requires Act Mode)
- [ ] Implement Sprint 10-11 tasks (requires Act Mode)
</task_progress>
</write_to_file>
