//go:build windows

package control

import (
	"fmt"
	"net"
	"time"
	"unsafe"

	"golang.org/x/sys/windows"
)

const (
	// Pipe configuration
	pipeBufSize     = 65536 // 64KB buffer size
	pipeTimeout     = 5000  // 5 seconds in milliseconds
	maxInstances    = 255   // Maximum pipe instances
	defaultPipeName = `\\.\pipe\sssonector`
)

// pipeListener implements net.Listener for Windows named pipes
type pipeListener struct {
	path         string
	securityAttr *windows.SecurityAttributes
	handle       windows.Handle
	closed       bool
}

// newPipeListener creates a new Windows named pipe listener
func newPipeListener(path string) (net.Listener, error) {
	if path == "" {
		path = defaultPipeName
	}

	// Create security attributes for the pipe
	sa, err := createPipeSecurityAttributes()
	if err != nil {
		return nil, fmt.Errorf("failed to create security attributes: %w", err)
	}

	// Create the first pipe instance
	handle, err := createNamedPipe(path, sa)
	if err != nil {
		return nil, err
	}

	return &pipeListener{
		path:         path,
		securityAttr: sa,
		handle:       handle,
		closed:       false,
	}, nil
}

// Accept waits for and returns the next connection
func (l *pipeListener) Accept() (net.Conn, error) {
	if l.closed {
		return nil, fmt.Errorf("listener closed")
	}

	// Wait for client connection
	err := windows.ConnectNamedPipe(l.handle, nil)
	if err != nil && err != windows.ERROR_PIPE_CONNECTED {
		return nil, fmt.Errorf("connect named pipe failed: %w", err)
	}

	// Create pipe connection
	conn := &pipeConn{
		handle: l.handle,
	}

	// Create new pipe instance for next client
	handle, err := createNamedPipe(l.path, l.securityAttr)
	if err != nil {
		conn.Close()
		return nil, err
	}
	l.handle = handle

	return conn, nil
}

// Close closes the listener
func (l *pipeListener) Close() error {
	if l.closed {
		return nil
	}
	l.closed = true
	return windows.CloseHandle(l.handle)
}

// Addr returns the listener's address
func (l *pipeListener) Addr() net.Addr {
	return pipeAddr(l.path)
}

// pipeConn implements net.Conn for Windows named pipes
type pipeConn struct {
	handle windows.Handle
	closed bool
}

// Read implements io.Reader
func (c *pipeConn) Read(b []byte) (int, error) {
	if c.closed {
		return 0, fmt.Errorf("connection closed")
	}

	var n uint32
	err := windows.ReadFile(c.handle, b, &n, nil)
	if err != nil {
		return 0, fmt.Errorf("read failed: %w", err)
	}
	return int(n), nil
}

// Write implements io.Writer
func (c *pipeConn) Write(b []byte) (int, error) {
	if c.closed {
		return 0, fmt.Errorf("connection closed")
	}

	var n uint32
	err := windows.WriteFile(c.handle, b, &n, nil)
	if err != nil {
		return 0, fmt.Errorf("write failed: %w", err)
	}
	return int(n), nil
}

// Close closes the connection
func (c *pipeConn) Close() error {
	if c.closed {
		return nil
	}
	c.closed = true
	return windows.CloseHandle(c.handle)
}

// LocalAddr returns the local network address
func (c *pipeConn) LocalAddr() net.Addr {
	return pipeAddr("")
}

// RemoteAddr returns the remote network address
func (c *pipeConn) RemoteAddr() net.Addr {
	return pipeAddr("")
}

// SetDeadline implements net.Conn
func (c *pipeConn) SetDeadline(t time.Time) error {
	return fmt.Errorf("deadlines not supported")
}

// SetReadDeadline implements net.Conn
func (c *pipeConn) SetReadDeadline(t time.Time) error {
	return fmt.Errorf("read deadlines not supported")
}

// SetWriteDeadline implements net.Conn
func (c *pipeConn) SetWriteDeadline(t time.Time) error {
	return fmt.Errorf("write deadlines not supported")
}

// pipeAddr implements net.Addr for Windows named pipes
type pipeAddr string

func (a pipeAddr) Network() string { return "pipe" }
func (a pipeAddr) String() string  { return string(a) }

// createPipeSecurityAttributes creates security attributes for the pipe
func createPipeSecurityAttributes() (*windows.SecurityAttributes, error) {
	// Create a security descriptor with default DACL
	sd, err := windows.NewSecurityDescriptor()
	if err != nil {
		return nil, fmt.Errorf("failed to create security descriptor: %w", err)
	}

	// Set default DACL allowing access to authenticated users
	dacl, err := windows.ACLFromEntries([]windows.EXPLICIT_ACCESS{
		{
			AccessPermissions: windows.GENERIC_READ | windows.GENERIC_WRITE,
			AccessMode:        windows.GRANT_ACCESS,
			Inheritance:       windows.NO_INHERITANCE,
			Trustee: windows.TRUSTEE{
				TrusteeForm:  windows.TRUSTEE_IS_SID,
				TrusteeType:  windows.TRUSTEE_IS_GROUP,
				TrusteeValue: getTrusteeSID(windows.WinAuthenticatedUserSid),
			},
		},
	}, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create DACL: %w", err)
	}

	if err := sd.SetDACL(dacl, true, false); err != nil {
		return nil, fmt.Errorf("failed to set DACL: %w", err)
	}

	return &windows.SecurityAttributes{
		Length:             uint32(unsafe.Sizeof(windows.SecurityAttributes{})),
		SecurityDescriptor: sd,
		InheritHandle:      1,
	}, nil
}

// getTrusteeSID creates a SID for the given well-known SID type
func getTrusteeSID(sidType windows.WELL_KNOWN_SID_TYPE) windows.TrusteeValue {
	sid, err := windows.CreateWellKnownSid(sidType)
	if err != nil {
		// Fall back to Everyone if we can't create the requested SID
		sid, _ = windows.CreateWellKnownSid(windows.WinWorldSid)
	}
	return windows.TrusteeValue(unsafe.Pointer(sid))
}

// createNamedPipe creates a new named pipe instance
func createNamedPipe(path string, sa *windows.SecurityAttributes) (windows.Handle, error) {
	path16, err := windows.UTF16PtrFromString(path)
	if err != nil {
		return 0, fmt.Errorf("failed to convert path: %w", err)
	}

	handle, err := windows.CreateNamedPipe(
		path16,
		windows.PIPE_ACCESS_DUPLEX|windows.FILE_FLAG_OVERLAPPED,
		windows.PIPE_TYPE_MESSAGE|windows.PIPE_READMODE_MESSAGE|windows.PIPE_WAIT,
		maxInstances,
		pipeBufSize,
		pipeBufSize,
		pipeTimeout,
		sa,
	)
	if err != nil {
		return 0, fmt.Errorf("failed to create named pipe: %w", err)
	}

	return handle, nil
}
