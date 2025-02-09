//go:build linux

package security

const (
	// Process capabilities
	SECBIT_KEEP_CAPS                   = 0x1
	SECBIT_KEEP_CAPS_LOCKED            = 0x2
	SECBIT_NO_CAP_AMBIENT_RAISE        = 0x4
	SECBIT_NO_CAP_AMBIENT_RAISE_LOCKED = 0x8

	// Memory management
	PR_SET_MM              = 0x9
	PR_SET_MM_MAP_MIN_ADDR = 0x1
	PR_SET_MM_START_STACK  = 0x2

	// Default values
	DEFAULT_MAP_MIN_ADDR = 65536
)
