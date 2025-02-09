package control

import (
	"encoding/json"
	"fmt"
	"io"
	"time"

	"github.com/o3willard-AI/SSSonector/internal/service"
)

// ControlInterface provides control interface functionality
type ControlInterface struct {
	service service.Service
	reader  io.Reader
	writer  io.Writer
}

// NewControlInterface creates a new control interface
func NewControlInterface(svc service.Service, r io.Reader, w io.Writer) *ControlInterface {
	return &ControlInterface{
		service: svc,
		reader:  r,
		writer:  w,
	}
}

// HandleCommand handles a control command
func (c *ControlInterface) HandleCommand(cmd service.ServiceCommand) (string, error) {
	var response string
	var err error

	switch cmd.Command {
	case service.CmdStatus:
		status, err := c.service.Status()
		if err != nil {
			return "", err
		}
		response = status

	case service.CmdMetrics:
		metrics, err := c.service.GetMetrics()
		if err != nil {
			return "", err
		}
		data, err := json.Marshal(metrics)
		if err != nil {
			return "", err
		}
		response = string(data)

	case service.CmdHealth:
		err = c.service.Health()
		if err != nil {
			return "", err
		}
		response = "healthy"

	default:
		response, err = c.service.ExecuteCommand(cmd)
		if err != nil {
			return "", err
		}
	}

	return response, nil
}

// SendResponse sends a response
func (c *ControlInterface) SendResponse(response string) error {
	_, err := fmt.Fprintln(c.writer, response)
	return err
}

// ReadCommand reads a command
func (c *ControlInterface) ReadCommand() (service.ServiceCommand, error) {
	var cmd service.ServiceCommand
	decoder := json.NewDecoder(c.reader)
	err := decoder.Decode(&cmd)
	if err != nil {
		return cmd, err
	}
	return cmd, nil
}

// Close closes the control interface
func (c *ControlInterface) Close() error {
	if closer, ok := c.reader.(io.Closer); ok {
		closer.Close()
	}
	if closer, ok := c.writer.(io.Closer); ok {
		closer.Close()
	}
	return nil
}

// WaitForCommand waits for a command with timeout
func (c *ControlInterface) WaitForCommand(timeout time.Duration) (service.ServiceCommand, error) {
	done := make(chan struct{})
	var cmd service.ServiceCommand
	var err error

	go func() {
		cmd, err = c.ReadCommand()
		close(done)
	}()

	select {
	case <-done:
		return cmd, err
	case <-time.After(timeout):
		return cmd, fmt.Errorf("timeout waiting for command")
	}
}
