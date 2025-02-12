package ratelimit

import (
	"context"
	"fmt"
	"sync"
	"time"

	"go.uber.org/zap"
)

// RateLimiter manages rate limiting for connections
type RateLimiter struct {
	mu            sync.RWMutex
	buckets       map[string]*TokenBucket
	defaultRate   int64
	defaultBurst  int64
	cleanupTicker *time.Ticker
	logger        *zap.Logger
}

// TokenBucket implements the token bucket algorithm
type TokenBucket struct {
	rate       int64
	burst      int64
	tokens     float64
	lastUpdate time.Time
	mu         sync.Mutex
}

// Config holds rate limiter configuration
type Config struct {
	DefaultRate  int64         // Tokens per second
	DefaultBurst int64         // Maximum burst size
	CleanupTime  time.Duration // How often to clean up inactive buckets
}

// NewRateLimiter creates a new rate limiter
func NewRateLimiter(cfg *Config, logger *zap.Logger) *RateLimiter {
	rl := &RateLimiter{
		buckets:      make(map[string]*TokenBucket),
		defaultRate:  cfg.DefaultRate,
		defaultBurst: cfg.DefaultBurst,
		logger:       logger,
	}

	// Start cleanup routine
	rl.cleanupTicker = time.NewTicker(cfg.CleanupTime)
	go rl.cleanup()

	return rl
}

// newTokenBucket creates a new token bucket
func newTokenBucket(rate, burst int64) *TokenBucket {
	return &TokenBucket{
		rate:       rate,
		burst:      burst,
		tokens:     float64(burst),
		lastUpdate: time.Now(),
	}
}

// Allow checks if a request should be allowed
func (rl *RateLimiter) Allow(id string) bool {
	return rl.AllowN(id, 1)
}

// AllowN checks if n requests should be allowed
func (rl *RateLimiter) AllowN(id string, n int64) bool {
	bucket := rl.getBucket(id)
	return bucket.takeN(n)
}

// SetRate sets the rate for a specific ID
func (rl *RateLimiter) SetRate(id string, rate, burst int64) error {
	if rate <= 0 || burst <= 0 {
		return fmt.Errorf("rate and burst must be positive")
	}

	rl.mu.Lock()
	defer rl.mu.Unlock()

	bucket := newTokenBucket(rate, burst)
	rl.buckets[id] = bucket

	rl.logger.Debug("rate limit set",
		zap.String("id", id),
		zap.Int64("rate", rate),
		zap.Int64("burst", burst))

	return nil
}

// GetRate gets the current rate for an ID
func (rl *RateLimiter) GetRate(id string) (rate, burst int64) {
	rl.mu.RLock()
	defer rl.mu.RUnlock()

	if bucket, exists := rl.buckets[id]; exists {
		return bucket.rate, bucket.burst
	}
	return rl.defaultRate, rl.defaultBurst
}

// Remove removes rate limiting for an ID
func (rl *RateLimiter) Remove(id string) {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	delete(rl.buckets, id)
	rl.logger.Debug("rate limit removed", zap.String("id", id))
}

// getBucket gets or creates a token bucket for an ID
func (rl *RateLimiter) getBucket(id string) *TokenBucket {
	rl.mu.RLock()
	bucket, exists := rl.buckets[id]
	rl.mu.RUnlock()

	if exists {
		return bucket
	}

	rl.mu.Lock()
	defer rl.mu.Unlock()

	// Check again in case another goroutine created it
	if bucket, exists = rl.buckets[id]; exists {
		return bucket
	}

	bucket = newTokenBucket(rl.defaultRate, rl.defaultBurst)
	rl.buckets[id] = bucket
	return bucket
}

// takeN attempts to take n tokens from the bucket
func (tb *TokenBucket) takeN(n int64) bool {
	tb.mu.Lock()
	defer tb.mu.Unlock()

	now := time.Now()
	elapsed := now.Sub(tb.lastUpdate).Seconds()
	tb.tokens = min(float64(tb.burst), tb.tokens+elapsed*float64(tb.rate))
	tb.lastUpdate = now

	if tb.tokens >= float64(n) {
		tb.tokens -= float64(n)
		return true
	}
	return false
}

// cleanup periodically removes inactive buckets
func (rl *RateLimiter) cleanup() {
	for range rl.cleanupTicker.C {
		rl.mu.Lock()
		now := time.Now()
		for id, bucket := range rl.buckets {
			bucket.mu.Lock()
			if now.Sub(bucket.lastUpdate) > 24*time.Hour {
				delete(rl.buckets, id)
				rl.logger.Debug("removed inactive rate limit bucket",
					zap.String("id", id))
			}
			bucket.mu.Unlock()
		}
		rl.mu.Unlock()
	}
}

// Stop stops the rate limiter
func (rl *RateLimiter) Stop() {
	rl.cleanupTicker.Stop()
}

// min returns the minimum of two float64 values
func min(a, b float64) float64 {
	if a < b {
		return a
	}
	return b
}

// WaitN waits until n tokens are available
func (rl *RateLimiter) WaitN(ctx context.Context, id string, n int64) error {
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			if rl.AllowN(id, n) {
				return nil
			}
			time.Sleep(time.Second / time.Duration(rl.defaultRate))
		}
	}
}

// Metrics returns rate limiting metrics
type Metrics struct {
	ActiveBuckets int
	TotalTokens   float64
	AvgTokens     float64
}

// GetMetrics returns current rate limiting metrics
func (rl *RateLimiter) GetMetrics() Metrics {
	rl.mu.RLock()
	defer rl.mu.RUnlock()

	metrics := Metrics{
		ActiveBuckets: len(rl.buckets),
	}

	if len(rl.buckets) > 0 {
		for _, bucket := range rl.buckets {
			bucket.mu.Lock()
			metrics.TotalTokens += bucket.tokens
			bucket.mu.Unlock()
		}
		metrics.AvgTokens = metrics.TotalTokens / float64(len(rl.buckets))
	}

	return metrics
}
