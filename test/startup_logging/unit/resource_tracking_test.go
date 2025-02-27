package unit

import (
	"testing"
	"time"

	"github.com/o3willard-AI/SSSonector/internal/config/types"
	"github.com/o3willard-AI/SSSonector/internal/startup"
	"github.com/o3willard-AI/SSSonector/test/startup_logging/helpers"
)

func TestResourceStateTracking(t *testing.T) {
	tests := []struct {
		name           string
		phase          types.StartupPhase
		component      types.StartupComponent
		resource       string
		initialState   map[string]interface{}
		operations     []string
		finalState     map[string]interface{}
		expectedStates []string
	}{
		{
			name:      "TUN Adapter Lifecycle",
			phase:     types.StartupPhaseInitialization,
			component: types.StartupComponentAdapter,
			resource:  "adapter",
			initialState: map[string]interface{}{
				"state":     "uninitialized",
				"interface": "tun0",
				"address":   "10.0.0.1",
			},
			operations: []string{
				"Create TUN adapter",
				"Configure adapter",
				"Start adapter",
			},
			finalState: map[string]interface{}{
				"state":     "ready",
				"interface": "tun0",
				"address":   "10.0.0.1",
				"mtu":       1500,
				"uptime":    "1s",
			},
			expectedStates: []string{
				"uninitialized",
				"initializing",
				"configuring",
				"ready",
			},
		},
		{
			name:      "Certificate Resource Lifecycle",
			phase:     types.StartupPhaseInitialization,
			component: types.StartupComponentSecurity,
			resource:  "certificates",
			initialState: map[string]interface{}{
				"state": "unloaded",
				"path":  "/etc/sssonector/certs",
			},
			operations: []string{
				"Load certificates",
				"Verify certificates",
			},
			finalState: map[string]interface{}{
				"state":    "loaded",
				"path":     "/etc/sssonector/certs",
				"valid":    true,
				"expires":  "2026-02-22T00:00:00Z",
				"subjects": []string{"server.sssonector.local"},
			},
			expectedStates: []string{
				"unloaded",
				"loading",
				"verifying",
				"loaded",
			},
		},
		{
			name:      "Network Connection Lifecycle",
			phase:     types.StartupPhaseConnection,
			component: types.StartupComponentConnection,
			resource:  "connection",
			initialState: map[string]interface{}{
				"state":   "disconnected",
				"address": "192.168.50.210:8443",
			},
			operations: []string{
				"Initialize connection",
				"Establish connection",
				"Verify connection",
			},
			finalState: map[string]interface{}{
				"state":    "connected",
				"address":  "192.168.50.210:8443",
				"latency":  "10ms",
				"protocol": "tcp",
				"secured":  true,
			},
			expectedStates: []string{
				"disconnected",
				"initializing",
				"connecting",
				"verifying",
				"connected",
			},
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

			// Log initial state
			startupLogger.LogResourceState(tt.resource, tt.initialState)

			// Execute operations and log state transitions
			for i, op := range tt.operations {
				// Log state before operation
				state := tt.expectedStates[i+1] // Skip initial state
				startupLogger.LogResourceState(tt.resource, map[string]interface{}{
					"state": state,
				})

				// Execute operation
				err := startupLogger.LogOperation(
					tt.component,
					op,
					func() error {
						time.Sleep(10 * time.Millisecond) // Simulate work
						return nil
					},
					map[string]interface{}{
						"resource": tt.resource,
						"state":    state,
					},
				)
				if err != nil {
					t.Fatalf("Operation failed: %v", err)
				}
			}

			// Log final state
			startupLogger.LogResourceState(tt.resource, tt.finalState)

			// Get log entries
			entries := testLogger.GetEntries()

			// Verify state transitions
			var states []string
			for _, entry := range entries {
				if state, ok := entry.Details["state"].(string); ok {
					states = append(states, state)
				}
			}

			// Verify all expected states were logged
			for _, expectedState := range tt.expectedStates {
				found := false
				for _, state := range states {
					if state == expectedState {
						found = true
						break
					}
				}
				if !found {
					t.Errorf("Expected state %s not found in log entries", expectedState)
				}
			}

			// Verify final state
			var finalEntry *helpers.LogEntry
			for i := len(entries) - 1; i >= 0; i-- {
				if entries[i].Details["state"] != nil {
					finalEntry = &entries[i]
					break
				}
			}

			if finalEntry == nil {
				t.Fatal("No final state entry found")
			}

			// Verify all final state fields
			for k, v := range tt.finalState {
				if k == "subjects" {
					// Compare string slices
					expected := v.([]string)
					actual, ok := finalEntry.Details[k].([]string)
					if !ok {
						t.Errorf("Expected subjects to be []string, got %T", finalEntry.Details[k])
						continue
					}
					if len(expected) != len(actual) {
						t.Errorf("Subjects length mismatch: expected %d, got %d", len(expected), len(actual))
						continue
					}
					for i := range expected {
						if expected[i] != actual[i] {
							t.Errorf("Subject at index %d: expected %s, got %s", i, expected[i], actual[i])
						}
					}
				} else {
					if finalEntry.Details[k] != v {
						t.Errorf("Final state mismatch for %s: expected %v, got %v", k, v, finalEntry.Details[k])
					}
				}
			}
		})
	}
}

func TestResourceCleanup(t *testing.T) {
	tests := []struct {
		name     string
		resource string
		setup    func(*startup.StartupLogger) error
		cleanup  func(*startup.StartupLogger) error
		verify   func([]helpers.LogEntry) error
	}{
		{
			name:     "TUN Adapter Cleanup",
			resource: "adapter",
			setup: func(sl *startup.StartupLogger) error {
				_ = sl.SetPhase(types.StartupPhasePreStartup)
				_ = sl.SetPhase(types.StartupPhaseInitialization)
				return sl.LogOperation(
					types.StartupComponentAdapter,
					"Create TUN adapter",
					func() error {
						time.Sleep(10 * time.Millisecond)
						return nil
					},
					map[string]interface{}{
						"interface": "tun0",
						"state":     "ready",
					},
				)
			},
			cleanup: func(sl *startup.StartupLogger) error {
				return sl.LogOperation(
					types.StartupComponentAdapter,
					"Cleanup adapter",
					func() error {
						time.Sleep(10 * time.Millisecond)
						return nil
					},
					map[string]interface{}{
						"interface": "tun0",
						"state":     "cleaned",
					},
				)
			},
			verify: func(entries []helpers.LogEntry) error {
				var found bool
				for _, entry := range entries {
					if entry.Operation == "Cleanup adapter" &&
						entry.Status == "Success" &&
						entry.Details["state"] == "cleaned" {
						found = true
						break
					}
				}
				if !found {
					t.Error("Cleanup operation not found or unsuccessful")
				}
				return nil
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			testLogger, logger := helpers.NewTestLoggerWithCapture()
			startupLogger := startup.NewStartupLogger(logger, helpers.CreateTestConfig())

			// Setup
			if err := tt.setup(startupLogger); err != nil {
				t.Fatalf("Setup failed: %v", err)
			}

			// Cleanup
			if err := tt.cleanup(startupLogger); err != nil {
				t.Fatalf("Cleanup failed: %v", err)
			}

			// Verify
			entries := testLogger.GetEntries()
			if err := tt.verify(entries); err != nil {
				t.Errorf("Verification failed: %v", err)
			}
		})
	}
}

func TestConcurrentResourceAccess(t *testing.T) {
	testLogger, logger := helpers.NewTestLoggerWithCapture()
	startupLogger := startup.NewStartupLogger(logger, helpers.CreateTestConfig())

	// Set phase
	_ = startupLogger.SetPhase(types.StartupPhasePreStartup)
	_ = startupLogger.SetPhase(types.StartupPhaseInitialization)

	// Run concurrent resource operations
	done := make(chan bool)
	for i := 0; i < 5; i++ {
		go func(id int) {
			defer func() { done <- true }()

			// Log resource state
			startupLogger.LogResourceState("shared_resource", map[string]interface{}{
				"accessor_id": id,
				"state":       "accessing",
			})

			// Simulate work
			time.Sleep(10 * time.Millisecond)

			// Log final state
			startupLogger.LogResourceState("shared_resource", map[string]interface{}{
				"accessor_id": id,
				"state":       "done",
			})
		}(i)
	}

	// Wait for all operations
	for i := 0; i < 5; i++ {
		<-done
	}

	entries := testLogger.GetEntries()

	// Verify all operations were logged
	accessCount := 0
	doneCount := 0
	for _, entry := range entries {
		if state, ok := entry.Details["state"].(string); ok {
			switch state {
			case "accessing":
				accessCount++
			case "done":
				doneCount++
			}
		}
	}

	if accessCount != 5 {
		t.Errorf("Expected 5 access operations, got %d", accessCount)
	}
	if doneCount != 5 {
		t.Errorf("Expected 5 done operations, got %d", doneCount)
	}
}
