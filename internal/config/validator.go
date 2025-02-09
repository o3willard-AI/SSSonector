package config

import (
	"encoding/json"
	"fmt"
	"net"
	"strings"

	"go.uber.org/zap"
)

// Validator implements configuration validation
type Validator struct {
	logger *zap.Logger
}

// NewValidator creates a new configuration validator
func NewValidator(logger *zap.Logger) *Validator {
	return &Validator{
		logger: logger,
	}
}

// Validate validates a configuration
func (v *Validator) Validate(cfg *AppConfig) error {
	// Validate mode
	if cfg.Mode != ModeServer && cfg.Mode != ModeClient {
		return fmt.Errorf("invalid mode: %s", cfg.Mode)
	}

	// Validate network configuration
	if err := v.validateNetwork(&cfg.Network); err != nil {
		return fmt.Errorf("invalid network configuration: %w", err)
	}

	// Validate tunnel configuration
	if err := v.validateTunnel(&cfg.Tunnel); err != nil {
		return fmt.Errorf("invalid tunnel configuration: %w", err)
	}

	// Validate monitor configuration
	if err := v.validateMonitor(&cfg.Monitor); err != nil {
		return fmt.Errorf("invalid monitor configuration: %w", err)
	}

	// Validate throttle configuration
	if err := v.validateThrottle(&cfg.Throttle); err != nil {
		return fmt.Errorf("invalid throttle configuration: %w", err)
	}

	// Validate security configuration
	if err := v.validateSecurity(&cfg.Security); err != nil {
		return fmt.Errorf("invalid security configuration: %w", err)
	}

	return nil
}

// ValidateSchema validates a configuration against its schema
func (v *Validator) ValidateSchema(cfg *AppConfig, schema []byte) error {
	// Parse schema
	var schemaMap map[string]interface{}
	if err := json.Unmarshal(schema, &schemaMap); err != nil {
		return fmt.Errorf("failed to parse schema: %w", err)
	}

	// TODO: Implement JSON Schema validation
	return nil
}

// validateNetwork validates network configuration
func (v *Validator) validateNetwork(cfg *NetworkConfig) error {
	// Validate MTU
	if cfg.MTU < 576 || cfg.MTU > 65535 {
		return fmt.Errorf("invalid MTU: %d", cfg.MTU)
	}

	// Validate IP address
	if cfg.IPAddress != "" {
		if ip := net.ParseIP(cfg.IPAddress); ip == nil {
			return fmt.Errorf("invalid IP address: %s", cfg.IPAddress)
		}
	}

	// Validate subnet mask
	if cfg.SubnetMask != "" {
		if ip := net.ParseIP(cfg.SubnetMask); ip == nil {
			return fmt.Errorf("invalid subnet mask: %s", cfg.SubnetMask)
		}
	}

	// Validate gateway
	if cfg.Gateway != "" {
		if ip := net.ParseIP(cfg.Gateway); ip == nil {
			return fmt.Errorf("invalid gateway: %s", cfg.Gateway)
		}
	}

	// Validate DNS servers
	for _, dns := range cfg.DNSServers {
		if ip := net.ParseIP(dns); ip == nil {
			return fmt.Errorf("invalid DNS server: %s", dns)
		}
	}

	return nil
}

// validateTunnel validates tunnel configuration
func (v *Validator) validateTunnel(cfg *TunnelConfig) error {
	// Validate server address
	if cfg.ServerAddress != "" {
		if _, err := net.ResolveTCPAddr("tcp", fmt.Sprintf("%s:%d", cfg.ServerAddress, cfg.ServerPort)); err != nil {
			return fmt.Errorf("invalid server address: %w", err)
		}
	}

	// Validate listen address
	if cfg.ListenAddress != "" {
		if _, err := net.ResolveTCPAddr("tcp", fmt.Sprintf("%s:%d", cfg.ListenAddress, cfg.ListenPort)); err != nil {
			return fmt.Errorf("invalid listen address: %w", err)
		}
	}

	// Validate protocol
	if cfg.Protocol != "tcp" && cfg.Protocol != "udp" {
		return fmt.Errorf("invalid protocol: %s", cfg.Protocol)
	}

	// Validate encryption
	if !strings.HasPrefix(cfg.Encryption, "aes-") {
		return fmt.Errorf("invalid encryption: %s", cfg.Encryption)
	}

	// Validate compression
	if cfg.Compression != "none" && cfg.Compression != "gzip" && cfg.Compression != "zlib" {
		return fmt.Errorf("invalid compression: %s", cfg.Compression)
	}

	// Validate keep alive
	if cfg.KeepAlive < 0 {
		return fmt.Errorf("invalid keep alive: %v", cfg.KeepAlive)
	}

	// Validate max connections
	if cfg.MaxConnections <= 0 {
		return fmt.Errorf("invalid max connections: %d", cfg.MaxConnections)
	}

	// Validate max clients
	if cfg.MaxClients <= 0 {
		return fmt.Errorf("invalid max clients: %d", cfg.MaxClients)
	}

	// Validate buffer size
	if cfg.BufferSize <= 0 {
		return fmt.Errorf("invalid buffer size: %d", cfg.BufferSize)
	}

	return nil
}

// validateMonitor validates monitor configuration
func (v *Validator) validateMonitor(cfg *MonitorConfig) error {
	if !cfg.Enabled {
		return nil
	}

	// Validate interval
	if cfg.Interval <= 0 {
		return fmt.Errorf("invalid interval: %v", cfg.Interval)
	}

	// Validate SNMP configuration
	if cfg.SNMPEnabled {
		if cfg.SNMPPort <= 0 || cfg.SNMPPort > 65535 {
			return fmt.Errorf("invalid SNMP port: %d", cfg.SNMPPort)
		}
		if cfg.SNMPCommunity == "" {
			return fmt.Errorf("SNMP community string required")
		}
	}

	// Validate Prometheus configuration
	if cfg.Prometheus.Enabled {
		if cfg.Prometheus.Port <= 0 || cfg.Prometheus.Port > 65535 {
			return fmt.Errorf("invalid Prometheus port: %d", cfg.Prometheus.Port)
		}
		if cfg.Prometheus.Path == "" {
			return fmt.Errorf("Prometheus metrics path required")
		}
	}

	// Validate log level
	switch strings.ToLower(cfg.LogLevel) {
	case "debug", "info", "warn", "error":
	default:
		return fmt.Errorf("invalid log level: %s", cfg.LogLevel)
	}

	return nil
}

// validateThrottle validates throttle configuration
func (v *Validator) validateThrottle(cfg *ThrottleConfig) error {
	if !cfg.Enabled {
		return nil
	}

	// Validate rate
	if cfg.Rate <= 0 {
		return fmt.Errorf("invalid rate: %f", cfg.Rate)
	}

	// Validate burst
	if cfg.Burst <= 0 {
		return fmt.Errorf("invalid burst: %d", cfg.Burst)
	}

	return nil
}

// validateSecurity validates security configuration
func (v *Validator) validateSecurity(cfg *SecurityConfig) error {
	// Validate TLS configuration
	if cfg.TLS.MinVersion == "" {
		return fmt.Errorf("TLS minimum version required")
	}
	if cfg.TLS.MaxVersion == "" {
		return fmt.Errorf("TLS maximum version required")
	}
	if len(cfg.TLS.Ciphers) == 0 {
		return fmt.Errorf("TLS ciphers required")
	}

	// Validate authentication method
	switch cfg.AuthMethod {
	case "certificate", "password", "token":
	default:
		return fmt.Errorf("invalid authentication method: %s", cfg.AuthMethod)
	}

	// Validate ACLs
	for _, acl := range cfg.ACLs {
		if _, _, err := net.ParseCIDR(acl.Network); err != nil {
			return fmt.Errorf("invalid ACL network: %s", acl.Network)
		}
		switch acl.Action {
		case "allow", "deny":
		default:
			return fmt.Errorf("invalid ACL action: %s", acl.Action)
		}
	}

	// Validate certificate rotation
	if cfg.CertRotation.Enabled {
		if cfg.CertRotation.Interval <= 0 {
			return fmt.Errorf("invalid certificate rotation interval: %v", cfg.CertRotation.Interval)
		}
	}

	return nil
}
