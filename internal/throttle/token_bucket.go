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
		tokens:     burst,
		lastUpdate: time.Now(),
	}
}

// Update updates the rate and burst size
func (b *TokenBucket) Update(rate, burst float64) {
	b.mu.Lock()
	defer b.mu.Unlock()

	b.rate = rate
	b.burst = burst
	if b.tokens > burst {
		b.tokens = burst
	}
}

// Wait waits until enough tokens are available
func (b *TokenBucket) Wait(size float64) {
	b.mu.Lock()
	defer b.mu.Unlock()

	now := time.Now()
	elapsed := now.Sub(b.lastUpdate).Seconds()
	b.tokens = min(b.burst, b.tokens+elapsed*b.rate)
	b.lastUpdate = now

	if b.tokens < size {
		sleepDuration := time.Duration((size - b.tokens) / b.rate * float64(time.Second))
		time.Sleep(sleepDuration)
		b.tokens = 0
		b.lastUpdate = time.Now()
	} else {
		b.tokens -= size
	}
}
