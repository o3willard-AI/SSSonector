//go:build !windows

package daemon

import (
	"fmt"
	"os"
	"os/exec"
	"os/user"
	"path/filepath"
	"strconv"
	"syscall"

	"github.com/o3willard-AI/SSSonector/internal/service"
	"go.uber.org/zap"
)

// Daemon represents a Unix daemon process
type Daemon struct {
	options service.ServiceOptions
	logger  *zap.Logger
	pidFile string
}

// New creates a new daemon instance
func New(opts service.ServiceOptions, logger *zap.Logger) (*Daemon, error) {
	return &Daemon{
		options: opts,
		logger:  logger,
		pidFile: opts.PIDFile,
	}, nil
}

// Daemonize forks the process into the background
func (d *Daemon) Daemonize() error {
	// Already a daemon
	if os.Getppid() == 1 {
		return nil
	}

	// Fork off the parent process
	args := os.Args[1:]
	cmd := exec.Command(os.Args[0], args...)
	cmd.Env = os.Environ()
	cmd.Start()
	os.Exit(0)
	return nil // Never reached
}

// Start initializes the daemon process
func (d *Daemon) Start() error {
	// Create working directory if needed
	if d.options.WorkingDir != "" {
		if err := os.MkdirAll(d.options.WorkingDir, 0755); err != nil {
			return fmt.Errorf("failed to create working directory: %w", err)
		}
	}

	// Change to working directory
	if d.options.WorkingDir != "" {
		if err := os.Chdir(d.options.WorkingDir); err != nil {
			return fmt.Errorf("failed to change working directory: %w", err)
		}
	}

	// Create PID file
	if d.pidFile != "" {
		if err := d.createPIDFile(); err != nil {
			return fmt.Errorf("failed to create PID file: %w", err)
		}
	}

	// Set file creation mask
	if d.options.Umask != "" {
		if err := d.setUmask(); err != nil {
			return fmt.Errorf("failed to set umask: %w", err)
		}
	}

	// Set up process limits
	if err := d.setResourceLimits(); err != nil {
		return fmt.Errorf("failed to set resource limits: %w", err)
	}

	// Change user/group if specified
	if err := d.setCredentials(); err != nil {
		return fmt.Errorf("failed to set credentials: %w", err)
	}

	return nil
}

// Stop cleans up daemon resources
func (d *Daemon) Stop() error {
	if d.pidFile != "" {
		if err := os.Remove(d.pidFile); err != nil && !os.IsNotExist(err) {
			return fmt.Errorf("failed to remove PID file: %w", err)
		}
	}
	return nil
}

// createPIDFile creates a PID file with the current process ID
func (d *Daemon) createPIDFile() error {
	// Create directory if needed
	dir := filepath.Dir(d.pidFile)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create PID directory: %w", err)
	}

	// Write PID
	pid := os.Getpid()
	if err := os.WriteFile(d.pidFile, []byte(fmt.Sprintf("%d\n", pid)), 0644); err != nil {
		return fmt.Errorf("failed to write PID file: %w", err)
	}

	return nil
}

// setUmask sets the file creation mask
func (d *Daemon) setUmask() error {
	// Parse umask string (e.g., "0027")
	var mask int
	if _, err := fmt.Sscanf(d.options.Umask, "%o", &mask); err != nil {
		return fmt.Errorf("invalid umask format: %w", err)
	}
	syscall.Umask(mask)
	return nil
}

// setResourceLimits sets process resource limits
func (d *Daemon) setResourceLimits() error {
	var rLimit syscall.Rlimit

	// Set max number of file descriptors
	rLimit.Max = 65536
	rLimit.Cur = 65536
	if err := syscall.Setrlimit(syscall.RLIMIT_NOFILE, &rLimit); err != nil {
		return fmt.Errorf("failed to set RLIMIT_NOFILE: %w", err)
	}

	// Set core dump size
	rLimit.Max = 0
	rLimit.Cur = 0
	if err := syscall.Setrlimit(syscall.RLIMIT_CORE, &rLimit); err != nil {
		return fmt.Errorf("failed to set RLIMIT_CORE: %w", err)
	}

	return nil
}

// setCredentials changes the process user and group
func (d *Daemon) setCredentials() error {
	if d.options.User == "" && d.options.Group == "" {
		return nil
	}

	// Get user info
	if d.options.User != "" {
		uid, gid, err := lookupUser(d.options.User)
		if err != nil {
			return fmt.Errorf("failed to lookup user: %w", err)
		}

		// Set supplementary groups
		if err := syscall.Setgroups([]int{int(gid)}); err != nil {
			return fmt.Errorf("failed to set groups: %w", err)
		}

		// Set group ID
		if err := syscall.Setgid(int(gid)); err != nil {
			return fmt.Errorf("failed to set GID: %w", err)
		}

		// Set user ID
		if err := syscall.Setuid(int(uid)); err != nil {
			return fmt.Errorf("failed to set UID: %w", err)
		}
	}

	// Set group only
	if d.options.Group != "" && d.options.User == "" {
		gid, err := lookupGroup(d.options.Group)
		if err != nil {
			return fmt.Errorf("failed to lookup group: %w", err)
		}

		if err := syscall.Setgid(int(gid)); err != nil {
			return fmt.Errorf("failed to set GID: %w", err)
		}
	}

	return nil
}

// lookupUser gets the UID and GID for a username
func lookupUser(username string) (uint32, uint32, error) {
	// Try to parse as numeric ID first
	var uid, gid uint32
	if _, err := fmt.Sscanf(username, "%d", &uid); err == nil {
		return uid, gid, nil
	}

	// Look up user
	userInfo, err := user.Lookup(username)
	if err != nil {
		return 0, 0, fmt.Errorf("failed to lookup user: %w", err)
	}

	// Parse UID
	uid64, err := strconv.ParseUint(userInfo.Uid, 10, 32)
	if err != nil {
		return 0, 0, fmt.Errorf("invalid UID: %w", err)
	}
	uid = uint32(uid64)

	// Parse GID
	gid64, err := strconv.ParseUint(userInfo.Gid, 10, 32)
	if err != nil {
		return 0, 0, fmt.Errorf("invalid GID: %w", err)
	}
	gid = uint32(gid64)

	return uid, gid, nil
}

// lookupGroup gets the GID for a group name
func lookupGroup(group string) (uint32, error) {
	// Try to parse as numeric ID first
	var gid uint32
	if _, err := fmt.Sscanf(group, "%d", &gid); err == nil {
		return gid, nil
	}

	// Look up group
	groupInfo, err := user.LookupGroup(group)
	if err != nil {
		return 0, fmt.Errorf("failed to lookup group: %w", err)
	}

	// Parse GID
	gid64, err := strconv.ParseUint(groupInfo.Gid, 10, 32)
	if err != nil {
		return 0, fmt.Errorf("invalid GID: %w", err)
	}
	gid = uint32(gid64)

	return gid, nil
}
