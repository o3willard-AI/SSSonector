//go:build linux

package control

import (
	"fmt"
	"net"

	"golang.org/x/sys/unix"
)

// setSocketCredentials sets socket credentials (Linux only)
func setSocketCredentials(conn net.Conn) error {
	// Get underlying connection
	unixConn, ok := conn.(*net.UnixConn)
	if !ok {
		return fmt.Errorf("not a Unix domain socket connection")
	}

	// Get socket file descriptor
	file, err := unixConn.File()
	if err != nil {
		return fmt.Errorf("failed to get socket file: %w", err)
	}
	defer file.Close()

	fd := int(file.Fd())

	// Enable SO_PASSCRED
	if err := unix.SetsockoptInt(fd, unix.SOL_SOCKET, unix.SO_PASSCRED, 1); err != nil {
		return fmt.Errorf("failed to enable credential passing: %w", err)
	}

	return nil
}

// getSocketCredentials gets peer credentials from socket (Linux only)
func getSocketCredentials(conn net.Conn) (*unix.Ucred, error) {
	// Get underlying connection
	unixConn, ok := conn.(*net.UnixConn)
	if !ok {
		return nil, fmt.Errorf("not a Unix domain socket connection")
	}

	// Get socket file descriptor
	file, err := unixConn.File()
	if err != nil {
		return nil, fmt.Errorf("failed to get socket file: %w", err)
	}
	defer file.Close()

	fd := int(file.Fd())

	// Get credentials
	ucred, err := unix.GetsockoptUcred(fd, unix.SOL_SOCKET, unix.SO_PEERCRED)
	if err != nil {
		return nil, fmt.Errorf("failed to get peer credentials: %w", err)
	}

	return ucred, nil
}
