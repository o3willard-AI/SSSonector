package preflight

import (
	"context"
	"fmt"
	"net"
	"os"
	"runtime"
	"syscall"

	"go.uber.org/zap"
)

// CheckResult represents the result of a pre-flight check
type CheckResult struct {
	Name        string
	Status      string
	Description string
	Error       error
}

// String returns a string representation of the check result
func (r CheckResult) String() string {
	if r.Error != nil {
		return fmt.Sprintf("%s: Failed - %s: %v", r.Name, r.Description, r.Error)
	}
	return fmt.Sprintf("%s: %s - %s", r.Name, r.Status, r.Description)
}

// PreFlightCheck defines a pre-flight check function
type PreFlightCheck func(ctx context.Context, logger *zap.Logger) CheckResult

// SystemRequirements checks system requirements
func SystemRequirements(ctx context.Context, logger *zap.Logger) CheckResult {
	result := CheckResult{
		Name:        "System Requirements",
		Description: "Checking system requirements",
	}

	// Check CPU cores
	if runtime.NumCPU() < 2 {
		result.Status = "Failed"
		result.Error = fmt.Errorf("insufficient CPU cores: minimum 2 required, got %d", runtime.NumCPU())
		return result
	}

	// Check available memory
	var memInfo syscall.Sysinfo_t
	if err := syscall.Sysinfo(&memInfo); err != nil {
		result.Status = "Failed"
		result.Error = fmt.Errorf("failed to get system info: %w", err)
		return result
	}

	totalRAM := memInfo.Totalram * uint64(memInfo.Unit)
	minRAM := uint64(2 * 1024 * 1024 * 1024) // 2GB
	if totalRAM < minRAM {
		result.Status = "Failed"
		result.Error = fmt.Errorf("insufficient memory: minimum 2GB required, got %d bytes", totalRAM)
		return result
	}

	result.Status = "Passed"
	return result
}

// FilePermissions checks file permissions
func FilePermissions(ctx context.Context, logger *zap.Logger) CheckResult {
	result := CheckResult{
		Name:        "File Permissions",
		Description: "Checking file permissions",
	}

	// Check config directory permissions
	configDir := "/etc/sssonector"
	if envDir := os.Getenv("SSSONECTOR_CONFIG_DIR"); envDir != "" {
		configDir = envDir
	}

	info, err := os.Stat(configDir)
	if err != nil {
		if os.IsNotExist(err) {
			result.Status = "Failed"
			result.Error = fmt.Errorf("config directory does not exist: %s", configDir)
			return result
		}
		result.Status = "Failed"
		result.Error = fmt.Errorf("failed to check config directory: %w", err)
		return result
	}

	mode := info.Mode().Perm()
	if mode&0077 != 0 {
		result.Status = "Failed"
		result.Error = fmt.Errorf("insecure config directory permissions: %s", mode)
		return result
	}

	result.Status = "Passed"
	return result
}

// NetworkConnectivity checks network connectivity
func NetworkConnectivity(ctx context.Context, logger *zap.Logger) CheckResult {
	result := CheckResult{
		Name:        "Network Connectivity",
		Description: "Checking network connectivity",
	}

	// Check if we can bind to required ports
	ports := []int{8080, 8443} // Example ports
	for _, port := range ports {
		addr := fmt.Sprintf(":%d", port)
		listener, err := net.Listen("tcp", addr)
		if err != nil {
			result.Status = "Failed"
			result.Error = fmt.Errorf("failed to bind to port %d: %w", port, err)
			return result
		}
		listener.Close()
	}

	// Check DNS resolution
	_, err := net.LookupHost("example.com")
	if err != nil {
		result.Status = "Failed"
		result.Error = fmt.Errorf("DNS resolution failed: %w", err)
		return result
	}

	result.Status = "Passed"
	return result
}

// SecurityChecks performs security-related checks
func SecurityChecks(ctx context.Context, logger *zap.Logger) CheckResult {
	result := CheckResult{
		Name:        "Security Checks",
		Description: "Performing security checks",
	}

	// Check if running as root
	if os.Geteuid() == 0 {
		result.Status = "Failed"
		result.Error = fmt.Errorf("running as root is not allowed")
		return result
	}

	// Check umask
	oldMask := syscall.Umask(0)
	syscall.Umask(oldMask)
	if oldMask != 0022 && oldMask != 0027 {
		result.Status = "Failed"
		result.Error = fmt.Errorf("insecure umask: %04o", oldMask)
		return result
	}

	result.Status = "Passed"
	return result
}

// DependencyChecks checks external dependencies
func DependencyChecks(ctx context.Context, logger *zap.Logger) CheckResult {
	result := CheckResult{
		Name:        "Dependencies",
		Description: "Checking external dependencies",
	}

	// Check TLS certificates
	_, err := os.Stat("/etc/ssl/certs/ca-certificates.crt")
	if err != nil {
		result.Status = "Failed"
		result.Error = fmt.Errorf("CA certificates not found: %w", err)
		return result
	}

	result.Status = "Passed"
	return result
}

// RunPreFlightChecks runs all pre-flight checks
func RunPreFlightChecks(ctx context.Context, logger *zap.Logger) ([]CheckResult, error) {
	checks := []PreFlightCheck{
		SystemRequirements,
		FilePermissions,
		NetworkConnectivity,
		SecurityChecks,
		DependencyChecks,
		ConfigValidationCheck, // Added configuration validation check
	}

	results := make([]CheckResult, 0, len(checks))
	for _, check := range checks {
		select {
		case <-ctx.Done():
			return results, ctx.Err()
		default:
			result := check(ctx, logger)
			results = append(results, result)
			if result.Error != nil {
				logger.Error("Pre-flight check failed",
					zap.String("check", result.Name),
					zap.Error(result.Error))
			} else {
				logger.Info("Pre-flight check passed",
					zap.String("check", result.Name))
			}
		}
	}

	return results, nil
}
