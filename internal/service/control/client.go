package control

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"time"

	"github.com/o3willard-AI/SSSonector/internal/config"
	"github.com/o3willard-AI/SSSonector/internal/service"
	"go.uber.org/zap"
)

// Client represents a control client
type Client struct {
	cfg        *config.AppConfig
	logger     *zap.Logger
	conn       net.Conn
	socketPath string
}

// NewClient creates a new control client
func NewClient(cfg *config.AppConfig, logger *zap.Logger) (*Client, error) {
	return &Client{
		cfg:        cfg,
		logger:     logger,
		socketPath: filepath.Join(os.TempDir(), "sssonector.sock"),
	}, nil
}

// SetSocketPath sets the control socket path
func (c *Client) SetSocketPath(path string) {
	c.socketPath = path
}

// Connect establishes a connection to the control socket
func (c *Client) Connect() error {
	var err error

	// Connect with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Create connection
	var d net.Dialer
	c.conn, err = d.DialContext(ctx, "unix", c.socketPath)
	if err != nil {
		return fmt.Errorf("failed to connect to control socket: %w", err)
	}

	return nil
}

// Close closes the control connection
func (c *Client) Close() error {
	if c.conn != nil {
		return c.conn.Close()
	}
	return nil
}

// ExecuteCommand executes a service command
func (c *Client) ExecuteCommand(cmd service.ServiceCommand, args map[string]interface{}) (*service.ServiceResponse, error) {
	if c.conn == nil {
		return nil, fmt.Errorf("not connected")
	}

	// Create command request
	request := struct {
		Command service.ServiceCommand `json:"command"`
		Args    map[string]interface{} `json:"args,omitempty"`
	}{
		Command: cmd,
		Args:    args,
	}

	// Marshal request
	data, err := json.Marshal(request)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	// Send request
	if _, err := c.conn.Write(append(data, '\n')); err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}

	// Read response
	buf := make([]byte, 4096)
	n, err := c.conn.Read(buf)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	// Parse response
	var response service.ServiceResponse
	if err := json.Unmarshal(buf[:n], &response); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	// Handle status and metrics responses
	if response.Data != nil {
		switch cmd {
		case service.CmdStatus:
			var status service.ServiceStatus
			data, err := json.Marshal(response.Data)
			if err != nil {
				return nil, fmt.Errorf("failed to marshal status data: %w", err)
			}
			if err := json.Unmarshal(data, &status); err != nil {
				return nil, fmt.Errorf("failed to unmarshal status data: %w", err)
			}
			response.Data = &status

		case service.CmdMetrics:
			var metrics service.ServiceMetrics
			data, err := json.Marshal(response.Data)
			if err != nil {
				return nil, fmt.Errorf("failed to marshal metrics data: %w", err)
			}
			if err := json.Unmarshal(data, &metrics); err != nil {
				return nil, fmt.Errorf("failed to unmarshal metrics data: %w", err)
			}
			response.Data = &metrics
		}
	}

	return &response, nil
}
