//go:build !windows

package provision

// restrictKeyFileWindows is only meaningful on Windows; unix hardening is
// chmod 0600 applied by RestrictKeyFile.
func restrictKeyFileWindows(path string) error { return nil }
