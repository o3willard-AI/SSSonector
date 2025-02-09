package throttle

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestTokenBucket(t *testing.T) {
	t.Run("Basic", func(t *testing.T) {
		rateKb := float64(100)  // 100 KB/s
		burstKb := float64(200) // 200 KB burst
		bucket := NewTokenBucket(rateKb*1024, burstKb*1024)

		// Should allow burst
		assert.True(t, bucket.Take(burstKb*1024))

		// Should not allow more than burst
		assert.False(t, bucket.Take(1))

		// Wait for tokens to refill
		time.Sleep(time.Second)

		// Should allow rate
		assert.True(t, bucket.Take(rateKb*1024))
	})

	t.Run("Rate", func(t *testing.T) {
		rateKb := float64(100)  // 100 KB/s
		burstKb := float64(200) // 200 KB burst
		bucket := NewTokenBucket(rateKb*1024, burstKb*1024)

		// Take initial burst
		assert.True(t, bucket.Take(burstKb*1024))

		// Wait for tokens to refill
		time.Sleep(time.Second)

		// Should allow rate per second
		for i := 0; i < 10; i++ {
			assert.True(t, bucket.Take(rateKb*1024/10))
			time.Sleep(100 * time.Millisecond)
		}
	})

	t.Run("Update", func(t *testing.T) {
		rateKb := float64(100)  // 100 KB/s
		burstKb := float64(200) // 200 KB burst
		bucket := NewTokenBucket(rateKb*1024, burstKb*1024)

		// Take initial burst
		assert.True(t, bucket.Take(burstKb*1024))

		// Update to higher rate/burst
		newRateKb := float64(200)  // 200 KB/s
		newBurstKb := float64(400) // 400 KB burst
		bucket.Update(newRateKb*1024, newBurstKb*1024)

		// Wait for tokens to refill
		time.Sleep(time.Second)

		// Should allow new rate
		assert.True(t, bucket.Take(newRateKb*1024))
	})
}

func TestRateLimitedIO(t *testing.T) {
	t.Run("Reader", func(t *testing.T) {
		data := make([]byte, 1024*1024)                                    // 1 MB
		reader := NewRateLimitedReader(&testReader{data: data}, 1024*1024) // 1 MB/s

		buf := make([]byte, 1024*100) // 100 KB chunks
		start := time.Now()

		// Read 1 MB in 100 KB chunks
		for i := 0; i < 10; i++ {
			n, err := reader.Read(buf)
			assert.NoError(t, err)
			assert.Equal(t, len(buf), n)
		}

		// Should take ~1 second
		elapsed := time.Since(start)
		assert.InDelta(t, 1.0, elapsed.Seconds(), 0.1)
	})

	t.Run("Writer", func(t *testing.T) {
		data := make([]byte, 1024*1024)                          // 1 MB
		writer := NewRateLimitedWriter(&testWriter{}, 1024*1024) // 1 MB/s

		start := time.Now()

		// Write 1 MB
		n, err := writer.Write(data)
		assert.NoError(t, err)
		assert.Equal(t, len(data), n)

		// Should take ~1 second
		elapsed := time.Since(start)
		assert.InDelta(t, 1.0, elapsed.Seconds(), 0.1)
	})
}

// Test helpers

type testReader struct {
	data []byte
	pos  int
}

func (r *testReader) Read(p []byte) (n int, err error) {
	n = copy(p, r.data[r.pos:])
	r.pos += n
	return n, nil
}

type testWriter struct{}

func (w *testWriter) Write(p []byte) (n int, err error) {
	return len(p), nil
}
