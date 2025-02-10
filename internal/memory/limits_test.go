package memory

import (
	"runtime"
	"testing"
	"time"

	"go.uber.org/zap"
)

func TestMemoryLimits(t *testing.T) {
	logger, _ := zap.NewDevelopment()
	config := &LimitConfig{
		MaxMemoryMB:     100,
		CheckInterval:   time.Millisecond * 100,
		CleanupInterval: time.Millisecond * 50,
		SoftLimitRatio:  0.85,
	}

	manager := NewManager(config, logger)
	defer manager.Stop()

	// Test successful allocation
	if !manager.CheckAndReserve(50 * 1024 * 1024) {
		t.Error("Expected allocation to succeed")
	}

	// Test allocation near limit
	if manager.CheckAndReserve(60 * 1024 * 1024) {
		t.Error("Expected allocation to fail")
	}

	// Test release
	manager.Release(30 * 1024 * 1024)
	if !manager.CheckAndReserve(30 * 1024 * 1024) {
		t.Error("Expected allocation to succeed after release")
	}

	// Verify stats
	current, max, cleanups, rejections := manager.GetStats()
	if current <= 0 {
		t.Error("Expected non-zero current memory")
	}
	if max <= 0 {
		t.Error("Expected non-zero max memory")
	}
	if cleanups < 0 {
		t.Error("Expected non-negative cleanup count")
	}
	if rejections != 1 {
		t.Errorf("Expected 1 rejection, got %d", rejections)
	}
}

func TestCleanupCallback(t *testing.T) {
	logger, _ := zap.NewDevelopment()
	config := &LimitConfig{
		MaxMemoryMB:     50,
		CheckInterval:   time.Millisecond * 50,
		CleanupInterval: time.Millisecond * 25,
		SoftLimitRatio:  0.85,
	}

	manager := NewManager(config, logger)
	defer manager.Stop()

	cleanupCalled := false
	manager.RegisterCleanupFunc(func() {
		cleanupCalled = true
	})

	// Allocate memory to trigger cleanup
	data := make([]byte, int(float64(config.MaxMemoryMB*1024*1024)*config.SoftLimitRatio))
	runtime.KeepAlive(data)

	// Wait for cleanup to run
	time.Sleep(config.CleanupInterval * 2)

	if !cleanupCalled {
		t.Error("Expected cleanup function to be called")
	}
}

func TestMemoryMonitoring(t *testing.T) {
	logger, _ := zap.NewDevelopment()
	config := &LimitConfig{
		MaxMemoryMB:     100,
		CheckInterval:   time.Millisecond * 50,
		CleanupInterval: time.Millisecond * 25,
		SoftLimitRatio:  0.85,
	}

	manager := NewManager(config, logger)
	defer manager.Stop()

	// Allocate some memory
	var allocations [][]byte
	for i := 0; i < 5; i++ {
		if manager.CheckAndReserve(10 * 1024 * 1024) {
			data := make([]byte, 10*1024*1024)
			allocations = append(allocations, data)
		}
	}

	// Wait for monitoring to update stats
	time.Sleep(config.CheckInterval * 2)

	current, max, _, _ := manager.GetStats()
	if current < 40*1024*1024 {
		t.Error("Expected higher current memory usage")
	}
	if max < current {
		t.Error("Expected max to be at least current usage")
	}

	// Keep allocations alive
	runtime.KeepAlive(allocations)
}

func TestDefaultConfig(t *testing.T) {
	config := DefaultLimitConfig()

	if config.MaxMemoryMB != defaultMaxMemoryMB {
		t.Errorf("Expected default max memory %d, got %d", defaultMaxMemoryMB, config.MaxMemoryMB)
	}
	if config.CheckInterval != defaultCheckInterval {
		t.Errorf("Expected default check interval %v, got %v", defaultCheckInterval, config.CheckInterval)
	}
	if config.CleanupInterval != defaultCleanupInterval {
		t.Errorf("Expected default cleanup interval %v, got %v", defaultCleanupInterval, config.CleanupInterval)
	}
	if config.SoftLimitRatio != defaultSoftLimitRatio {
		t.Errorf("Expected default soft limit ratio %v, got %v", defaultSoftLimitRatio, config.SoftLimitRatio)
	}
}

func TestMetricsReset(t *testing.T) {
	logger, _ := zap.NewDevelopment()
	manager := NewManager(nil, logger)
	defer manager.Stop()

	// Perform some allocations
	manager.CheckAndReserve(1024 * 1024)
	manager.CheckAndReserve(2 * 1024 * 1024 * 1024) // Should be rejected

	// Verify initial stats
	_, _, _, rejections := manager.GetStats()
	if rejections != 1 {
		t.Errorf("Expected 1 rejection, got %d", rejections)
	}

	// Reset metrics
	manager.stats.Lock()
	manager.stats.current = 0
	manager.stats.max = 0
	manager.stats.cleanups = 0
	manager.stats.rejections = 0
	manager.stats.Unlock()

	// Verify reset
	current, max, cleanups, rejections := manager.GetStats()
	if current != 0 || max != 0 || cleanups != 0 || rejections != 0 {
		t.Error("Expected all metrics to be reset to 0")
	}
}

func TestFormatStats(t *testing.T) {
	logger, _ := zap.NewDevelopment()
	manager := NewManager(nil, logger)
	defer manager.Stop()

	// Set some stats
	manager.stats.Lock()
	manager.stats.current = 100 * 1024 * 1024
	manager.stats.max = 200 * 1024 * 1024
	manager.stats.cleanups = 5
	manager.stats.rejections = 3
	manager.stats.Unlock()

	stats := manager.FormatStats()
	if len(stats) == 0 {
		t.Error("Expected non-empty stats string")
	}
}
