package errors

import (
	"fmt"
	"time"
)

// Severity represents the severity level of an error
type Severity int

const (
	SeverityFatal Severity = iota
	SeverityError
	SeverityWarning
	SeverityInfo
)

func (s Severity) String() string {
	switch s {
	case SeverityFatal:
		return "FATAL"
	case SeverityError:
		return "ERROR"
	case SeverityWarning:
		return "WARNING"
	case SeverityInfo:
		return "INFO"
	default:
		return "UNKNOWN"
	}
}

// Category represents the broad category of an error
type Category int

const (
	CategoryConfig Category = iota
	CategorySystem
	CategoryResource
	CategoryPermission
	CategoryNetwork
	CategorySecurity
)

func (c Category) String() string {
	switch c {
	case CategoryConfig:
		return "Configuration"
	case CategorySystem:
		return "System"
	case CategoryResource:
		return "Resource"
	case CategoryPermission:
		return "Permission"
	case CategoryNetwork:
		return "Network"
	case CategorySecurity:
		return "Security"
	default:
		return "Unknown"
	}
}

// ErrorCode represents specific error conditions
type ErrorCode int

const (
	// Configuration Errors (1000-1999)
	ErrConfigSyntax     ErrorCode = 1000
	ErrConfigSchema     ErrorCode = 1001
	ErrConfigDependency ErrorCode = 1002
	ErrConfigPermission ErrorCode = 1003
	ErrConfigEnvVar     ErrorCode = 1004

	// System Errors (2000-2999)
	ErrSystemCapability ErrorCode = 2000
	ErrSystemGroup      ErrorCode = 2001
	ErrSystemResource   ErrorCode = 2002
	ErrSystemState      ErrorCode = 2003

	// Resource Errors (3000-3999)
	ErrResourceBusy       ErrorCode = 3000
	ErrResourceMissing    ErrorCode = 3001
	ErrResourcePermission ErrorCode = 3002
	ErrResourceExhausted  ErrorCode = 3003

	// Network Errors (4000-4999)
	ErrNetworkPort      ErrorCode = 4000
	ErrNetworkInterface ErrorCode = 4001
	ErrNetworkProtocol  ErrorCode = 4002

	// Security Errors (5000-5999)
	ErrSecurityCert       ErrorCode = 5000
	ErrSecurityKey        ErrorCode = 5001
	ErrSecurityPermission ErrorCode = 5002
)

// Location represents the source location where an error occurred
type Location struct {
	File     string
	Line     int
	Function string
	Stack    []string
}

// Context provides rich context about an error
type Context struct {
	Code       ErrorCode
	Category   Category
	Severity   Severity
	Message    string
	Details    map[string]interface{}
	Location   *Location
	Suggestion string
	Resolution []string
	Reference  string
	Timestamp  time.Time
}

// Error implements the error interface
func (c *Context) Error() string {
	return fmt.Sprintf("[%s] %s: %s", c.Severity, c.Category, c.Message)
}

// WithDetails adds details to the error context
func (c *Context) WithDetails(details map[string]interface{}) *Context {
	if c.Details == nil {
		c.Details = make(map[string]interface{})
	}
	for k, v := range details {
		c.Details[k] = v
	}
	return c
}

// WithSuggestion adds a suggestion to the error context
func (c *Context) WithSuggestion(suggestion string) *Context {
	c.Suggestion = suggestion
	return c
}

// WithResolution adds resolution steps to the error context
func (c *Context) WithResolution(steps ...string) *Context {
	c.Resolution = append(c.Resolution, steps...)
	return c
}

// WithReference adds a documentation reference to the error context
func (c *Context) WithReference(ref string) *Context {
	c.Reference = ref
	return c
}

// NewError creates a new error context
func NewError(code ErrorCode, category Category, severity Severity, message string) *Context {
	return &Context{
		Code:      code,
		Category:  category,
		Severity:  severity,
		Message:   message,
		Timestamp: time.Now(),
	}
}
