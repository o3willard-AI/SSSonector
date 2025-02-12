package ratelimit

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"
)

func TestRateLimiter(t *testing.T) {
	logger, _ := zap.NewDevelopment()
	cfg := &Config{
		DefaultRate:  10,
		DefaultBurst: 5,
		CleanupTime:  time.Hour,
	}

	t.Run("basic rate limiting", func(t *testing.T) {
		rl := NewRateLimiter(cfg, logger)
		defer rl.Stop()

		// Should allow burst
		for i := 0; i < 5; i++ {
			assert.True(t, rl.Allow("test"))
		}

		// Should deny after burst
		assert.False(t, rl.Allow("test"))

		// Wait for token replenishment
		time.Sleep(time.Second)

		// Should allow after replenishment
		assert.True(t, rl.Allow("test"))
	})

	t.Run("custom rate", func(t *testing.T) {
		rl := NewRateLimiter(cfg, logger)
		defer rl.Stop()

		err := rl.SetRate("test", 2, 2)
		assert.NoError(t, err)

		// Should allow burst
		assert.True(t, rl.Allow("test"))
		assert.True(t, rl.Allow("test"))

		// Should deny after burst
		assert.False(t, rl.Allow("test"))

		// Wait for token replenishment
		time.Sleep(time.Second)

		// Should allow after replenishment
		assert.True(t, rl.Allow("test"))
	})

	t.Run("invalid rate", func(t *testing.T) {
		rl := NewRateLimiter(cfg, logger)
		defer rl.Stop()

		err := rl.SetRate("test", 0, 1)
		assert.Error(t, err)

		err = rl.SetRate("test", 1, 0)
		assert.Error(t, err)

		err = rl.SetRate("test", -1, 1)
		assert.Error(t, err)
	})

	t.Run("get rate", func(t *testing.T) {
		rl := NewRateLimiter(cfg, logger)
		defer rl.Stop()

		// Should return default rate for new ID
		rate, burst := rl.GetRate("test")
		assert.Equal(t, int64(10), rate)
		assert.Equal(t, int64(5), burst)

		// Set custom rate
		err := rl.SetRate("test", 20, 10)
		assert.NoError(t, err)

		// Should return custom rate
		rate, burst = rl.GetRate("test")
		assert.Equal(t, int64(20), rate)
		assert.Equal(t, int64(10), burst)

		// Remove rate limit
		rl.Remove("test")

		// Should return default rate after removal
		rate, burst = rl.GetRate("test")
		assert.Equal(t, int64(10), rate)
		assert.Equal(t, int64(5), burst)
	})

	t.Run("wait for tokens", func(t *testing.T) {
		rl := NewRateLimiter(cfg, logger)
		defer rl.Stop()

		// Set low rate for testing
		err := rl.SetRate("test", 2, 1)
		assert.NoError(t, err)

		// Use up the burst
		assert.True(t, rl.Allow("test"))

		// Start waiting for token
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()

		// Should get token within timeout
		err = rl.WaitN(ctx, "test", 1)
		assert.NoError(t, err)

		// Try with cancelled context
		cancelCtx, cancel := context.WithCancel(context.Background())
		cancel()
		err = rl.WaitN(cancelCtx, "test", 1)
		assert.Error(t, err)
	})

	t.Run("multiple tokens", func(t *testing.T) {
		rl := NewRateLimiter(cfg, logger)
		defer rl.Stop()

		// Set rate with burst
		err := rl.SetRate("test", 10, 5)
		assert.NoError(t, err)

		// Should allow taking multiple tokens
		assert.True(t, rl.AllowN("test", 3))

		// Should deny when requesting more than available
		assert.False(t, rl.AllowN("test", 3))

		// Wait for replenishment
		time.Sleep(time.Second)

		// Should allow after replenishment
		assert.True(t, rl.AllowN("test", 2))
	})

	t.Run("metrics", func(t *testing.T) {
		rl := NewRateLimiter(cfg, logger)
		defer rl.Stop()

		// Add some rate limits
		rl.Allow("test1")
		rl.Allow("test2")
		rl.Allow("test3")

		metrics := rl.GetMetrics()
		assert.Equal(t, 3, metrics.ActiveBuckets)
		assert.Greater(t, metrics.TotalTokens, float64(0))
		assert.Greater(t, metrics.AvgTokens, float64(0))

		// Remove a rate limit
		rl.Remove("test1")

		metrics = rl.GetMetrics()
		assert.Equal(t, 2, metrics.ActiveBuckets)
	})

	t.Run("cleanup", func(t *testing.T) {
		// Create limiter with short cleanup time
		rl := NewRateLimiter(&Config{
			DefaultRate:  10,
			DefaultBurst: 5,
			CleanupTime:  100 * time.Millisecond,
		}, logger)
		defer rl.Stop()

		// Add some rate limits
		rl.Allow("test1")
		rl.Allow("test2")

		// Wait for cleanup
		time.Sleep(200 * time.Millisecond)

		// Buckets should still exist as they're not old enough
		metrics := rl.GetMetrics()
		assert.Equal(t, 2, metrics.ActiveBuckets)

		// Force bucket lastUpdate to be old
		rl.mu.Lock()
		for _, bucket := range rl.buckets {
			bucket.lastUpdate = time.Now().Add(-25 * time.Hour)
		}
		rl.mu.Unlock()

		// Wait for cleanup
		time.Sleep(200 * time.Millisecond)

		// Buckets should be cleaned up
		metrics = rl.GetMetrics()
		assert.Equal(t, 0, metrics.ActiveBuckets)
	})
}
