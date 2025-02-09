package platform

import (
	"fmt"
	"os"
	"runtime"
)

// Platform represents the current operating system and its capabilities
type Platform struct {
	OS          string   // Operating system (linux, windows, darwin)
	Init        string   // Init system (systemd, windows, launchd)
	Features    []string // Supported features
	Constraints []string // Platform limitations
}

// Feature constants for platform capabilities
const (
	FeatureNamedPipes  = "named_pipes"
	FeatureUnixSockets = "unix_sockets"
	FeatureSystemd     = "systemd"
	FeatureLaunchd     = "launchd"
	FeatureWinService  = "windows_service"
	FeatureSetUID      = "setuid"
	FeaturePIDFile     = "pid_file"
	FeatureSyslog      = "syslog"
	FeatureEventLog    = "eventlog"
	FeatureChroot      = "chroot"
	FeatureNamespaces  = "namespaces"
	FeatureCgroups     = "cgroups"
)

// Constraint constants for platform limitations
const (
	ConstraintNoFork        = "no_fork"
	ConstraintNoSetUID      = "no_setuid"
	ConstraintNoNamespaces  = "no_namespaces"
	ConstraintNoCgroups     = "no_cgroups"
	ConstraintNoUnixSockets = "no_unix_sockets"
)

// DetectPlatform identifies the current platform and its capabilities
func DetectPlatform() (*Platform, error) {
	p := &Platform{
		OS:          runtime.GOOS,
		Features:    make([]string, 0),
		Constraints: make([]string, 0),
	}

	// Detect init system
	switch p.OS {
	case "linux":
		if _, err := os.Stat("/run/systemd/system"); err == nil {
			p.Init = "systemd"
			p.Features = append(p.Features, FeatureSystemd)
		} else {
			p.Init = "other"
		}
		// Add Linux-specific features
		p.Features = append(p.Features,
			FeatureUnixSockets,
			FeatureSetUID,
			FeaturePIDFile,
			FeatureSyslog,
			FeatureChroot)

		// Check for additional Linux capabilities
		if _, err := os.Stat("/proc/self/ns"); err == nil {
			p.Features = append(p.Features, FeatureNamespaces)
		}
		if _, err := os.Stat("/sys/fs/cgroup"); err == nil {
			p.Features = append(p.Features, FeatureCgroups)
		}

	case "windows":
		p.Init = "windows"
		p.Features = append(p.Features,
			FeatureNamedPipes,
			FeatureWinService,
			FeatureEventLog)
		p.Constraints = append(p.Constraints,
			ConstraintNoFork,
			ConstraintNoSetUID,
			ConstraintNoNamespaces,
			ConstraintNoCgroups,
			ConstraintNoUnixSockets)

	case "darwin":
		p.Init = "launchd"
		p.Features = append(p.Features,
			FeatureLaunchd,
			FeatureUnixSockets,
			FeaturePIDFile,
			FeatureSyslog)
		p.Constraints = append(p.Constraints,
			ConstraintNoNamespaces,
			ConstraintNoCgroups)

	default:
		return nil, fmt.Errorf("unsupported platform: %s", p.OS)
	}

	return p, nil
}

// HasFeature checks if the platform supports a specific feature
func (p *Platform) HasFeature(feature string) bool {
	for _, f := range p.Features {
		if f == feature {
			return true
		}
	}
	return false
}

// HasConstraint checks if the platform has a specific constraint
func (p *Platform) HasConstraint(constraint string) bool {
	for _, c := range p.Constraints {
		if c == constraint {
			return true
		}
	}
	return false
}

// ValidateFeatures checks if all required features are supported
func (p *Platform) ValidateFeatures(required []string) error {
	missing := make([]string, 0)
	for _, req := range required {
		if !p.HasFeature(req) {
			missing = append(missing, req)
		}
	}
	if len(missing) > 0 {
		return fmt.Errorf("platform missing required features: %v", missing)
	}
	return nil
}

// String returns a human-readable description of the platform
func (p *Platform) String() string {
	return fmt.Sprintf("Platform{OS: %s, Init: %s, Features: %v, Constraints: %v}",
		p.OS, p.Init, p.Features, p.Constraints)
}

// IsPrivileged checks if the current process has elevated privileges
func IsPrivileged() bool {
	if runtime.GOOS == "windows" {
		// Windows privilege check would go here
		// For now, return false as it requires Win32 API calls
		return false
	}
	return os.Geteuid() == 0
}

// RequirePrivileged returns an error if the process is not privileged
func RequirePrivileged() error {
	if !IsPrivileged() {
		return fmt.Errorf("operation requires elevated privileges")
	}
	return nil
}
