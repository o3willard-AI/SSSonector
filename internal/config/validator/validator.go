// Package validator provides configuration validation functionality
package validator

import (
	"fmt"
	"net"
	"strings"

	"github.com/o3willard-AI/SSSonector/internal/config/types"
)

// Validator implements ConfigValidator interface
type Validator struct{}

// NewValidator creates a new Validator instance
func NewValidator() *Validator {
	return &Validator{}
}

// Validate validates the configuration
func (v *Validator) Validate(config *types.AppConfig) error {
	if config == nil {
		return fmt.Errorf("config cannot be nil")
	}

	if config.Config == nil {
		return fmt.Errorf("config.Config cannot be nil")
	}

	if err := v.validateMode(config.Config.Mode); err != nil {
		return fmt.Errorf("invalid mode: %v", err)
	}

	if err := v.validateLogging(config.Config.Logging); err != nil {
		return fmt.Errorf("invalid logging config: %v", err)
	}

	if err := v.validateNetwork(config.Config.Network); err != nil {
		return fmt.Errorf("invalid network config: %v", err)
	}

	if err := v.validateTunnel(config.Config.Tunnel); err != nil {
		return fmt.Errorf("invalid tunnel config: %v", err)
	}

	if err := v.validateSecurity(config.Config.Security); err != nil {
		return fmt.Errorf("invalid security config: %v", err)
	}

	if err := v.validateMonitor(config.Config.Monitor); err != nil {
		return fmt.Errorf("invalid monitor config: %v", err)
	}

	if err := v.validateMetrics(config.Config.Metrics); err != nil {
		return fmt.Errorf("invalid metrics config: %v", err)
	}

	if err := v.validateThrottle(config.Throttle); err != nil {
		return fmt.Errorf("invalid throttle config: %v", err)
	}

	return nil
}

func (v *Validator) validateMode(mode string) error {
	switch mode {
	case types.ModeServer, types.ModeClient:
		return nil
	default:
		return fmt.Errorf("invalid mode: %s", mode)
	}
}

func (v *Validator) validateLogging(config types.LoggingConfig) error {
	if config.Level == "" {
		return fmt.Errorf("log level cannot be empty")
	}

	validLevels := map[string]bool{
		"debug": true,
		"info":  true,
		"warn":  true,
		"error": true,
		"fatal": true,
	}

	if !validLevels[strings.ToLower(config.Level)] {
		return fmt.Errorf("invalid log level: %s", config.Level)
	}

	return nil
}

func (v *Validator) validateNetwork(config types.NetworkConfig) error {
	if config.Interface == "" {
		return fmt.Errorf("interface cannot be empty")
	}

	if config.MTU < 576 || config.MTU > 65535 {
		return fmt.Errorf("invalid MTU: %d", config.MTU)
	}

	if config.Address != "" {
		if ip := net.ParseIP(config.Address); ip == nil {
			return fmt.Errorf("invalid IP address: %s", config.Address)
		}
	}

	for _, dns := range config.DNSServers {
		if ip := net.ParseIP(dns); ip == nil {
			return fmt.Errorf("invalid DNS server IP: %s", dns)
		}
	}

	return nil
}

func (v *Validator) validateTunnel(config types.TunnelConfig) error {
	if config.Port < 1 || config.Port > 65535 {
		return fmt.Errorf("invalid port: %d", config.Port)
	}

	validProtocols := map[string]bool{
		"tcp":  true,
		"udp":  true,
		"quic": true,
	}

	if !validProtocols[strings.ToLower(config.Protocol)] {
		return fmt.Errorf("invalid protocol: %s", config.Protocol)
	}

	return nil
}

func (v *Validator) validateSecurity(config types.SecurityConfig) error {
	if config.TLS.MinVersion == "" {
		return fmt.Errorf("TLS min version cannot be empty")
	}

	if config.TLS.MaxVersion == "" {
		return fmt.Errorf("TLS max version cannot be empty")
	}

	validVersions := map[string]bool{
		"1.2": true,
		"1.3": true,
	}

	if !validVersions[config.TLS.MinVersion] {
		return fmt.Errorf("invalid TLS min version: %s", config.TLS.MinVersion)
	}

	if !validVersions[config.TLS.MaxVersion] {
		return fmt.Errorf("invalid TLS max version: %s", config.TLS.MaxVersion)
	}

	return nil
}

func (v *Validator) validateMonitor(config types.MonitorConfig) error {
	if !config.Enabled {
		return nil
	}

	if config.Type == "" {
		return fmt.Errorf("monitor type cannot be empty when monitoring is enabled")
	}

	validTypes := map[string]bool{
		"prometheus": true,
		"snmp":       true,
	}

	if !validTypes[strings.ToLower(config.Type)] {
		return fmt.Errorf("invalid monitor type: %s", config.Type)
	}

	if config.Interval.Seconds() < 1 {
		return fmt.Errorf("invalid monitor interval: %v", config.Interval)
	}

	return nil
}

func (v *Validator) validateMetrics(config types.MetricsConfig) error {
	if !config.Enabled {
		return nil
	}

	if config.Address == "" {
		return fmt.Errorf("metrics address cannot be empty when metrics are enabled")
	}

	if config.Interval.Seconds() < 1 {
		return fmt.Errorf("invalid metrics interval: %v", config.Interval)
	}

	if config.BufferSize < 1 {
		return fmt.Errorf("invalid metrics buffer size: %d", config.BufferSize)
	}

	return nil
}

func (v *Validator) validateThrottle(config types.ThrottleConfig) error {
	if !config.Enabled {
		return nil
	}

	if config.Rate <= 0 {
		return fmt.Errorf("invalid rate: %f", config.Rate)
	}

	if config.Burst <= 0 {
		return fmt.Errorf("invalid burst: %d", config.Burst)
	}

	return nil
}
