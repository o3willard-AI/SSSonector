package unit

import (
	"strings"
	"testing"
	"time"

	"github.com/o3willard-AI/SSSonector/internal/config/types"
	"github.com/o3willard-AI/SSSonector/internal/startup"
	"github.com/o3willard-AI/SSSonector/test/startup_logging/helpers"
)

func TestPhaseTransitions(t *testing.T) {
	tests := []struct {
		name          string
		phases        []types.StartupPhase
		expectedOrder bool
		expectedTime  time.Duration
	}{
		{
			name: "Normal Startup Sequence",
			phases: []types.StartupPhase{
				types.StartupPhasePreStartup,
				types.StartupPhaseInitialization,
				types.StartupPhaseConnection,
				types.StartupPhaseListen,
			},
			expectedOrder: true,
			expectedTime:  5 * time.Second,
		},
		{
			name: "Client Startup Sequence",
			phases: []types.StartupPhase{
				types.StartupPhasePreStartup,
				types.StartupPhaseInitialization,
				types.StartupPhaseConnection,
			},
			expectedOrder: true,
			expectedTime:  3 * time.Second,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			testLogger, logger := helpers.NewTestLoggerWithCapture()
			startupLogger := startup.NewStartupLogger(logger, helpers.CreateTestConfig())

			// Simulate startup sequence
			for _, phase := range tt.phases {
				err := startupLogger.SetPhase(phase)
				if err != nil {
					t.Fatalf("Failed to set phase %s: %v", phase, err)
				}
				startupLogger.LogOperation(
					types.StartupComponentConfig,
					"Phase transition",
					func() error {
						time.Sleep(100 * time.Millisecond) // Simulate work
						return nil
					},
					map[string]interface{}{
						"phase": string(phase),
					},
				)
			}

			entries := testLogger.GetEntries()

			// Verify phase order
			helpers.AssertPhaseOrder(t, entries, tt.phases)

			// Verify timing
			helpers.AssertDuration(t, entries, "Phase transition", tt.expectedTime)

			// Verify no errors occurred
			helpers.AssertNoErrors(t, entries)
		})
	}
}

func TestPhaseTransitionErrors(t *testing.T) {
	tests := []struct {
		name          string
		initialPhase  types.StartupPhase
		nextPhase     types.StartupPhase
		expectError   bool
		errorContains string
	}{
		{
			name:          "Skip Initialization Phase",
			initialPhase:  types.StartupPhasePreStartup,
			nextPhase:     types.StartupPhaseConnection,
			expectError:   true,
			errorContains: "invalid phase transition",
		},
		{
			name:          "Invalid Initial Phase",
			initialPhase:  types.StartupPhasePreStartup,
			nextPhase:     types.StartupPhaseConnection,
			expectError:   true,
			errorContains: "invalid phase transition",
		},
		{
			name:         "Valid Phase Transition",
			initialPhase: types.StartupPhasePreStartup,
			nextPhase:    types.StartupPhaseInitialization,
			expectError:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			testLogger, logger := helpers.NewTestLoggerWithCapture()
			startupLogger := startup.NewStartupLogger(logger, helpers.CreateTestConfig())

			// Set up initial phase sequence
			if tt.initialPhase != "" {
				// Always start with PreStartup
				err := startupLogger.SetPhase(types.StartupPhasePreStartup)
				if err != nil {
					t.Fatalf("Failed to set PreStartup phase: %v", err)
				}

				// If we need a different initial phase, set it through the proper sequence
				if tt.initialPhase != types.StartupPhasePreStartup {
					switch tt.initialPhase {
					case types.StartupPhaseInitialization:
						err = startupLogger.SetPhase(types.StartupPhaseInitialization)
					case types.StartupPhaseConnection:
						err = startupLogger.SetPhase(types.StartupPhaseInitialization)
						if err == nil {
							err = startupLogger.SetPhase(types.StartupPhaseConnection)
						}
					case types.StartupPhaseListen:
						err = startupLogger.SetPhase(types.StartupPhaseInitialization)
						if err == nil {
							err = startupLogger.SetPhase(types.StartupPhaseConnection)
						}
						if err == nil {
							err = startupLogger.SetPhase(types.StartupPhaseListen)
						}
					}
					if err != nil {
						t.Fatalf("Failed to set initial phase sequence: %v", err)
					}
				}
			}

			// Attempt phase transition
			err := startupLogger.SetPhase(tt.nextPhase)

			// Verify error expectation
			if tt.expectError {
				if err == nil {
					t.Error("Expected error but got none")
				} else if tt.errorContains != "" && !strings.Contains(err.Error(), tt.errorContains) {
					t.Errorf("Expected error containing %q, got %q", tt.errorContains, err.Error())
				}

				// Verify error was logged
				entries := testLogger.GetEntries()
				var foundError bool
				for _, entry := range entries {
					if entry.Error != "" && strings.Contains(entry.Error, tt.errorContains) {
						foundError = true
						break
					}
				}
				if !foundError {
					t.Error("Expected error log entry for invalid phase transition")
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error: %v", err)
				}
			}
		})
	}
}

func TestPhaseDurations(t *testing.T) {
	testLogger, logger := helpers.NewTestLoggerWithCapture()
	startupLogger := startup.NewStartupLogger(logger, helpers.CreateTestConfig())

	phases := []struct {
		phase    types.StartupPhase
		duration time.Duration
	}{
		{types.StartupPhasePreStartup, 100 * time.Millisecond},
		{types.StartupPhaseInitialization, 200 * time.Millisecond},
		{types.StartupPhaseConnection, 300 * time.Millisecond},
	}

	for _, phase := range phases {
		startupLogger.SetPhase(phase.phase)
		startupLogger.LogOperation(
			types.StartupComponentConfig,
			"Phase execution",
			func() error {
				time.Sleep(phase.duration)
				return nil
			},
			map[string]interface{}{
				"expected_duration": phase.duration,
			},
		)
	}

	entries := testLogger.GetEntries()

	// Verify each phase's duration
	for _, phase := range phases {
		for _, entry := range entries {
			if entry.Phase == phase.phase && entry.Operation == "Phase execution" {
				if entry.Duration.Duration < phase.duration {
					t.Errorf("Phase %s: expected duration >= %v, got %v",
						string(phase.phase), phase.duration, entry.Duration.Duration)
				}
				break
			}
		}
	}
}

func TestConcurrentPhaseOperations(t *testing.T) {
	testLogger, logger := helpers.NewTestLoggerWithCapture()
	startupLogger := startup.NewStartupLogger(logger, helpers.CreateTestConfig())

	// Set up initial phase
	err := startupLogger.SetPhase(types.StartupPhasePreStartup)
	if err != nil {
		t.Fatalf("Failed to set initial phase: %v", err)
	}
	err = startupLogger.SetPhase(types.StartupPhaseInitialization)
	if err != nil {
		t.Fatalf("Failed to set initialization phase: %v", err)
	}

	// Run multiple operations concurrently
	done := make(chan bool)
	for i := 0; i < 5; i++ {
		go func(id int) {
			startupLogger.LogOperation(
				types.StartupComponentConfig,
				"Concurrent operation",
				func() error {
					time.Sleep(50 * time.Millisecond)
					return nil
				},
				map[string]interface{}{
					"operation_id": id,
				},
			)
			done <- true
		}(i)
	}

	// Wait for all operations to complete
	for i := 0; i < 5; i++ {
		<-done
	}

	entries := testLogger.GetEntries()

	// Verify all operations were logged
	operationCount := 0
	for _, entry := range entries {
		if entry.Operation == "Concurrent operation" {
			operationCount++
		}
	}

	if operationCount != 5 {
		t.Errorf("Expected 5 concurrent operations, got %d", operationCount)
	}

	// Verify no errors in concurrent execution
	for _, entry := range entries {
		if entry.Error != "" && !strings.Contains(entry.Error, "invalid phase transition") {
			t.Errorf("Unexpected error in log entry: %s", entry.Error)
		}
	}
}

func TestPhaseStateTracking(t *testing.T) {
	testLogger, logger := helpers.NewTestLoggerWithCapture()
	startupLogger := startup.NewStartupLogger(logger, helpers.CreateTestConfig())

	phases := []types.StartupPhase{
		types.StartupPhasePreStartup,
		types.StartupPhaseInitialization,
		types.StartupPhaseConnection,
	}
	expectedStates := []string{"started", "completed", "completed"}

	for i, phase := range phases {
		startupLogger.SetPhase(phase)
		startupLogger.LogOperation(
			types.StartupComponentConfig,
			"Phase state check",
			func() error {
				return nil
			},
			map[string]interface{}{
				"state": expectedStates[i],
			},
		)
	}

	entries := testLogger.GetEntries()

	// Verify state transitions
	for i, phase := range phases {
		found := false
		for _, entry := range entries {
			if entry.Phase == phase && entry.Operation == "Phase state check" {
				state, ok := entry.Details["state"].(string)
				if !ok {
					t.Errorf("Phase %s: state not found in details", string(phase))
					continue
				}
				if state != expectedStates[i] {
					t.Errorf("Phase %s: expected state %s, got %s",
						string(phase), expectedStates[i], state)
				}
				found = true
				break
			}
		}
		if !found {
			t.Errorf("Phase %s: no state tracking entry found", string(phase))
		}
	}
}
