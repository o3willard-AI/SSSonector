package validator

import (
	"fmt"
	"time"

	"github.com/o3willard-AI/SSSonector/internal/config/types"
)

// Validator represents a configuration validator
type Validator struct{}

// NewValidator creates a new validator
func NewValidator() *Validator {
	return &Validator{}
}

// Validate validates the configuration
func (v *Validator) Validate(config *types.AppConfig) error {
	if config == nil {
		return fmt.Errorf("config is nil")
	}

	if err := v.validateMode(config.Config.Mode); err != nil {
		return err
	}

	if err := v.validateNetwork(&config.Config.Network); err != nil {
		return err
	}

	if err := v.validateTunnel(&config.Config.Tunnel); err != nil {
		return err
	}

	if err := v.validateSecurity(&config.Config.Security); err != nil {
		return err
	}

	if err := v.validateMonitor(&config.Config.Monitor); err != nil {
		return err
	}

	if err := v.validateMetrics(&config.Config.Metrics); err != nil {
		return err
	}

	if err := v.validateThrottle(&config.Throttle); err != nil {
		return err
	}

	return nil
}

// validateMode validates the mode
func (v *Validator) validateMode(mode string) error {
	switch mode {
	case types.ModeServer.String(), types.ModeClient.String():
		return nil
	default:
		return fmt.Errorf("invalid mode: %s", mode)
	}
}

// validateNetwork validates network configuration
func (v *Validator) validateNetwork(config *types.NetworkConfig) error {
	if config.MTU <= 0 {
		config.MTU = 1500
	}

	return nil
}

// validateTunnel validates tunnel configuration
func (v *Validator) validateTunnel(config *types.TunnelConfig) error {
	if config.Protocol == "" {
		config.Protocol = "tcp"
	}

	if config.ServerPort <= 0 {
		config.ServerPort = 8080
	}

	if config.ListenPort <= 0 {
		config.ListenPort = 8080
	}

	return nil
}

// validateSecurity validates security configuration
func (v *Validator) validateSecurity(config *types.SecurityConfig) error {
	if err := v.validateTLS(&config.TLS); err != nil {
		return err
	}

	return nil
}

// validateTLS validates TLS configuration
func (v *Validator) validateTLS(config *types.TLSConfig) error {
	if config.MinVersion == "" {
		config.MinVersion = "1.2"
	}

	if config.MaxVersion == "" {
		config.MaxVersion = "1.3"
	}

	return nil
}

// validateMonitor validates monitor configuration
func (v *Validator) validateMonitor(config *types.MonitorConfig) error {
	if config.Type == "" {
		config.Type = "basic"
	}

	// Set default interval if not set
	if config.Interval == (types.Duration{}) {
		config.Interval = types.Duration{Duration: time.Minute}
	}

	return nil
}

// validateMetrics validates metrics configuration
func (v *Validator) validateMetrics(config *types.MetricsConfig) error {
	if config.Address == "" {
		config.Address = "localhost:8080"
	}

	// Set default interval if not set
	if config.Interval == (types.Duration{}) {
		config.Interval = types.Duration{Duration: 10 * time.Second}
	}

	if config.BufferSize <= 0 {
		config.BufferSize = 1000
	}

	return nil
}

// validateThrottle validates throttle configuration
func (v *Validator) validateThrottle(config *types.ThrottleConfig) error {
	if config.Rate < 0 {
		return fmt.Errorf("invalid rate: %d", config.Rate)
	}

	if config.Burst < 0 {
		return fmt.Errorf("invalid burst: %d", config.Burst)
	}

	return nil
}
