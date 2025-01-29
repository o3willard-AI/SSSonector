package throttle

import (
	"context"
	"io"
	"sync"
	"time"
)

// TokenBucket implements a token bucket rate limiter
type TokenBucket struct {
	rate       float64    // tokens per second
	burstSize  float64    // maximum bucket size
	tokens     float64    // current number of tokens
	lastUpdate time.Time  // last time tokens were added
	mu         sync.Mutex // protects tokens and lastUpdate
}

// NewTokenBucket creates a new token bucket rate limiter
func NewTokenBucket(rateKbps float64, burstKb float64) *TokenBucket {
	return &TokenBucket{
		rate:       rateKbps * 1024, // convert to bytes per second
		burstSize:  burstKb * 1024,  // convert to bytes
		tokens:     burstKb * 1024,  // start with full bucket
		lastUpdate: time.Now(),
	}
}

// updateTokens adds tokens based on elapsed time
func (tb *TokenBucket) updateTokens() {
	now := time.Now()
	elapsed := now.Sub(tb.lastUpdate).Seconds()
	tb.tokens = min(tb.burstSize, tb.tokens+elapsed*tb.rate)
	tb.lastUpdate = now
}

// Take attempts to take n tokens from the bucket
func (tb *TokenBucket) Take(n float64) time.Duration {
	tb.mu.Lock()
	defer tb.mu.Unlock()

	tb.updateTokens()

	// If we have enough tokens, take them immediately
	if tb.tokens >= n {
		tb.tokens -= n
		return 0
	}

	// Calculate how long to wait for enough tokens
	needed := n - tb.tokens
	wait := time.Duration(needed / tb.rate * float64(time.Second))
	tb.tokens = 0
	return wait
}

// RateLimitedReader wraps an io.Reader with rate limiting
type RateLimitedReader struct {
	reader io.Reader
	bucket *TokenBucket
	ctx    context.Context
}

// NewRateLimitedReader creates a new rate-limited reader
func NewRateLimitedReader(ctx context.Context, reader io.Reader, rateKbps float64) *RateLimitedReader {
	// Use burst size of 1 second worth of data
	return &RateLimitedReader{
		reader: reader,
		bucket: NewTokenBucket(rateKbps, rateKbps),
		ctx:    ctx,
	}
}

// Read implements io.Reader with rate limiting
func (r *RateLimitedReader) Read(p []byte) (int, error) {
	// Check if context is cancelled
	select {
	case <-r.ctx.Done():
		return 0, r.ctx.Err()
	default:
	}

	// Calculate wait time for the requested number of bytes
	wait := r.bucket.Take(float64(len(p)))
	if wait > 0 {
		timer := time.NewTimer(wait)
		select {
		case <-r.ctx.Done():
			timer.Stop()
			return 0, r.ctx.Err()
		case <-timer.C:
		}
	}

	return r.reader.Read(p)
}

// RateLimitedWriter wraps an io.Writer with rate limiting
type RateLimitedWriter struct {
	writer io.Writer
	bucket *TokenBucket
	ctx    context.Context
}

// NewRateLimitedWriter creates a new rate-limited writer
func NewRateLimitedWriter(ctx context.Context, writer io.Writer, rateKbps float64) *RateLimitedWriter {
	// Use burst size of 1 second worth of data
	return &RateLimitedWriter{
		writer: writer,
		bucket: NewTokenBucket(rateKbps, rateKbps),
		ctx:    ctx,
	}
}

// Write implements io.Writer with rate limiting
func (w *RateLimitedWriter) Write(p []byte) (int, error) {
	// Check if context is cancelled
	select {
	case <-w.ctx.Done():
		return 0, w.ctx.Err()
	default:
	}

	// Calculate wait time for the requested number of bytes
	wait := w.bucket.Take(float64(len(p)))
	if wait > 0 {
		timer := time.NewTimer(wait)
		select {
		case <-w.ctx.Done():
			timer.Stop()
			return 0, w.ctx.Err()
		case <-timer.C:
		}
	}

	return w.writer.Write(p)
}

// min returns the minimum of two float64 values
func min(a, b float64) float64 {
	if a < b {
		return a
	}
	return b
}
