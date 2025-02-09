package control

import (
	"encoding/json"
	"fmt"
	"net"
	"time"

	"github.com/o3willard-AI/SSSonector/internal/config"
	"github.com/o3willard-AI/SSSonector/internal/service"
)

// Client provides control client functionality
type Client struct {
	conn net.Conn
}

// NewClient creates a new control client
func NewClient(network, address string) (*Client, error) {
	conn, err := net.Dial(network, address)
	if err != nil {
		return nil, fmt.Errorf("failed to connect: %w", err)
	}

	return &Client{
		conn: conn,
	}, nil
}

// Close closes the client connection
func (c *Client) Close() error {
	return c.conn.Close()
}

// SendCommand sends a command to the service
func (c *Client) SendCommand(cmd service.ServiceCommand) (string, error) {
	// Encode command
	data, err := json.Marshal(cmd)
	if err != nil {
		return "", fmt.Errorf("failed to encode command: %w", err)
	}

	// Send command
	_, err = c.conn.Write(append(data, '\n'))
	if err != nil {
		return "", fmt.Errorf("failed to send command: %w", err)
	}

	// Read response
	var response string
	decoder := json.NewDecoder(c.conn)
	err = decoder.Decode(&response)
	if err != nil {
		return "", fmt.Errorf("failed to read response: %w", err)
	}

	return response, nil
}

// GetStatus gets service status
func (c *Client) GetStatus() (string, error) {
	cmd := service.ServiceCommand{
		Command:   service.CmdStatus,
		Type:      "status",
		RequestID: fmt.Sprintf("status-%d", time.Now().Unix()),
	}
	return c.SendCommand(cmd)
}

// Reload reloads the service
func (c *Client) Reload() error {
	cmd := service.ServiceCommand{
		Command:   service.CmdReload,
		Type:      "reload",
		RequestID: fmt.Sprintf("reload-%d", time.Now().Unix()),
	}
	_, err := c.SendCommand(cmd)
	return err
}

// UpdateConfig updates service configuration
func (c *Client) UpdateConfig(cfg *config.Config) error {
	// Encode config
	data, err := json.Marshal(cfg)
	if err != nil {
		return fmt.Errorf("failed to encode config: %w", err)
	}

	cmd := service.ServiceCommand{
		Command:   service.CmdUpdateConfig,
		Type:      "update-config",
		RequestID: fmt.Sprintf("config-%d", time.Now().Unix()),
		Payload:   data,
	}
	_, err = c.SendCommand(cmd)
	return err
}

// RotateCerts rotates service certificates
func (c *Client) RotateCerts() error {
	cmd := service.ServiceCommand{
		Command:   service.CmdRotateCerts,
		Type:      "rotate-certs",
		RequestID: fmt.Sprintf("certs-%d", time.Now().Unix()),
	}
	_, err := c.SendCommand(cmd)
	return err
}

// GetMetrics gets service metrics
func (c *Client) GetMetrics() (*service.ServiceMetrics, error) {
	cmd := service.ServiceCommand{
		Command:   service.CmdMetrics,
		Type:      "metrics",
		RequestID: fmt.Sprintf("metrics-%d", time.Now().Unix()),
	}

	response, err := c.SendCommand(cmd)
	if err != nil {
		return nil, err
	}

	var metrics service.ServiceMetrics
	err = json.Unmarshal([]byte(response), &metrics)
	if err != nil {
		return nil, fmt.Errorf("failed to decode metrics: %w", err)
	}

	return &metrics, nil
}

// CheckHealth checks service health
func (c *Client) CheckHealth() error {
	cmd := service.ServiceCommand{
		Command:   service.CmdHealth,
		Type:      "health",
		RequestID: fmt.Sprintf("health-%d", time.Now().Unix()),
	}
	_, err := c.SendCommand(cmd)
	return err
}
