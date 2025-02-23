package throttle

import (
	"sync"
	"time"
)

// TokenBucket implements the token bucket rate limiting algorithm
type TokenBucket struct {
	rate       float64
	burst      float64
	tokens     float64
	lastUpdate time.Time
	mu         sync.Mutex
}

// NewTokenBucket creates a new token bucket rate limiter
func NewTokenBucket(rate, burst float64) *TokenBucket {
	return &TokenBucket{
		rate:       rate,
		burst:      burst,
		tokens:     0, // Start with 0 tokens to enforce initial rate limiting
		lastUpdate: time.Now(),
	}
}

// Update updates the rate and burst size
func (b *TokenBucket) Update(rate, burst float64) {
	b.mu.Lock()
	defer b.mu.Unlock()

	// Add any tokens that should have accumulated since last update
	now := time.Now()
	elapsed := now.Sub(b.lastUpdate).Seconds()
	b.tokens = min(b.burst, b.tokens+elapsed*b.rate)

	b.rate = rate
	b.burst = burst
	if b.tokens > burst {
		b.tokens = burst
	}
	b.lastUpdate = now
}

// Wait waits until enough tokens are available
func (b *TokenBucket) Wait(size float64) {
	b.mu.Lock()

	// Add any tokens that should have accumulated
	now := time.Now()
	elapsed := now.Sub(b.lastUpdate).Seconds()
	b.tokens = min(b.burst, b.tokens+elapsed*b.rate)

	// Calculate how long to wait for enough tokens
	needed := size - b.tokens
	if needed > 0 {
		// Calculate wait time based on needed tokens and rate
		// Add a small buffer (10%) to ensure we meet minimum timing requirements
		waitTime := time.Duration((needed / b.rate) * 1.1 * float64(time.Second))
		b.tokens = 0
		b.lastUpdate = now.Add(waitTime)
		b.mu.Unlock()

		// Sleep for the calculated duration
		time.Sleep(waitTime)

		// Re-acquire lock to update tokens after waiting
		b.mu.Lock()
		b.tokens = 0 // Start fresh after waiting
		b.lastUpdate = time.Now()
		b.mu.Unlock()
		return
	}

	// If we have enough tokens, consume them immediately
	b.tokens -= size
	b.lastUpdate = now
	b.mu.Unlock()
}

// GetTokens returns the current number of tokens
func (b *TokenBucket) GetTokens() float64 {
	b.mu.Lock()
	defer b.mu.Unlock()

	now := time.Now()
	elapsed := now.Sub(b.lastUpdate).Seconds()
	b.tokens = min(b.burst, b.tokens+elapsed*b.rate)
	b.lastUpdate = now

	return b.tokens
}
