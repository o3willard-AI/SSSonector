package adapter

import "fmt"

// NetworkConfig holds network interface configuration
type NetworkConfig struct {
	Interface string
	Address   string
	MTU       int
}

// validateInterfaceConfig validates the interface configuration
func validateInterfaceConfig(cfg *NetworkConfig) error {
	if cfg.Address == "" {
		return fmt.Errorf("interface address is required")
	}
	if cfg.Interface == "" {
		return fmt.Errorf("interface name is required")
	}
	return nil
}
