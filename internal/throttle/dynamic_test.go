package throttle

import (
	"bytes"
	"testing"
	"time"

	"github.com/o3willard-AI/SSSonector/internal/config/types"
	"go.uber.org/zap"
)

func TestDynamicLimiter(t *testing.T) {
	logger, _ := zap.NewDevelopment()

	// Create a base config
	baseConfig := &types.AppConfig{
		Throttle: &types.ThrottleConfig{
			Enabled: true,
			Rate:    1024 * 1024, // 1MB/s
			Burst:   1024 * 100,  // 100KB burst
		},
	}

	// Create base limiter
	reader := bytes.NewBuffer(nil)
	writer := bytes.NewBuffer(nil)
	baseLimiter := NewLimiter(baseConfig, reader, writer, logger)

	t.Run("Initial Configuration", func(t *testing.T) {
		dl := NewDynamicLimiter(baseConfig, baseLimiter, logger)

		expectedMinRate := float64(baseConfig.Throttle.Rate) / 2
		expectedMaxRate := float64(baseConfig.Throttle.Rate) * 2

		if dl.minRate != expectedMinRate {
			t.Errorf("Expected min rate %f, got %f", expectedMinRate, dl.minRate)
		}
		if dl.maxRate != expectedMaxRate {
			t.Errorf("Expected max rate %f, got %f", expectedMaxRate, dl.maxRate)
		}
	})

	t.Run("Rate Increase", func(t *testing.T) {
		dl := NewDynamicLimiter(baseConfig, baseLimiter, logger)
		initialRate := float64(baseConfig.Throttle.Rate)

		// Increase by 50%
		if !dl.IncreaseRate(50) {
			t.Error("Rate increase should have succeeded")
		}

		inMetrics, _ := dl.GetMetrics()
		newRate := inMetrics.Rate

		expectedRate := initialRate * 1.5
		if expectedRate > dl.maxRate {
			expectedRate = dl.maxRate
		}

		if abs(newRate-expectedRate) > 0.1 {
			t.Errorf("Expected rate around %f, got %f", expectedRate, newRate)
		}
	})

	t.Run("Rate Decrease", func(t *testing.T) {
		dl := NewDynamicLimiter(baseConfig, baseLimiter, logger)
		initialRate := float64(baseConfig.Throttle.Rate)

		// Decrease by 50%
		if !dl.DecreaseRate(50) {
			t.Error("Rate decrease should have succeeded")
		}

		inMetrics, _ := dl.GetMetrics()
		newRate := inMetrics.Rate

		expectedRate := initialRate * 0.5
		if expectedRate < dl.minRate {
			expectedRate = dl.minRate
		}

		if abs(newRate-expectedRate) > 0.1 {
			t.Errorf("Expected rate around %f, got %f", expectedRate, newRate)
		}
	})

	t.Run("Rate Bounds", func(t *testing.T) {
		dl := NewDynamicLimiter(baseConfig, baseLimiter, logger)

		// Try to increase beyond max rate
		dl.IncreaseRate(1000) // Large increase

		inMetrics, _ := dl.GetMetrics()
		if inMetrics.Rate > dl.maxRate {
			t.Errorf("Rate %f exceeded maximum %f", inMetrics.Rate, dl.maxRate)
		}

		// Reset and try to decrease below min rate
		dl = NewDynamicLimiter(baseConfig, baseLimiter, logger)
		dl.DecreaseRate(90) // Large decrease

		inMetrics, _ = dl.GetMetrics()
		if inMetrics.Rate < dl.minRate {
			t.Errorf("Rate %f went below minimum %f", inMetrics.Rate, dl.minRate)
		}
	})

	t.Run("Adjustment Count", func(t *testing.T) {
		dl := NewDynamicLimiter(baseConfig, baseLimiter, logger)

		if count := dl.GetAdjustCount(); count != 0 {
			t.Errorf("Expected initial adjust count 0, got %d", count)
		}

		// Set a very short cooldown for testing
		dl.SetCooldown(types.NewDuration(1 * time.Millisecond))

		if !dl.IncreaseRate(10) {
			t.Error("First increase should have succeeded")
		}
		if count := dl.GetAdjustCount(); count != 1 {
			t.Errorf("Expected adjust count 1, got %d", count)
		}

		// Wait for cooldown
		time.Sleep(2 * time.Millisecond)

		if !dl.DecreaseRate(10) {
			t.Error("First decrease should have succeeded")
		}
		if count := dl.GetAdjustCount(); count != 2 {
			t.Errorf("Expected adjust count 2, got %d", count)
		}

		dl.ResetAdjustCount()
		if count := dl.GetAdjustCount(); count != 0 {
			t.Errorf("Expected reset adjust count 0, got %d", count)
		}
	})

	t.Run("Rate Adjustment Cooldown", func(t *testing.T) {
		dl := NewDynamicLimiter(baseConfig, baseLimiter, logger)
		initialRate := float64(baseConfig.Throttle.Rate)

		// Set cooldown before first adjustment
		cooldown := types.NewDuration(200 * time.Millisecond)
		dl.SetCooldown(cooldown)

		// First adjustment should work
		if !dl.IncreaseRate(50) {
			t.Error("First adjustment should have succeeded")
		}
		inMetrics, _ := dl.GetMetrics()
		firstAdjustRate := inMetrics.Rate

		if firstAdjustRate == initialRate {
			t.Error("First adjustment had no effect")
		}

		// Wait a bit but not enough for cooldown
		time.Sleep(100 * time.Millisecond)

		// Immediate second adjustment should be ignored
		if dl.IncreaseRate(50) {
			t.Error("Second adjustment should have been blocked by cooldown")
		}
		inMetrics, _ = dl.GetMetrics()
		secondAdjustRate := inMetrics.Rate

		if secondAdjustRate != firstAdjustRate {
			t.Error("Second adjustment should have been ignored due to cooldown")
		}

		// Wait for cooldown
		time.Sleep(200 * time.Millisecond)

		// Third adjustment should work
		if !dl.IncreaseRate(50) {
			t.Error("Third adjustment should have succeeded after cooldown")
		}
		inMetrics, _ = dl.GetMetrics()
		thirdAdjustRate := inMetrics.Rate

		if thirdAdjustRate <= secondAdjustRate {
			t.Error("Third adjustment should have increased the rate after cooldown")
		}
	})
}

// Helper function to calculate absolute difference
func abs(x float64) float64 {
	if x < 0 {
		return -x
	}
	return x
}
