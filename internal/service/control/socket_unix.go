//go:build !windows

package control

import (
	"fmt"
	"net"
	"os"
	"path/filepath"

	"golang.org/x/sys/unix"
)

// newUnixListener creates a new Unix domain socket listener
func newUnixListener(path string) (net.Listener, error) {
	// Create socket directory if needed
	socketDir := filepath.Dir(path)
	if err := os.MkdirAll(socketDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create socket directory: %w", err)
	}

	// Remove existing socket if any
	if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
		return nil, fmt.Errorf("failed to remove existing socket: %w", err)
	}

	// Create Unix domain socket
	listener, err := net.Listen("unix", path)
	if err != nil {
		return nil, fmt.Errorf("failed to create Unix domain socket: %w", err)
	}

	// Set socket permissions
	if err := os.Chmod(path, 0660); err != nil {
		listener.Close()
		return nil, fmt.Errorf("failed to set socket permissions: %w", err)
	}

	// Get socket file descriptor
	unixListener, ok := listener.(*net.UnixListener)
	if !ok {
		listener.Close()
		return nil, fmt.Errorf("not a Unix domain socket")
	}

	// Set socket options
	file, err := unixListener.File()
	if err != nil {
		listener.Close()
		return nil, fmt.Errorf("failed to get socket file: %w", err)
	}
	defer file.Close()

	fd := int(file.Fd())

	// Set close-on-exec
	if err := unix.SetNonblock(fd, true); err != nil {
		listener.Close()
		return nil, fmt.Errorf("failed to set non-blocking mode: %w", err)
	}

	// Set socket buffer sizes
	if err := unix.SetsockoptInt(fd, unix.SOL_SOCKET, unix.SO_RCVBUF, 65536); err != nil {
		listener.Close()
		return nil, fmt.Errorf("failed to set receive buffer size: %w", err)
	}

	if err := unix.SetsockoptInt(fd, unix.SOL_SOCKET, unix.SO_SNDBUF, 65536); err != nil {
		listener.Close()
		return nil, fmt.Errorf("failed to set send buffer size: %w", err)
	}

	// Set socket timeouts
	tv := unix.Timeval{Sec: 5, Usec: 0}
	if err := unix.SetsockoptTimeval(fd, unix.SOL_SOCKET, unix.SO_RCVTIMEO, &tv); err != nil {
		listener.Close()
		return nil, fmt.Errorf("failed to set receive timeout: %w", err)
	}

	if err := unix.SetsockoptTimeval(fd, unix.SOL_SOCKET, unix.SO_SNDTIMEO, &tv); err != nil {
		listener.Close()
		return nil, fmt.Errorf("failed to set send timeout: %w", err)
	}

	return listener, nil
}

// setSocketNonBlocking sets a socket to non-blocking mode
func setSocketNonBlocking(conn net.Conn) error {
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

	// Set non-blocking mode
	if err := unix.SetNonblock(fd, true); err != nil {
		return fmt.Errorf("failed to set non-blocking mode: %w", err)
	}

	return nil
}

// setSocketBuffers sets socket buffer sizes
func setSocketBuffers(conn net.Conn, size int) error {
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

	// Set buffer sizes
	if err := unix.SetsockoptInt(fd, unix.SOL_SOCKET, unix.SO_RCVBUF, size); err != nil {
		return fmt.Errorf("failed to set receive buffer size: %w", err)
	}

	if err := unix.SetsockoptInt(fd, unix.SOL_SOCKET, unix.SO_SNDBUF, size); err != nil {
		return fmt.Errorf("failed to set send buffer size: %w", err)
	}

	return nil
}

// setSocketTimeouts sets socket read/write timeouts
func setSocketTimeouts(conn net.Conn, timeout int64) error {
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

	// Set timeouts
	tv := unix.Timeval{
		Sec:  timeout,
		Usec: 0,
	}

	if err := unix.SetsockoptTimeval(fd, unix.SOL_SOCKET, unix.SO_RCVTIMEO, &tv); err != nil {
		return fmt.Errorf("failed to set receive timeout: %w", err)
	}

	if err := unix.SetsockoptTimeval(fd, unix.SOL_SOCKET, unix.SO_SNDTIMEO, &tv); err != nil {
		return fmt.Errorf("failed to set send timeout: %w", err)
	}

	return nil
}

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
