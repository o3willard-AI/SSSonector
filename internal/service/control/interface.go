package control

import (
	"encoding/json"
	"fmt"
	"net"
	"os"
	"path/filepath"

	"github.com/o3willard-AI/SSSonector/internal/service"
)

// ControlServer represents a control server
type ControlServer struct {
	service    service.Service
	socket     net.Listener
	socketPath string
}

// NewControlServer creates a new control server
func NewControlServer(svc service.Service) (*ControlServer, error) {
	return &ControlServer{
		service: svc,
	}, nil
}

// SetSocketPath sets the control socket path
func (c *ControlServer) SetSocketPath(path string) {
	c.socketPath = path
}

// handleCommand handles a control command
func (c *ControlServer) handleCommand(cmd service.ServiceCommand) (*service.ServiceResponse, error) {
	switch cmd {
	case service.CmdStatus:
		status, err := c.service.Status()
		if err != nil {
			return nil, err
		}
		return &service.ServiceResponse{
			Success: true,
			Data:    status,
		}, nil

	case service.CmdMetrics:
		metrics, err := c.service.Metrics()
		if err != nil {
			return nil, err
		}
		return &service.ServiceResponse{
			Success: true,
			Data:    metrics,
		}, nil

	case service.CmdHealth:
		if err := c.service.Health(); err != nil {
			return nil, err
		}
		return &service.ServiceResponse{
			Success: true,
			Message: "Service is healthy",
		}, nil

	case service.CmdStart:
		if err := c.service.Start(); err != nil {
			return nil, err
		}
		return &service.ServiceResponse{
			Success: true,
			Message: "Service started",
		}, nil

	case service.CmdStop:
		if err := c.service.Stop(); err != nil {
			return nil, err
		}
		return &service.ServiceResponse{
			Success: true,
			Message: "Service stopped",
		}, nil

	case service.CmdReload:
		if err := c.service.Reload(); err != nil {
			return nil, err
		}
		return &service.ServiceResponse{
			Success: true,
			Message: "Configuration reloaded",
		}, nil

	default:
		return nil, service.NewServiceError(service.ErrInvalidCommand, fmt.Sprintf("Unknown command: %s", cmd))
	}
}

// handleConnection handles a client connection
func (c *ControlServer) handleConnection(conn net.Conn) {
	defer conn.Close()

	// Read command
	buf := make([]byte, 4096)
	n, err := conn.Read(buf)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to read command: %v\n", err)
		return
	}

	// Parse command
	var cmd service.ServiceCommand
	if err := json.Unmarshal(buf[:n], &cmd); err != nil {
		fmt.Fprintf(os.Stderr, "Failed to parse command: %v\n", err)
		return
	}

	// Handle command
	resp, err := c.handleCommand(cmd)
	if err != nil {
		resp = &service.ServiceResponse{
			Success: false,
			Message: err.Error(),
		}
	}

	// Send response
	data, err := json.Marshal(resp)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to marshal response: %v\n", err)
		return
	}

	if _, err := conn.Write(data); err != nil {
		fmt.Fprintf(os.Stderr, "Failed to send response: %v\n", err)
	}
}

// Start starts the control server
func (c *ControlServer) Start() error {
	if c.socketPath == "" {
		c.socketPath = filepath.Join(os.TempDir(), "sssonector.sock")
	}

	// Create socket directory if it doesn't exist
	if err := os.MkdirAll(filepath.Dir(c.socketPath), 0755); err != nil {
		return fmt.Errorf("failed to create socket directory: %w", err)
	}

	// Remove existing socket if it exists
	if err := os.Remove(c.socketPath); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to remove existing socket: %w", err)
	}

	// Create socket
	listener, err := net.Listen("unix", c.socketPath)
	if err != nil {
		return fmt.Errorf("failed to create control socket: %w", err)
	}
	c.socket = listener

	// Handle connections
	go func() {
		for {
			conn, err := c.socket.Accept()
			if err != nil {
				if !isClosedError(err) {
					fmt.Fprintf(os.Stderr, "Failed to accept connection: %v\n", err)
				}
				return
			}
			go c.handleConnection(conn)
		}
	}()

	return nil
}

// Stop stops the control server
func (c *ControlServer) Stop() error {
	if c.socket != nil {
		return c.socket.Close()
	}
	return nil
}

// isClosedError checks if the error is due to closed connection
func isClosedError(err error) bool {
	if err == nil {
		return false
	}
	return err.Error() == "use of closed network connection"
}
