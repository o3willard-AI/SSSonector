package network

import (
	"fmt"
	"os"
	"strconv"
	"strings"

	"go.uber.org/zap"
)

const (
	// IPForwardingPath is the path to the IP forwarding sysctl setting
	IPForwardingPath = "/proc/sys/net/ipv4/ip_forward"
)

// CheckIPForwarding checks if IP forwarding is enabled
func CheckIPForwarding(logger *zap.Logger) (bool, error) {
	// Read the current IP forwarding setting
	data, err := os.ReadFile(IPForwardingPath)
	if err != nil {
		if logger != nil {
			logger.Error("Failed to read IP forwarding setting",
				zap.Error(err),
				zap.String("path", IPForwardingPath),
			)
		}
		return false, fmt.Errorf("failed to read IP forwarding setting: %w", err)
	}

	// Parse the value
	value := strings.TrimSpace(string(data))
	enabled, err := strconv.ParseBool(value)
	if err != nil {
		if logger != nil {
			logger.Error("Failed to parse IP forwarding value",
				zap.Error(err),
				zap.String("value", value),
			)
		}
		return false, fmt.Errorf("failed to parse IP forwarding value: %w", err)
	}

	if logger != nil {
		logger.Debug("IP forwarding status",
			zap.Bool("enabled", enabled),
		)
	}

	return enabled, nil
}

// EnableIPForwarding enables IP forwarding
func EnableIPForwarding(logger *zap.Logger) error {
	// Check if IP forwarding is already enabled
	enabled, err := CheckIPForwarding(logger)
	if err != nil {
		return err
	}

	if enabled {
		if logger != nil {
			logger.Debug("IP forwarding is already enabled")
		}
		return nil
	}

	// Enable IP forwarding
	if logger != nil {
		logger.Info("Enabling IP forwarding")
	}

	// Write "1" to the IP forwarding setting
	err = os.WriteFile(IPForwardingPath, []byte("1\n"), 0644)
	if err != nil {
		if logger != nil {
			logger.Error("Failed to enable IP forwarding",
				zap.Error(err),
				zap.String("path", IPForwardingPath),
			)
		}
		return fmt.Errorf("failed to enable IP forwarding: %w", err)
	}

	// Verify that IP forwarding is now enabled
	enabled, err = CheckIPForwarding(logger)
	if err != nil {
		return err
	}

	if !enabled {
		if logger != nil {
			logger.Error("IP forwarding is still disabled after enabling attempt")
		}
		return fmt.Errorf("IP forwarding is still disabled after enabling attempt")
	}

	if logger != nil {
		logger.Info("IP forwarding enabled successfully")
	}

	return nil
}

// DisableIPForwarding disables IP forwarding
func DisableIPForwarding(logger *zap.Logger) error {
	// Check if IP forwarding is already disabled
	enabled, err := CheckIPForwarding(logger)
	if err != nil {
		return err
	}

	if !enabled {
		if logger != nil {
			logger.Debug("IP forwarding is already disabled")
		}
		return nil
	}

	// Disable IP forwarding
	if logger != nil {
		logger.Info("Disabling IP forwarding")
	}

	// Write "0" to the IP forwarding setting
	err = os.WriteFile(IPForwardingPath, []byte("0\n"), 0644)
	if err != nil {
		if logger != nil {
			logger.Error("Failed to disable IP forwarding",
				zap.Error(err),
				zap.String("path", IPForwardingPath),
			)
		}
		return fmt.Errorf("failed to disable IP forwarding: %w", err)
	}

	// Verify that IP forwarding is now disabled
	enabled, err = CheckIPForwarding(logger)
	if err != nil {
		return err
	}

	if enabled {
		if logger != nil {
			logger.Error("IP forwarding is still enabled after disabling attempt")
		}
		return fmt.Errorf("IP forwarding is still enabled after disabling attempt")
	}

	if logger != nil {
		logger.Info("IP forwarding disabled successfully")
	}

	return nil
}

// SetupIPForwarding sets up IP forwarding based on the provided enable flag
func SetupIPForwarding(enable bool, logger *zap.Logger) error {
	if enable {
		return EnableIPForwarding(logger)
	}
	return DisableIPForwarding(logger)
}
