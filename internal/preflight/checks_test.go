package preflight

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"go.uber.org/zap/zaptest"
)

func TestPreFlightChecks(t *testing.T) {
	logger := zaptest.NewLogger(t)
	ctx := context.Background()

	t.Run("system requirements check", func(t *testing.T) {
		result := SystemRequirements(ctx, logger)
		assert.Equal(t, "System Requirements", result.Name)
		if result.Status == "Failed" {
			t.Logf("System requirements check failed: %v", result.Error)
		}
	})

	t.Run("file permissions check", func(t *testing.T) {
		result := FilePermissions(ctx, logger)
		assert.Equal(t, "File Permissions", result.Name)
		if result.Status == "Failed" {
			t.Logf("File permissions check failed: %v", result.Error)
		}
	})

	t.Run("network connectivity check", func(t *testing.T) {
		result := NetworkConnectivity(ctx, logger)
		assert.Equal(t, "Network Connectivity", result.Name)
		if result.Status == "Failed" {
			t.Logf("Network connectivity check failed: %v", result.Error)
		}
	})

	t.Run("security checks", func(t *testing.T) {
		result := SecurityChecks(ctx, logger)
		assert.Equal(t, "Security Checks", result.Name)
		if result.Status == "Failed" {
			t.Logf("Security check failed: %v", result.Error)
		}
	})

	t.Run("dependency checks", func(t *testing.T) {
		result := DependencyChecks(ctx, logger)
		assert.Equal(t, "Dependencies", result.Name)
		if result.Status == "Failed" {
			t.Logf("Dependency check failed: %v", result.Error)
		}
	})

	t.Run("run all checks", func(t *testing.T) {
		results, err := RunPreFlightChecks(ctx, logger)
		require.NoError(t, err)
		assert.Len(t, results, 6) // Updated to include config validation check

		// Count passed checks
		passed := 0
		for _, result := range results {
			if result.Status == "Passed" {
				passed++
			} else {
				t.Logf("Check failed: %s - %v", result.Name, result.Error)
			}
		}
		t.Logf("%d out of %d checks passed", passed, len(results))
	})

	t.Run("context cancellation", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		cancel() // Cancel immediately

		results, err := RunPreFlightChecks(ctx, logger)
		assert.Error(t, err)
		assert.Equal(t, context.Canceled, err)
		assert.Empty(t, results)
	})
}

func TestCheckResult_String(t *testing.T) {
	result := CheckResult{
		Name:        "Test Check",
		Status:      "Passed",
		Description: "Test description",
		Error:       nil,
	}

	expected := "Test Check: Passed - Test description"
	assert.Equal(t, expected, result.String())

	result.Status = "Failed"
	result.Error = assert.AnError
	expected = "Test Check: Failed - Test description: assert.AnError general error for testing"
	assert.Equal(t, expected, result.String())
}

type mockCheck struct {
	name        string
	status      string
	description string
	err         error
}

func (m mockCheck) Check(ctx context.Context, logger *zap.Logger) CheckResult {
	return CheckResult{
		Name:        m.name,
		Status:      m.status,
		Description: m.description,
		Error:       m.err,
	}
}

func TestMockChecks(t *testing.T) {
	logger := zaptest.NewLogger(t)
	ctx := context.Background()

	t.Run("mock system requirements", func(t *testing.T) {
		check := mockCheck{
			name:        "Mock System Requirements",
			status:      "Passed",
			description: "Mock check",
		}

		result := check.Check(ctx, logger)
		assert.Equal(t, "Passed", result.Status)
		assert.NoError(t, result.Error)
	})

	t.Run("mock failed check", func(t *testing.T) {
		check := mockCheck{
			name:        "Mock Failed Check",
			status:      "Failed",
			description: "Mock check",
			err:         assert.AnError,
		}

		result := check.Check(ctx, logger)
		assert.Equal(t, "Failed", result.Status)
		assert.Error(t, result.Error)
	})
}
