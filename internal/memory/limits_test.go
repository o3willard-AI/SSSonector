package memory

import (
	"testing"
	"time"

	"go.uber.org/zap"
)

func TestMemoryLimits(t *testing.T) {
	logger, _ := zap.NewDevelopment()
	manager := NewManager(100, time.Second, logger) // 100MB limit

	// Test memory allocation within limit
	if !manager.CheckAndReserve(50 * MB) {
		t.Error("Failed to allocate memory within limit")
	}

	// Test memory allocation exceeding limit
	if manager.CheckAndReserve(60 * MB) {
		t.Error("Should not allow allocation exceeding limit")
	}

	// Test memory release
	manager.Release(50 * MB)

	// Verify memory is released
	if manager.GetCurrentMemory() != 0 {
		t.Errorf("Expected 0 current memory, got %d", manager.GetCurrentMemory())
	}

	// Test max memory
	if manager.GetMaxMemory() != 100 {
		t.Errorf("Expected max memory 100MB, got %d", manager.GetMaxMemory())
	}
}

func TestCleanupCallback(t *testing.T) {
	logger, _ := zap.NewDevelopment()
	manager := NewManager(50, time.Second, logger)

	cleaned := false
	manager.RegisterCleanup(func() {
		cleaned = true
	})

	// Trigger cleanup by exceeding limit
	manager.CheckAndReserve(40 * MB)
	manager.CheckAndReserve(20 * MB)

	// Wait for cleanup to run
	time.Sleep(2 * time.Second)

	if !cleaned {
		t.Error("Cleanup callback was not executed")
	}
}

func TestMemoryMonitoring(t *testing.T) {
	logger, _ := zap.NewDevelopment()
	manager := NewManager(50, 100*time.Millisecond, logger)

	// Allocate memory
	manager.CheckAndReserve(30 * MB)

	// Wait for monitoring cycle
	time.Sleep(200 * time.Millisecond)

	// Check current memory
	if manager.GetCurrentMemory() != 30 {
		t.Errorf("Expected current memory 30MB, got %d", manager.GetCurrentMemory())
	}

	// Release memory
	manager.Release(30 * MB)

	// Wait for monitoring cycle
	time.Sleep(200 * time.Millisecond)

	// Check memory is released
	if manager.GetCurrentMemory() != 0 {
		t.Errorf("Expected 0 current memory, got %d", manager.GetCurrentMemory())
	}
}

func TestDefaultConfig(t *testing.T) {
	logger, _ := zap.NewDevelopment()
	manager := NewManager(0, 0, logger)

	if manager.GetMaxMemory() != DefaultMaxMemoryMB {
		t.Errorf("Expected default max memory %d, got %d", DefaultMaxMemoryMB, manager.GetMaxMemory())
	}
}

func TestMetricsReset(t *testing.T) {
	logger, _ := zap.NewDevelopment()
	manager := NewManager(512, time.Second, logger)

	// Test allocation
	manager.CheckAndReserve(1 * MB)

	// Test stats before reset
	if manager.GetCurrentMemory() != 1 {
		t.Errorf("Expected current memory 1MB, got %d", manager.GetCurrentMemory())
	}

	// Test allocation exceeding limit
	if manager.CheckAndReserve(2 * 1024 * MB) { // 2GB
		t.Error("Should not allow allocation exceeding limit")
	}
}

func TestFormatStats(t *testing.T) {
	logger, _ := zap.NewDevelopment()
	manager := NewManager(100, time.Second, logger)

	manager.CheckAndReserve(50 * MB)

	stats := manager.GetStats()
	if stats == "" {
		t.Error("Expected non-empty stats string")
	}
}
