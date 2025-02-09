package throttle

import (
	"testing"
	"time"

	"go.uber.org/zap"
)

func TestDynamicAdjuster(t *testing.T) {
	logger, _ := zap.NewDevelopment()
	adjuster := NewDynamicAdjuster(logger)

	t.Run("Initial Configuration", func(t *testing.T) {
		if !adjuster.enabled {
			t.Error("Expected dynamic adjuster to be enabled by default")
		}
		if adjuster.minRate != minRate {
			t.Errorf("Expected min rate %f, got %f", minRate, adjuster.minRate)
		}
		if adjuster.maxRate != maxRate {
			t.Errorf("Expected max rate %f, got %f", maxRate, adjuster.maxRate)
		}
	})

	t.Run("Configure Parameters", func(t *testing.T) {
		newMinRate := float64(1024 * 1024)      // 1MB/s
		newMaxRate := float64(1024 * 1024 * 50) // 50MB/s
		newInterval := 10 * time.Second

		adjuster.Configure(newMinRate, newMaxRate, newInterval)

		if adjuster.minRate != newMinRate {
			t.Errorf("Expected min rate %f, got %f", newMinRate, adjuster.minRate)
		}
		if adjuster.maxRate != newMaxRate {
			t.Errorf("Expected max rate %f, got %f", newMaxRate, adjuster.maxRate)
		}
		if adjuster.adjustmentInterval != newInterval {
			t.Errorf("Expected interval %v, got %v", newInterval, adjuster.adjustmentInterval)
		}
	})

	t.Run("Rate Increase on High Utilization", func(t *testing.T) {
		currentRate := float64(1024 * 1024) // 1MB/s
		utilization := 0.9                  // 90% utilization

		// Reset last adjustment to ensure adjustment occurs
		adjuster.lastAdjustment = time.Now().Add(-adjuster.adjustmentInterval)

		newRate := adjuster.AdjustRate(currentRate, utilization)
		expectedRate := currentRate * defaultIncreaseRatio

		if newRate != expectedRate {
			t.Errorf("Expected rate %f, got %f", expectedRate, newRate)
		}
	})

	t.Run("Rate Decrease on Low Utilization", func(t *testing.T) {
		currentRate := float64(1024 * 1024) // 1MB/s
		utilization := 0.1                  // 10% utilization

		// Reset last adjustment to ensure adjustment occurs
		adjuster.lastAdjustment = time.Now().Add(-adjuster.adjustmentInterval)

		newRate := adjuster.AdjustRate(currentRate, utilization)
		expectedRate := currentRate * defaultDecreaseRatio

		if newRate != expectedRate {
			t.Errorf("Expected rate %f, got %f", expectedRate, newRate)
		}
	})

	t.Run("No Adjustment Within Interval", func(t *testing.T) {
		currentRate := float64(1024 * 1024) // 1MB/s
		utilization := 0.9                  // 90% utilization

		// Set last adjustment to now
		adjuster.lastAdjustment = time.Now()

		newRate := adjuster.AdjustRate(currentRate, utilization)
		if newRate != currentRate {
			t.Errorf("Expected rate to remain at %f, got %f", currentRate, newRate)
		}
	})

	t.Run("Rate Bounds Enforcement", func(t *testing.T) {
		// Test minimum bound
		adjuster.lastAdjustment = time.Now().Add(-adjuster.adjustmentInterval)
		currentRate := adjuster.minRate * defaultDecreaseRatio // Try to go below minimum
		newRate := adjuster.AdjustRate(currentRate, 0.1)
		if newRate < adjuster.minRate {
			t.Errorf("Rate %f went below minimum %f", newRate, adjuster.minRate)
		}

		// Test maximum bound
		adjuster.lastAdjustment = time.Now().Add(-adjuster.adjustmentInterval)
		currentRate = adjuster.maxRate * defaultIncreaseRatio // Try to go above maximum
		newRate = adjuster.AdjustRate(currentRate, 0.9)
		if newRate > adjuster.maxRate {
			t.Errorf("Rate %f went above maximum %f", newRate, adjuster.maxRate)
		}
	})
}

func TestUtilizationStats(t *testing.T) {
	stats := NewUtilizationStats()
	targetRate := float64(1024 * 1024) // 1MB/s

	t.Run("Initial State", func(t *testing.T) {
		if stats.bytesProcessed != 0 {
			t.Error("Expected initial bytes processed to be 0")
		}
		if stats.GetUtilization(targetRate) != 0 {
			t.Error("Expected initial utilization to be 0")
		}
	})

	t.Run("Add Bytes and Calculate Utilization", func(t *testing.T) {
		// Simulate processing at exactly the target rate for 1 second
		time.Sleep(time.Second)
		stats.AddBytes(uint64(targetRate))

		utilization := stats.GetUtilization(targetRate)
		if utilization < 0.9 || utilization > 1.1 { // Allow 10% margin for timing variations
			t.Errorf("Expected utilization close to 1.0, got %f", utilization)
		}
	})

	t.Run("Stats Reset After Getting Utilization", func(t *testing.T) {
		// Add some bytes
		stats.AddBytes(1000)
		// Get utilization (which should reset stats)
		stats.GetUtilization(targetRate)

		// Verify reset
		if stats.bytesProcessed != 0 {
			t.Error("Expected bytes processed to reset to 0")
		}
		if !stats.startTime.After(time.Now().Add(-time.Second)) {
			t.Error("Expected start time to reset to recent time")
		}
	})

	t.Run("Multiple Updates", func(t *testing.T) {
		// Add bytes in multiple chunks
		stats.AddBytes(500)
		stats.AddBytes(500)
		time.Sleep(time.Second)

		utilization := stats.GetUtilization(targetRate)
		expectedUtilization := float64(1000) / targetRate
		if utilization < expectedUtilization*0.9 || utilization > expectedUtilization*1.1 {
			t.Errorf("Expected utilization around %f, got %f", expectedUtilization, utilization)
		}
	})
}
