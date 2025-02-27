package unit

import (
	"errors"
	"testing"
	"time"

	"github.com/o3willard-AI/SSSonector/internal/config/types"
	"github.com/o3willard-AI/SSSonector/internal/startup"
	"github.com/o3willard-AI/SSSonector/test/startup_logging/helpers"
)

func TestErrorHandling(t *testing.T) {
	tests := []struct {
		name          string
		phase         types.StartupPhase
		component     types.StartupComponent
		operation     string
		injectError   error
		details       map[string]interface{}
		expectCleanup bool
	}{
		{
			name:        "TUN Interface Creation Failure",
			phase:       types.StartupPhaseInitialization,
			component:   types.StartupComponentAdapter,
			operation:   "Create TUN adapter",
			injectError: errors.New("failed to create TUN interface: permission denied"),
			details: map[string]interface{}{
				"interface": "tun0",
				"mtu":       1500,
				"address":   "10.0.0.1",
			},
			expectCleanup: true,
		},
		{
			name:        "Certificate Loading Failure",
			phase:       types.StartupPhaseInitialization,
			component:   types.StartupComponentSecurity,
			operation:   "Load certificates",
			injectError: errors.New("failed to load certificate: file not found"),
			details: map[string]interface{}{
				"cert_file": "/etc/sssonector/certs/server.crt",
				"key_file":  "/etc/sssonector/certs/server.key",
			},
			expectCleanup: false,
		},
		{
			name:        "Network Connection Failure",
			phase:       types.StartupPhaseConnection,
			component:   types.StartupComponentConnection,
			operation:   "Connect to server",
			injectError: errors.New("connection failed: timeout"),
			details: map[string]interface{}{
				"address": "192.168.50.210:8443",
				"timeout": "30s",
			},
			expectCleanup: true,
		},
		{
			name:        "Invalid Configuration",
			phase:       types.StartupPhasePreStartup,
			component:   types.StartupComponentConfig,
			operation:   "Validate configuration",
			injectError: errors.New("invalid configuration: missing required field 'address'"),
			details: map[string]interface{}{
				"config_file": "/etc/sssonector/config.yaml",
			},
			expectCleanup: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			testLogger, logger := helpers.NewTestLoggerWithCapture()
			startupLogger := startup.NewStartupLogger(logger, helpers.CreateTestConfig())

			// Set up the phase sequence correctly
			switch tt.phase {
			case types.StartupPhaseInitialization:
				_ = startupLogger.SetPhase(types.StartupPhasePreStartup)
				_ = startupLogger.SetPhase(types.StartupPhaseInitialization)
			case types.StartupPhaseConnection:
				_ = startupLogger.SetPhase(types.StartupPhasePreStartup)
				_ = startupLogger.SetPhase(types.StartupPhaseInitialization)
				_ = startupLogger.SetPhase(types.StartupPhaseConnection)
			case types.StartupPhaseListen:
				_ = startupLogger.SetPhase(types.StartupPhasePreStartup)
				_ = startupLogger.SetPhase(types.StartupPhaseInitialization)
				_ = startupLogger.SetPhase(types.StartupPhaseConnection)
				_ = startupLogger.SetPhase(types.StartupPhaseListen)
			default:
				_ = startupLogger.SetPhase(tt.phase)
			}

			// Execute operation that will fail
			err := startupLogger.LogOperation(
				tt.component,
				tt.operation,
				func() error {
					return tt.injectError
				},
				tt.details,
			)

			// Verify error was returned
			if err == nil {
				t.Error("Expected error but got nil")
			}

			// Get log entries
			entries := testLogger.GetEntries()

			// Find the error entry
			var errorEntry *helpers.LogEntry
			for _, entry := range entries {
				if entry.Error != "" {
					errorEntry = &entry
					break
				}
			}

			// Verify error was logged
			if errorEntry == nil {
				t.Error("No error entry found in logs")
				return
			}

			// Verify error details
			if errorEntry.Phase != tt.phase {
				t.Errorf("Expected phase %s, got %s", tt.phase, errorEntry.Phase)
			}
			if errorEntry.Component != tt.component {
				t.Errorf("Expected component %s, got %s", tt.component, errorEntry.Component)
			}
			if errorEntry.Operation != tt.operation {
				t.Errorf("Expected operation %s, got %s", tt.operation, errorEntry.Operation)
			}
			if errorEntry.Error != tt.injectError.Error() {
				t.Errorf("Expected error %s, got %s", tt.injectError.Error(), errorEntry.Error)
			}
			if errorEntry.Status != "Failed" {
				t.Errorf("Expected status Failed, got %s", errorEntry.Status)
			}

			// Verify details were logged
			for k, v := range tt.details {
				if errorEntry.Details[k] != v {
					t.Errorf("Expected detail %s=%v, got %v", k, v, errorEntry.Details[k])
				}
			}

			// Verify timing information
			if errorEntry.Duration.Duration <= 0 {
				t.Error("Expected duration to be set")
			}

			// Verify cleanup for operations that require it
			if tt.expectCleanup {
				var cleanupEntry *helpers.LogEntry
				for _, entry := range entries {
					if entry.Operation == "Cleanup" {
						cleanupEntry = &entry
						break
					}
				}
				if cleanupEntry == nil {
					t.Error("Expected cleanup entry but found none")
				}
			}
		})
	}
}

func TestErrorRecovery(t *testing.T) {
	testLogger, logger := helpers.NewTestLoggerWithCapture()
	startupLogger := startup.NewStartupLogger(logger, helpers.CreateTestConfig())

	// Set initial phase
	startupLogger.SetPhase(types.StartupPhaseInitialization)

	// Simulate a failed operation
	_ = startupLogger.LogOperation(
		types.StartupComponentAdapter,
		"Create TUN adapter",
		func() error {
			return errors.New("adapter creation failed")
		},
		map[string]interface{}{
			"interface": "tun0",
		},
	)

	// Attempt recovery
	err := startupLogger.LogOperation(
		types.StartupComponentAdapter,
		"Retry adapter creation",
		func() error {
			time.Sleep(100 * time.Millisecond) // Simulate work
			return nil
		},
		map[string]interface{}{
			"attempt":   2,
			"interface": "tun0",
		},
	)

	if err != nil {
		t.Errorf("Recovery failed: %v", err)
	}

	entries := testLogger.GetEntries()

	// Verify error was logged
	var errorFound bool
	for _, entry := range entries {
		if entry.Error != "" {
			errorFound = true
			break
		}
	}
	if !errorFound {
		t.Error("No error entry found for initial failure")
	}

	// Verify recovery was logged
	var recoveryFound bool
	for _, entry := range entries {
		if entry.Operation == "Retry adapter creation" && entry.Status == "Success" {
			recoveryFound = true
			break
		}
	}
	if !recoveryFound {
		t.Error("No recovery entry found")
	}
}

func TestConcurrentErrorHandling(t *testing.T) {
	testLogger, logger := helpers.NewTestLoggerWithCapture()
	startupLogger := startup.NewStartupLogger(logger, helpers.CreateTestConfig())

	// Set phase
	startupLogger.SetPhase(types.StartupPhaseConnection)

	// Run multiple operations concurrently
	done := make(chan bool)
	for i := 0; i < 5; i++ {
		go func(id int) {
			defer func() { done <- true }()

			// Simulate operations that may fail
			_ = startupLogger.LogOperation(
				types.StartupComponentConnection,
				"Concurrent operation",
				func() error {
					if id%2 == 0 {
						return errors.New("simulated failure")
					}
					time.Sleep(50 * time.Millisecond)
					return nil
				},
				map[string]interface{}{
					"operation_id": id,
				},
			)
		}(i)
	}

	// Wait for all operations
	for i := 0; i < 5; i++ {
		<-done
	}

	entries := testLogger.GetEntries()

	// Count successes and failures
	successes := 0
	failures := 0
	for _, entry := range entries {
		if entry.Operation == "Concurrent operation" {
			if entry.Error != "" {
				failures++
			} else {
				successes++
			}
		}
	}

	// Verify expected counts
	if successes != 2 {
		t.Errorf("Expected 2 successful operations, got %d", successes)
	}
	if failures != 3 {
		t.Errorf("Expected 3 failed operations, got %d", failures)
	}

	// Verify all operations were logged
	if successes+failures != 5 {
		t.Errorf("Expected 5 total operations, got %d", successes+failures)
	}
}
