package preflight

import (
	"log"
	"os"
	"sort"
	"testing"
	"time"

	"github.com/o3willard-AI/SSSonector/internal/errors"
)

func TestChecker(t *testing.T) {
	// Create test logger
	logger := log.New(os.Stderr, "[TEST] ", log.LstdFlags)

	// Create test error handler
	errorHandler := errors.NewHandler(logger)

	// Create test options
	opts := &Options{
		ConfigPath: "/tmp/test-config.yaml",
		Mode:       "server",
		Fix:        false,
		Format:     "text",
		Quiet:      false,
		Verbose:    true,
	}

	// Create test config file
	testConfig := []byte(`
mode: server
network:
  interface: tun0
  address: 10.0.0.1/24
tunnel:
  port: 8443
  cert: /etc/sssonector/certs/server.crt
  key: /etc/sssonector/certs/server.key
`)
	if err := os.WriteFile(opts.ConfigPath, testConfig, 0640); err != nil {
		t.Fatalf("Failed to create test config: %v", err)
	}
	defer os.Remove(opts.ConfigPath)

	// Create checker
	checker := NewChecker(opts, errorHandler)

	// Run checks with timeout
	done := make(chan struct{})
	var report *Report
	var err error

	go func() {
		report, err = checker.RunChecks()
		close(done)
	}()

	select {
	case <-done:
		// Test completed normally
	case <-time.After(5 * time.Second):
		t.Fatal("Test timed out")
	}

	// Check for errors
	if err != nil {
		t.Fatalf("RunChecks failed: %v", err)
	}

	// Verify report
	if report == nil {
		t.Fatal("Expected report, got nil")
	}

	// Check report fields
	if report.System == "" {
		t.Error("Expected system info, got empty string")
	}
	if report.Version == "" {
		t.Error("Expected version info, got empty string")
	}
	if report.Timestamp.IsZero() {
		t.Error("Expected timestamp, got zero time")
	}

	// Check results
	if len(report.Results) == 0 {
		t.Error("Expected results, got empty slice")
	}

	// Verify each result has required fields
	for _, result := range report.Results {
		if result.Name == "" {
			t.Error("Expected result name, got empty string")
		}
		if result.Status == "" {
			t.Error("Expected result status, got empty string")
		}
		if result.Duration == 0 {
			t.Error("Expected non-zero duration")
		}
	}

	// Test summary calculation
	if report.Summary.Total != len(report.Results) {
		t.Errorf("Expected total %d, got %d", len(report.Results), report.Summary.Total)
	}

	// Test fix mode
	opts.Fix = true
	checker = NewChecker(opts, errorHandler)
	report, err = checker.RunChecks()
	if err != nil {
		t.Fatalf("RunChecks with fix mode failed: %v", err)
	}

	// Test specific validator
	opts.CheckOnly = "Configuration"
	checker = NewChecker(opts, errorHandler)
	report, err = checker.RunChecks()
	if err != nil {
		t.Fatalf("RunChecks with specific validator failed: %v", err)
	}
	if len(report.Results) != 1 {
		t.Errorf("Expected 1 result, got %d", len(report.Results))
	}
	if report.Results[0].Name != "Configuration" {
		t.Errorf("Expected Configuration check, got %s", report.Results[0].Name)
	}
}

func TestReport(t *testing.T) {
	report := &Report{
		Timestamp: time.Now(),
		System:    "Test System",
		Version:   "1.0.0",
		Results: []Result{
			{
				Name:        "Test Check",
				Status:      StatusPass,
				Details:     "Test details",
				Duration:    time.Second,
				FixPossible: false,
			},
		},
		Summary: Summary{
			Total:        1,
			Passed:       1,
			Failed:       0,
			Warnings:     0,
			FixableCount: 0,
		},
	}

	// Test text format
	if err := report.Print("text"); err != nil {
		t.Errorf("Print text format failed: %v", err)
	}

	// Test JSON format
	if err := report.Print("json"); err != nil {
		t.Errorf("Print JSON format failed: %v", err)
	}

	// Test Markdown format
	if err := report.Print("markdown"); err != nil {
		t.Errorf("Print Markdown format failed: %v", err)
	}

	// Test invalid format
	if err := report.Print("invalid"); err == nil {
		t.Error("Expected error for invalid format, got nil")
	}
}

func TestValidatorPriority(t *testing.T) {
	// Create test validators with different priorities
	validators := []Validator{
		&SecurityValidator{BaseValidator: BaseValidator{name: "Security", priority: 4}},
		&ConfigValidator{BaseValidator: BaseValidator{name: "Config", priority: 0}},
		&SystemValidator{BaseValidator: BaseValidator{name: "System", priority: 1}},
		&ResourceValidator{BaseValidator: BaseValidator{name: "Resource", priority: 2}},
		&NetworkValidator{BaseValidator: BaseValidator{name: "Network", priority: 3}},
	}

	// Sort validators
	sort.Slice(validators, func(i, j int) bool {
		return validators[i].Priority() < validators[j].Priority()
	})

	// Verify order
	expectedOrder := []string{"Config", "System", "Resource", "Network", "Security"}
	for i, v := range validators {
		if v.Name() != expectedOrder[i] {
			t.Errorf("Expected validator %s at position %d, got %s", expectedOrder[i], i, v.Name())
		}
	}
}
