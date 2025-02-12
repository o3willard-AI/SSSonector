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

	if config.Config == nil {
		config.Config = types.NewConfig()
	}

	if err := v.validateMode(config.Config.Mode.String()); err != nil {
		return err
	}

	if config.Config.Logging != nil {
		if err := v.validateLogging(config.Config.Logging); err != nil {
			return err
		}
	}

	if config.Config.Network != nil {
		if err := v.validateNetwork(config.Config.Network); err != nil {
			return err
		}
	}

	if config.Config.Tunnel != nil {
		if err := v.validateTunnel(config.Config.Tunnel); err != nil {
			return err
		}
	}

	if config.Config.Security != nil {
		if err := v.validateSecurity(config.Config.Security); err != nil {
			return err
		}
	}

	if config.Config.Monitor != nil {
		if err := v.validateMonitor(config.Config.Monitor); err != nil {
			return err
		}
	}

	if config.Config.Metrics != nil {
		if err := v.validateMetrics(config.Config.Metrics); err != nil {
			return err
		}
	}

	if config.Throttle != nil {
		if err := v.validateThrottle(config.Throttle); err != nil {
			return err
		}
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

// validateLogging validates logging configuration
func (v *Validator) validateLogging(config *types.LoggingConfig) error {
	if config == nil {
		return nil
	}

	if config.Level == "" {
		config.Level = "info"
	}

	if config.Format == "" {
		config.Format = "text"
	}

	if config.Output == "" {
		config.Output = "stdout"
	}

	return nil
}

// validateNetwork validates network configuration
func (v *Validator) validateNetwork(config *types.NetworkConfig) error {
	if config == nil {
		return nil
	}

	if config.MTU <= 0 {
		config.MTU = 1500
	}

	return nil
}

// validateTunnel validates tunnel configuration
func (v *Validator) validateTunnel(config *types.TunnelConfig) error {
	if config == nil {
		return nil
	}

	if config.MTU <= 0 {
		config.MTU = 1500
	}

	if config.Protocol == "" {
		config.Protocol = "tcp"
	}

	if config.Keepalive.Duration == 0 {
		config.Keepalive = types.NewDuration(60 * time.Second)
	}

	return nil
}

// validateSecurity validates security configuration
func (v *Validator) validateSecurity(config *types.SecurityConfig) error {
	if config == nil {
		return nil
	}

	if config.TLS != nil {
		if err := v.validateTLS(config.TLS); err != nil {
			return err
		}
	}

	return nil
}

// validateTLS validates TLS configuration
func (v *Validator) validateTLS(config *types.TLSConfig) error {
	if config == nil {
		return nil
	}

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
	if config == nil {
		return nil
	}

	if config.Type == "" {
		config.Type = "basic"
	}

	if config.Interval.Duration == 0 {
		config.Interval = types.NewDuration(time.Minute)
	}

	return nil
}

// validateMetrics validates metrics configuration
func (v *Validator) validateMetrics(config *types.MetricsConfig) error {
	if config == nil {
		return nil
	}

	if config.Address == "" {
		config.Address = "localhost:8080"
	}

	if config.Interval.Duration == 0 {
		config.Interval = types.NewDuration(10 * time.Second)
	}

	if config.BufferSize <= 0 {
		config.BufferSize = 1000
	}

	return nil
}

// validateThrottle validates throttle configuration
func (v *Validator) validateThrottle(config *types.ThrottleConfig) error {
	if config == nil {
		return nil
	}

	if config.Rate < 0 {
		return fmt.Errorf("invalid rate: %d", config.Rate)
	}

	if config.Burst < 0 {
		return fmt.Errorf("invalid burst: %d", config.Burst)
	}

	return nil
}
