package config

// NetworkConfig holds network interface configuration
type NetworkConfig struct {
	// Interface is the name of the network interface
	Interface string `mapstructure:"interface"`

	// Address is the IP address to assign to the interface
	Address string `mapstructure:"address"`

	// MTU is the Maximum Transmission Unit for the interface
	MTU int `mapstructure:"mtu"`

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
}

// ToAdapterConfig converts the NetworkConfig to adapter.NetworkConfig
func (c *NetworkConfig) ToAdapterConfig() *struct {
	Interface string
	Address   string
	MTU       int
} {
	return &struct {
		Interface string
		Address   string
		MTU       int
	}{
		Interface: c.Interface,
		Address:   c.Address,
		MTU:       c.MTU,
	}
}
