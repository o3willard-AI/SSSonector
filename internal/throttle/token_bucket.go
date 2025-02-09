package throttle

import (
	"sync"
	"time"
)

// TokenBucket implements the token bucket algorithm
type TokenBucket struct {
	rate   float64
	burst  float64
	tokens float64
	last   time.Time
	mu     sync.Mutex
}

// NewTokenBucket creates a new token bucket
func NewTokenBucket(rate float64, burst float64) *TokenBucket {
	return &TokenBucket{
		rate:   rate,
		burst:  burst,
		tokens: burst,
		last:   time.Now(),
	}
}

// Allow checks if an operation should be allowed
func (b *TokenBucket) Allow() bool {
	b.mu.Lock()
	defer b.mu.Unlock()

	now := time.Now()
	elapsed := now.Sub(b.last).Seconds()
	b.tokens = min(b.burst, b.tokens+elapsed*b.rate)
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
	b.tokens = min(b.burst, b.tokens+elapsed*b.rate)
	b.last = now

	if b.tokens < n {
		return false
	}

	b.tokens -= n
	return true
}

// Wait waits until an operation is allowed
func (b *TokenBucket) Wait() {
	for !b.Allow() {
		time.Sleep(time.Second / time.Duration(b.rate))
	}
}

// WaitN waits until n tokens are available
func (b *TokenBucket) WaitN(n float64) {
	for !b.Take(n) {
		time.Sleep(time.Second / time.Duration(b.rate))
	}
}

// Update updates the token bucket configuration
func (b *TokenBucket) Update(rate float64, burst float64) {
	b.mu.Lock()
	defer b.mu.Unlock()

	b.rate = rate
	b.burst = burst
	b.tokens = min(b.tokens, burst)
}

// Available returns the number of available tokens
func (b *TokenBucket) Available() float64 {
	b.mu.Lock()
	defer b.mu.Unlock()

	now := time.Now()
	elapsed := now.Sub(b.last).Seconds()
	b.tokens = min(b.burst, b.tokens+elapsed*b.rate)
	b.last = now

	return b.tokens
}
