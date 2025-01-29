package config

import (
	"fmt"

	"github.com/spf13/viper"
)

// Config holds the complete application configuration
type Config struct {
	// Mode specifies whether the application runs in client or server mode
	Mode string `mapstructure:"mode"`

	// Network contains network interface configuration
	Network *NetworkConfig `mapstructure:"network"`

	// Tunnel contains SSL tunnel configuration
	Tunnel *TunnelConfig `mapstructure:"tunnel"`

	// Monitor contains monitoring configuration
	Monitor *MonitorConfig `mapstructure:"monitor"`
}

// TunnelConfig holds SSL tunnel configuration
type TunnelConfig struct {
	// CertFile is the path to the SSL certificate file
	CertFile string `mapstructure:"cert_file"`

	// KeyFile is the path to the SSL private key file
	KeyFile string `mapstructure:"key_file"`

	// CAFile is the path to the CA certificate file for client verification
	CAFile string `mapstructure:"ca_file"`

	// ListenAddress is the address to listen on (server mode)
	ListenAddress string `mapstructure:"listen_address"`

	// ListenPort is the port to listen on (server mode)
	ListenPort int `mapstructure:"listen_port"`

	// ServerAddress is the address to connect to (client mode)
	ServerAddress string `mapstructure:"server_address"`

	// ServerPort is the port to connect to (client mode)
	ServerPort int `mapstructure:"server_port"`

	// MaxClients is the maximum number of concurrent clients (server mode)
	MaxClients int `mapstructure:"max_clients"`

	// RetryAttempts is the number of connection retry attempts (client mode)
	RetryAttempts int `mapstructure:"retry_attempts"`

	// RetryInterval is the interval between retry attempts in seconds (client mode)
	RetryInterval int `mapstructure:"retry_interval"`

	// BandwidthLimit is the maximum bandwidth in bytes per second (0 for unlimited)
	BandwidthLimit int64 `mapstructure:"bandwidth_limit"`
}

// MonitorConfig holds monitoring configuration
type MonitorConfig struct {
	// LogFile is the path to the log file
	LogFile string `mapstructure:"log_file"`

	// LogLevel is the minimum log level to record
	LogLevel string `mapstructure:"log_level"`

	// SNMPEnabled enables SNMP monitoring
	SNMPEnabled bool `mapstructure:"snmp_enabled"`

	// SNMPAddress is the address to listen for SNMP requests
	SNMPAddress string `mapstructure:"snmp_address"`

	// SNMPPort is the port to listen for SNMP requests
	SNMPPort int `mapstructure:"snmp_port"`

	// SNMPCommunity is the SNMP community string
	SNMPCommunity string `mapstructure:"snmp_community"`
}

// LoadConfig loads the configuration from the specified file
func LoadConfig(file string) (*Config, error) {
	viper.SetConfigFile(file)
	viper.SetConfigType("yaml")

	if err := viper.ReadInConfig(); err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	var config Config
	if err := viper.Unmarshal(&config); err != nil {
		return nil, fmt.Errorf("failed to unmarshal config: %w", err)
	}

	if err := validateConfig(&config); err != nil {
		return nil, fmt.Errorf("invalid configuration: %w", err)
	}

	return &config, nil
}

// validateConfig validates the configuration
func validateConfig(cfg *Config) error {
	if cfg.Mode != "client" && cfg.Mode != "server" {
		return fmt.Errorf("invalid mode: %s (must be 'client' or 'server')", cfg.Mode)
	}

	if cfg.Network == nil {
		return fmt.Errorf("network configuration is required")
	}

	if cfg.Tunnel == nil {
		return fmt.Errorf("tunnel configuration is required")
	}

	if cfg.Mode == "server" {
		if cfg.Tunnel.ListenAddress == "" {
			return fmt.Errorf("listen_address is required in server mode")
		}
		if cfg.Tunnel.ListenPort == 0 {
			return fmt.Errorf("listen_port is required in server mode")
		}
	} else {
		if cfg.Tunnel.ServerAddress == "" {
			return fmt.Errorf("server_address is required in client mode")
		}
		if cfg.Tunnel.ServerPort == 0 {
			return fmt.Errorf("server_port is required in client mode")
		}
	}

	return nil
}
