//go:build darwin

package control

import (
	"net"
	"os"

	"golang.org/x/sys/unix"
)

// darwinCred represents process credentials on Darwin systems
type darwinCred struct {
	Uid uint32
}

// setSocketCredentials is a no-op on Darwin systems
func setSocketCredentials(conn net.Conn) error {
	// macOS doesn't support credential passing
	return nil
}

// getSocketCredentials returns a mock credential on Darwin systems
func getSocketCredentials(conn net.Conn) (*unix.Ucred, error) {
	// Return current process credentials as a fallback
	return &unix.Ucred{
		Uid: uint32(os.Getuid()),
	}, nil
}
