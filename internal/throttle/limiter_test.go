package throttle

import (
	"bytes"
	"io"
	"testing"
	"time"
)

func TestTokenBucket(t *testing.T) {
	t.Run("respects rate limit", func(t *testing.T) {
		rateKbps := float64(100) // 100 KB/s
		burstKb := float64(100)  // 100 KB burst
		bucket := NewTokenBucket(rateKbps*1024, burstKb*1024)

		// Try to take more tokens than burst size
		success := bucket.Take(150 * 1024) // 150 KB
		if success {
			t.Error("expected failure when exceeding burst size")
		}

		// Should succeed with burst size
		success = bucket.Take(100 * 1024)
		if !success {
			t.Error("expected success with burst size")
		}
	})

	t.Run("allows burst", func(t *testing.T) {
		rateKbps := float64(100) // 100 KB/s
		burstKb := float64(100)  // 100 KB burst
		bucket := NewTokenBucket(rateKbps*1024, burstKb*1024)

		// Should allow full burst size immediately
		success := bucket.Take(100 * 1024)
		if !success {
			t.Error("expected success for burst size")
		}
	})

	t.Run("refills over time", func(t *testing.T) {
		rateKbps := float64(100) // 100 KB/s
		burstKb := float64(100)  // 100 KB burst
		bucket := NewTokenBucket(rateKbps*1024, burstKb*1024)

		// Take all tokens
		bucket.Take(100 * 1024)

		// Wait for 0.5 seconds
		time.Sleep(500 * time.Millisecond)

		// Should have ~50 KB available
		success := bucket.Take(50 * 1024)
		if !success {
			t.Error("expected success after refill")
		}
	})
}

func TestRateLimitedReader(t *testing.T) {
	t.Run("limits read rate", func(t *testing.T) {
		data := make([]byte, 200*1024) // 200 KB
		reader := bytes.NewReader(data)
		rateKbps := float64(100) // 100 KB/s

		limited := NewRateLimitedReader(reader, rateKbps*1024)

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
		expectedDuration := time.Duration(float64(len(data)) / (rateKbps * 1024) * float64(time.Second))

		// Allow 10% margin for timing variations
		if elapsed < time.Duration(float64(expectedDuration)*0.9) || elapsed > time.Duration(float64(expectedDuration)*1.1) {
			t.Errorf("expected duration around %v, got %v", expectedDuration, elapsed)
		}
	})
}

func TestRateLimitedWriter(t *testing.T) {
	t.Run("limits write rate", func(t *testing.T) {
		var buf bytes.Buffer
		data := make([]byte, 200*1024) // 200 KB
		rateKbps := float64(100)       // 100 KB/s

		limited := NewRateLimitedWriter(&buf, rateKbps*1024)

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
		expectedDuration := time.Duration(float64(len(data)) / (rateKbps * 1024) * float64(time.Second))

		// Allow 10% margin for timing variations
		if elapsed < time.Duration(float64(expectedDuration)*0.9) || elapsed > time.Duration(float64(expectedDuration)*1.1) {
			t.Errorf("expected duration around %v, got %v", expectedDuration, elapsed)
		}
	})
}
