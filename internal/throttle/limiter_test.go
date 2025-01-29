package throttle

import (
	"bytes"
	"context"
	"io"
	"testing"
	"time"
)

func TestTokenBucket(t *testing.T) {
	t.Run("respects rate limit", func(t *testing.T) {
		rateKbps := float64(100) // 100 KB/s
		burstKb := float64(100)  // 100 KB burst
		bucket := NewTokenBucket(rateKbps, burstKb)

		// Try to take more tokens than burst size
		wait := bucket.Take(150 * 1024) // 150 KB
		if wait == 0 {
			t.Error("expected wait time for exceeding burst size")
		}

		// Expected wait time for 50 KB at 100 KB/s = 0.5s
		expectedWait := time.Duration(0.5 * float64(time.Second))
		if wait < time.Duration(float64(expectedWait)*0.9) || wait > time.Duration(float64(expectedWait)*1.1) {
			t.Errorf("expected wait time around %v, got %v", expectedWait, wait)
		}
	})

	t.Run("allows burst", func(t *testing.T) {
		rateKbps := float64(100) // 100 KB/s
		burstKb := float64(100)  // 100 KB burst
		bucket := NewTokenBucket(rateKbps, burstKb)

		// Should allow full burst size immediately
		wait := bucket.Take(100 * 1024)
		if wait != 0 {
			t.Errorf("expected no wait for burst size, got %v", wait)
		}
	})

	t.Run("refills over time", func(t *testing.T) {
		rateKbps := float64(100) // 100 KB/s
		burstKb := float64(100)  // 100 KB burst
		bucket := NewTokenBucket(rateKbps, burstKb)

		// Take all tokens
		bucket.Take(100 * 1024)

		// Wait for 0.5 seconds
		time.Sleep(500 * time.Millisecond)

		// Should have ~50 KB available
		wait := bucket.Take(50 * 1024)
		if wait != 0 {
			t.Errorf("expected no wait after refill, got %v", wait)
		}
	})
}

func TestRateLimitedReader(t *testing.T) {
	t.Run("limits read rate", func(t *testing.T) {
		data := make([]byte, 200*1024) // 200 KB
		reader := bytes.NewReader(data)
		rateKbps := float64(100) // 100 KB/s

		ctx := context.Background()
		limited := NewRateLimitedReader(ctx, reader, rateKbps)

		start := time.Now()

		// Try to read all data
		buf := make([]byte, len(data))
		n, err := io.ReadFull(limited, buf)
		if err != nil {
			t.Fatalf("read error: %v", err)
		}
		if n != len(data) {
			t.Errorf("expected to read %d bytes, got %d", len(data), n)
		}

		elapsed := time.Since(start)
		expectedDuration := time.Duration(float64(len(data)) / 1024 / rateKbps * float64(time.Second))

		// Allow 10% margin for timing variations
		if elapsed < time.Duration(float64(expectedDuration)*0.9) || elapsed > time.Duration(float64(expectedDuration)*1.1) {
			t.Errorf("expected duration around %v, got %v", expectedDuration, elapsed)
		}
	})

	t.Run("respects context cancellation", func(t *testing.T) {
		data := make([]byte, 100*1024) // 100 KB
		reader := bytes.NewReader(data)
		rateKbps := float64(10) // 10 KB/s

		ctx, cancel := context.WithCancel(context.Background())
		limited := NewRateLimitedReader(ctx, reader, rateKbps)

		// Cancel context after a short delay
		go func() {
			time.Sleep(100 * time.Millisecond)
			cancel()
		}()

		// Try to read all data
		buf := make([]byte, len(data))
		_, err := io.ReadFull(limited, buf)
		if err != context.Canceled {
			t.Errorf("expected context.Canceled error, got %v", err)
		}
	})
}

func TestRateLimitedWriter(t *testing.T) {
	t.Run("limits write rate", func(t *testing.T) {
		var buf bytes.Buffer
		data := make([]byte, 200*1024) // 200 KB
		rateKbps := float64(100)       // 100 KB/s

		ctx := context.Background()
		limited := NewRateLimitedWriter(ctx, &buf, rateKbps)

		start := time.Now()

		// Try to write all data
		n, err := limited.Write(data)
		if err != nil {
			t.Fatalf("write error: %v", err)
		}
		if n != len(data) {
			t.Errorf("expected to write %d bytes, got %d", len(data), n)
		}

		elapsed := time.Since(start)
		expectedDuration := time.Duration(float64(len(data)) / 1024 / rateKbps * float64(time.Second))

		// Allow 10% margin for timing variations
		if elapsed < time.Duration(float64(expectedDuration)*0.9) || elapsed > time.Duration(float64(expectedDuration)*1.1) {
			t.Errorf("expected duration around %v, got %v", expectedDuration, elapsed)
		}
	})

	t.Run("respects context cancellation", func(t *testing.T) {
		var buf bytes.Buffer
		data := make([]byte, 100*1024) // 100 KB
		rateKbps := float64(10)        // 10 KB/s

		ctx, cancel := context.WithCancel(context.Background())
		limited := NewRateLimitedWriter(ctx, &buf, rateKbps)

		// Cancel context after a short delay
		go func() {
			time.Sleep(100 * time.Millisecond)
			cancel()
		}()

		// Try to write all data
		_, err := limited.Write(data)
		if err != context.Canceled {
			t.Errorf("expected context.Canceled error, got %v", err)
		}
	})
}
