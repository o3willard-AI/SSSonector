package provision

import (
	"errors"
	"os"
	"path/filepath"
	"runtime"
)

// ErrNotTerminal is returned when secret entry is attempted without a TTY.
// The daemon refuses to read pairing secrets from pipes/files (fail closed).
var ErrNotTerminal = errors.New("provision: stdin is not a terminal; refusing to read pairing code non-interactively (pipe it via --from file redemption or use a real console)")

// DefaultCertsDir returns the platform-correct certificate directory:
//
//	linux:   /etc/sssonector/certs
//	darwin:  /Library/Application Support/sssonector/certs
//	windows: %ProgramData%\SSSonector\certs
func DefaultCertsDir() string {
	switch runtime.GOOS {
	case "windows":
		base := os.Getenv("ProgramData")
		if base == "" {
			base = `C:\ProgramData`
		}
		return filepath.Join(base, "SSSonector", "certs")
	case "darwin":
		return "/Library/Application Support/sssonector/certs"
	default:
		return "/etc/sssonector/certs"
	}
}

// RestrictKeyFile hardens a private-key file after writing: chmod 0600 on
// unix; owner/Administrators/SYSTEM-only ACL on Windows. Best-effort on the
// Windows path (ACL application failures are reported but non-fatal there,
// since the default ProgramData inheritance already limits access).
func RestrictKeyFile(path string) error {
	if runtime.GOOS == "windows" {
		return restrictKeyFileWindows(path)
	}
	return os.Chmod(path, 0o600)
}
