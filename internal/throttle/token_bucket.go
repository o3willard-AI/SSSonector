package throttle

import (
	"fmt"
	"sync"
	"time"
)

const (
	// TCP overhead factor (5%)
	tcpOverheadFactor = 1.05
	// Default timeout for wait operations
	defaultTimeout = 5 * time.Second
)

// TokenBucket implements the token bucket algorithm with TCP overhead compensation
type TokenBucket struct {
	rate          float64 // Base rate without overhead
	effectiveRate float64 // Rate with TCP overhead compensation
	burst         float64 // Maximum burst size
	tokens        float64 // Current token count
	last          time.Time
	mu            sync.Mutex
}

// NewTokenBucket creates a new token bucket with TCP overhead compensation
func NewTokenBucket(rate float64, burst float64) *TokenBucket {
	// Apply TCP overhead compensation to rate
	effectiveRate := rate * tcpOverheadFactor

	// Set burst to 100ms worth of data
	burstFactor := 0.1 // 100ms
	effectiveBurst := effectiveRate * burstFactor

	return &TokenBucket{
		rate:          rate,
		effectiveRate: effectiveRate,
		burst:         effectiveBurst,
		tokens:        effectiveBurst,
		last:          time.Now(),
	}
}

// Allow checks if an operation should be allowed
func (b *TokenBucket) Allow() bool {
	b.mu.Lock()
	defer b.mu.Unlock()

	now := time.Now()
	elapsed := now.Sub(b.last).Seconds()
	b.tokens = min(b.burst, b.tokens+elapsed*b.effectiveRate)
	b.last = now

	if b.tokens < 1 {
		return false
	}

	b.tokens--
	return true
}

// Take attempts to take n tokens from the bucket
func (b *TokenBucket) Take(n float64) bool {
	b.mu.Lock()
	defer b.mu.Unlock()

	now := time.Now()
	elapsed := now.Sub(b.last).Seconds()
	b.tokens = min(b.burst, b.tokens+elapsed*b.effectiveRate)
	b.last = now

	if b.tokens < n {
		return false
	}

	b.tokens -= n
	return true
}

// Wait waits until an operation is allowed or timeout occurs
func (b *TokenBucket) Wait() error {
	return b.WaitWithTimeout(defaultTimeout)
}

// WaitWithTimeout waits until an operation is allowed or timeout occurs
func (b *TokenBucket) WaitWithTimeout(timeout time.Duration) error {
	deadline := time.Now().Add(timeout)
	for !b.Allow() {
		if time.Now().After(deadline) {
			return fmt.Errorf("timeout waiting for rate limit after %v", timeout)
		}
		// Sleep for the time it would take to accumulate one token
		time.Sleep(time.Second / time.Duration(b.effectiveRate))
	}
	return nil
}

// WaitN waits until n tokens are available or timeout occurs
func (b *TokenBucket) WaitN(n float64) error {
	return b.WaitNWithTimeout(n, defaultTimeout)
}

// WaitNWithTimeout waits until n tokens are available or timeout occurs
func (b *TokenBucket) WaitNWithTimeout(n float64, timeout time.Duration) error {
	deadline := time.Now().Add(timeout)
	for !b.Take(n) {
		if time.Now().After(deadline) {
			return fmt.Errorf("timeout waiting for %v tokens after %v", n, timeout)
		}
		// Sleep for the time it would take to accumulate the needed tokens
		sleepDuration := time.Duration(float64(time.Second) * n / b.effectiveRate)
		if sleepDuration > time.Until(deadline) {
			sleepDuration = time.Until(deadline)
		}
		time.Sleep(sleepDuration)
	}
	return nil
}

// Update updates the token bucket configuration
func (b *TokenBucket) Update(rate float64, burst float64) {
	b.mu.Lock()
	defer b.mu.Unlock()

	b.rate = rate
	b.effectiveRate = rate * tcpOverheadFactor

	// Update burst to 100ms worth of data
	burstFactor := 0.1 // 100ms
	b.burst = b.effectiveRate * burstFactor
	b.tokens = min(b.tokens, b.burst)
}

// Available returns the number of available tokens
func (b *TokenBucket) Available() float64 {
	b.mu.Lock()
	defer b.mu.Unlock()

	now := time.Now()
	elapsed := now.Sub(b.last).Seconds()
	b.tokens = min(b.burst, b.tokens+elapsed*b.effectiveRate)
	b.last = now

	return b.tokens
}

// GetRates returns the configured and effective rates
func (b *TokenBucket) GetRates() (float64, float64) {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.rate, b.effectiveRate
}

// GetBurst returns the burst size
func (b *TokenBucket) GetBurst() float64 {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.burst
}
