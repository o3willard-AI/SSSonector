//go:build windows
// +build windows

package daemon

import (
	"fmt"

	"golang.org/x/sys/windows/svc"
	"golang.org/x/sys/windows/svc/eventlog"
	"golang.org/x/sys/windows/svc/mgr"
)

// InstallService installs the Windows service
func InstallService(exePath string) error {
	// Open service manager
	m, err := mgr.Connect()
	if err != nil {
		return fmt.Errorf("failed to connect to service manager: %w", err)
	}
	defer m.Disconnect()

	// Check if service already exists
	s, err := m.OpenService(ServiceName)
	if err == nil {
		s.Close()
		return fmt.Errorf("service %s already exists", ServiceName)
	}

	// Create service
	config := mgr.Config{
		DisplayName:      ServiceDisplayName,
		Description:      ServiceDescription,
		StartType:        mgr.StartAutomatic,
		ErrorControl:     mgr.ErrorNormal,
		Dependencies:     []string{"Tcpip", "Dnscache"},
		DelayedAutoStart: true,
	}

	s, err = m.CreateService(ServiceName, exePath, config)
	if err != nil {
		return fmt.Errorf("failed to create service: %w", err)
	}
	defer s.Close()

	// Set up event logging
	if err := eventlog.InstallAsEventCreate(ServiceName, eventlog.Error|eventlog.Warning|eventlog.Info); err != nil {
		s.Delete()
		return fmt.Errorf("failed to set up event logging: %w", err)
	}

	return nil
}

// UninstallService uninstalls the Windows service
func UninstallService() error {
	// Open service manager
	m, err := mgr.Connect()
	if err != nil {
		return fmt.Errorf("failed to connect to service manager: %w", err)
	}
	defer m.Disconnect()

	// Open service
	s, err := m.OpenService(ServiceName)
	if err != nil {
		return fmt.Errorf("service %s is not installed", ServiceName)
	}
	defer s.Close()

	// Remove event logging
	if err := eventlog.Remove(ServiceName); err != nil {
		return fmt.Errorf("failed to remove event logging: %w", err)
	}

	// Delete service
	if err := s.Delete(); err != nil {
		return fmt.Errorf("failed to delete service: %w", err)
	}

	return nil
}

// StartService starts the Windows service
func StartService() error {
	// Open service manager
	m, err := mgr.Connect()
	if err != nil {
		return fmt.Errorf("failed to connect to service manager: %w", err)
	}
	defer m.Disconnect()

	// Open service
	s, err := m.OpenService(ServiceName)
	if err != nil {
		return fmt.Errorf("service %s is not installed", ServiceName)
	}
	defer s.Close()

	// Start service
	if err := s.Start(); err != nil {
		return fmt.Errorf("failed to start service: %w", err)
	}

	return nil
}

// StopService stops the Windows service
func StopService() error {
	// Open service manager
	m, err := mgr.Connect()
	if err != nil {
		return fmt.Errorf("failed to connect to service manager: %w", err)
	}
	defer m.Disconnect()

	// Open service
	s, err := m.OpenService(ServiceName)
	if err != nil {
		return fmt.Errorf("service %s is not installed", ServiceName)
	}
	defer s.Close()

	// Stop service
	if err := s.Control(svc.Stop); err != nil {
		return fmt.Errorf("failed to stop service: %w", err)
	}

	return nil
}

// GetServiceStatus gets the Windows service status
func GetServiceStatus() (string, error) {
	// Open service manager
	m, err := mgr.Connect()
	if err != nil {
		return "", fmt.Errorf("failed to connect to service manager: %w", err)
	}
	defer m.Disconnect()

	// Open service
	s, err := m.OpenService(ServiceName)
	if err != nil {
		return "", fmt.Errorf("service %s is not installed", ServiceName)
	}
	defer s.Close()

	// Get service status
	status, err := s.Query()
	if err != nil {
		return "", fmt.Errorf("failed to query service status: %w", err)
	}

	switch status.State {
	case svc.Running:
		return "Running", nil
	case svc.Stopped:
		return "Stopped", nil
	case svc.StartPending:
		return "Starting", nil
	case svc.StopPending:
		return "Stopping", nil
	case svc.Paused:
		return "Paused", nil
	case svc.PausePending:
		return "Pausing", nil
	case svc.ContinuePending:
		return "Resuming", nil
	default:
		return "Unknown", nil
	}
}

// GetServiceConfig gets the Windows service configuration
func GetServiceConfig() (*mgr.Config, error) {
	// Open service manager
	m, err := mgr.Connect()
	if err != nil {
		return nil, fmt.Errorf("failed to connect to service manager: %w", err)
	}
	defer m.Disconnect()

	// Open service
	s, err := m.OpenService(ServiceName)
	if err != nil {
		return nil, fmt.Errorf("service %s is not installed", ServiceName)
	}
	defer s.Close()

	// Get service config
	config, err := s.Config()
	if err != nil {
		return nil, fmt.Errorf("failed to get service config: %w", err)
	}

	return &config, nil
}

// UpdateServiceConfig updates the Windows service configuration
func UpdateServiceConfig(config *mgr.Config) error {
	// Open service manager
	m, err := mgr.Connect()
	if err != nil {
		return fmt.Errorf("failed to connect to service manager: %w", err)
	}
	defer m.Disconnect()

	// Open service
	s, err := m.OpenService(ServiceName)
	if err != nil {
		return fmt.Errorf("service %s is not installed", ServiceName)
	}
	defer s.Close()

	// Update service config
	if err := s.UpdateConfig(*config); err != nil {
		return fmt.Errorf("failed to update service config: %w", err)
	}

	return nil
}
