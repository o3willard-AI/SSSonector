package preflight

import (
	"fmt"
	"os"
	"os/exec"
	"sort"

	"github.com/o3willard-AI/SSSonector/internal/errors"
)

// Validator defines the interface for validation checks
type Validator interface {
	// Name returns the validator's name
	Name() string

	// Check performs the validation
	Check() Result

	// Fix attempts to fix any issues found
	Fix() error

	// Priority returns the validator's execution priority
	Priority() int
}

// BaseValidator provides common validator functionality
type BaseValidator struct {
	name     string
	priority int
}

func (v *BaseValidator) Name() string {
	return v.name
}

func (v *BaseValidator) Priority() int {
	return v.priority
}

// registerValidators adds all validators to the checker
func (c *Checker) registerValidators() {
	// Add validators in priority order
	validators := []Validator{
		NewConfigValidator(c.opts, c.errorHandler),
		NewSystemValidator(c.opts, c.errorHandler),
		NewResourceValidator(c.opts, c.errorHandler),
		NewNetworkValidator(c.opts, c.errorHandler),
		NewSecurityValidator(c.opts, c.errorHandler),
	}

	// Sort by priority
	sort.Slice(validators, func(i, j int) bool {
		return validators[i].Priority() < validators[j].Priority()
	})

	c.validators = validators
}

// ConfigValidator validates configuration
type ConfigValidator struct {
	BaseValidator
	opts         *Options
	errorHandler *errors.Handler
}

func NewConfigValidator(opts *Options, errorHandler *errors.Handler) *ConfigValidator {
	return &ConfigValidator{
		BaseValidator: BaseValidator{
			name:     "Configuration",
			priority: 0,
		},
		opts:         opts,
		errorHandler: errorHandler,
	}
}

func (v *ConfigValidator) Check() Result {
	// Check if config file exists
	if _, err := os.Stat(v.opts.ConfigPath); os.IsNotExist(err) {
		return Result{
			Name:        v.Name(),
			Status:      StatusFail,
			Details:     fmt.Sprintf("Configuration file not found: %s", v.opts.ConfigPath),
			Error:       err,
			FixPossible: false,
		}
	}

	// Check file permissions
	info, err := os.Stat(v.opts.ConfigPath)
	if err != nil {
		return Result{
			Name:        v.Name(),
			Status:      StatusFail,
			Details:     "Failed to check file permissions",
			Error:       err,
			FixPossible: true,
			FixCommand:  fmt.Sprintf("sudo chmod 640 %s && sudo chown root:sssonector %s", v.opts.ConfigPath, v.opts.ConfigPath),
		}
	}

	if info.Mode().Perm() != 0640 {
		return Result{
			Name:        v.Name(),
			Status:      StatusFail,
			Details:     fmt.Sprintf("Invalid file permissions: %v (expected: 640)", info.Mode().Perm()),
			FixPossible: true,
			FixCommand:  fmt.Sprintf("sudo chmod 640 %s", v.opts.ConfigPath),
		}
	}

	return Result{
		Name:   v.Name(),
		Status: StatusPass,
	}
}

func (v *ConfigValidator) Fix() error {
	cmd := exec.Command("sudo", "chmod", "640", v.opts.ConfigPath)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to fix permissions: %v", err)
	}
	return nil
}

// SystemValidator validates system requirements
type SystemValidator struct {
	BaseValidator
	opts         *Options
	errorHandler *errors.Handler
}

func NewSystemValidator(opts *Options, errorHandler *errors.Handler) *SystemValidator {
	return &SystemValidator{
		BaseValidator: BaseValidator{
			name:     "System",
			priority: 1,
		},
		opts:         opts,
		errorHandler: errorHandler,
	}
}

func (v *SystemValidator) Check() Result {
	// Check capabilities
	cmd := exec.Command("getcap", "/usr/local/bin/sssonector")
	output, err := cmd.CombinedOutput()
	if err != nil || !containsCapabilities(string(output)) {
		return Result{
			Name:        v.Name(),
			Status:      StatusFail,
			Details:     "Missing required capabilities",
			FixPossible: true,
			FixCommand:  "sudo setcap cap_net_admin,cap_net_raw+ep /usr/local/bin/sssonector",
		}
	}

	return Result{
		Name:   v.Name(),
		Status: StatusPass,
	}
}

func (v *SystemValidator) Fix() error {
	cmd := exec.Command("sudo", "setcap", "cap_net_admin,cap_net_raw+ep", "/usr/local/bin/sssonector")
	return cmd.Run()
}

// ResourceValidator validates resource availability
type ResourceValidator struct {
	BaseValidator
	opts         *Options
	errorHandler *errors.Handler
}

func NewResourceValidator(opts *Options, errorHandler *errors.Handler) *ResourceValidator {
	return &ResourceValidator{
		BaseValidator: BaseValidator{
			name:     "Resources",
			priority: 2,
		},
		opts:         opts,
		errorHandler: errorHandler,
	}
}

func (v *ResourceValidator) Check() Result {
	// Check TUN device
	if _, err := os.Stat("/dev/net/tun"); os.IsNotExist(err) {
		return Result{
			Name:        v.Name(),
			Status:      StatusFail,
			Details:     "TUN device not found",
			Error:       err,
			FixPossible: false,
		}
	}

	// Check TUN permissions
	info, err := os.Stat("/dev/net/tun")
	if err != nil {
		return Result{
			Name:        v.Name(),
			Status:      StatusFail,
			Details:     "Failed to check TUN permissions",
			Error:       err,
			FixPossible: true,
			FixCommand:  "sudo chmod 660 /dev/net/tun && sudo chown root:sssonector /dev/net/tun",
		}
	}

	if info.Mode().Perm() != 0660 {
		return Result{
			Name:        v.Name(),
			Status:      StatusFail,
			Details:     fmt.Sprintf("Invalid TUN permissions: %v (expected: 660)", info.Mode().Perm()),
			FixPossible: true,
			FixCommand:  "sudo chmod 660 /dev/net/tun",
		}
	}

	return Result{
		Name:   v.Name(),
		Status: StatusPass,
	}
}

func (v *ResourceValidator) Fix() error {
	cmd := exec.Command("sudo", "chmod", "660", "/dev/net/tun")
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to fix TUN permissions: %v", err)
	}
	return nil
}

// NetworkValidator validates network configuration
type NetworkValidator struct {
	BaseValidator
	opts         *Options
	errorHandler *errors.Handler
}

func NewNetworkValidator(opts *Options, errorHandler *errors.Handler) *NetworkValidator {
	return &NetworkValidator{
		BaseValidator: BaseValidator{
			name:     "Network",
			priority: 3,
		},
		opts:         opts,
		errorHandler: errorHandler,
	}
}

func (v *NetworkValidator) Check() Result {
	// Check port availability
	cmd := exec.Command("lsof", "-i", ":8443")
	if err := cmd.Run(); err == nil {
		return Result{
			Name:        v.Name(),
			Status:      StatusFail,
			Details:     "Port 8443 is already in use",
			FixPossible: true,
			FixCommand:  "sudo fuser -k 8443/tcp",
		}
	}

	return Result{
		Name:   v.Name(),
		Status: StatusPass,
	}
}

func (v *NetworkValidator) Fix() error {
	cmd := exec.Command("sudo", "fuser", "-k", "8443/tcp")
	return cmd.Run()
}

// SecurityValidator validates security requirements
type SecurityValidator struct {
	BaseValidator
	opts         *Options
	errorHandler *errors.Handler
}

func NewSecurityValidator(opts *Options, errorHandler *errors.Handler) *SecurityValidator {
	return &SecurityValidator{
		BaseValidator: BaseValidator{
			name:     "Security",
			priority: 4,
		},
		opts:         opts,
		errorHandler: errorHandler,
	}
}

func (v *SecurityValidator) Check() Result {
	// Check certificate permissions
	certDir := "/etc/sssonector/certs"
	if _, err := os.Stat(certDir); os.IsNotExist(err) {
		return Result{
			Name:        v.Name(),
			Status:      StatusFail,
			Details:     "Certificate directory not found",
			Error:       err,
			FixPossible: false,
		}
	}

	// Check key file permissions
	keyFiles, err := os.ReadDir(certDir)
	if err != nil {
		return Result{
			Name:        v.Name(),
			Status:      StatusFail,
			Details:     "Failed to read certificate directory",
			Error:       err,
			FixPossible: false,
		}
	}

	for _, file := range keyFiles {
		if file.Type().IsRegular() && file.Name()[len(file.Name())-4:] == ".key" {
			info, err := file.Info()
			if err != nil {
				continue
			}
			if info.Mode().Perm() != 0600 {
				return Result{
					Name:        v.Name(),
					Status:      StatusFail,
					Details:     fmt.Sprintf("Invalid key file permissions: %v (expected: 600)", info.Mode().Perm()),
					FixPossible: true,
					FixCommand:  fmt.Sprintf("sudo chmod 600 %s/*.key", certDir),
				}
			}
		}
	}

	return Result{
		Name:   v.Name(),
		Status: StatusPass,
	}
}

func (v *SecurityValidator) Fix() error {
	cmd := exec.Command("sudo", "chmod", "600", "/etc/sssonector/certs/*.key")
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to fix key file permissions: %v", err)
	}
	return nil
}

func containsCapabilities(output string) bool {
	return len(output) > 0 && (output[len(output)-3:] == "+ep" || output[len(output)-4:] == "+eip")
}
