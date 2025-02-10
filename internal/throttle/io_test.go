package throttle

import (
	"bytes"
	"io"
	"testing"
	"time"

	"github.com/o3willard-AI/SSSonector/internal/config"
	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"
)

func setupTestLogger() *zap.Logger {
	logger, _ := zap.NewDevelopment()
	return logger
}

func TestThrottledReader(t *testing.T) {
	logger := setupTestLogger()

	// Test with small buffer (no pooling)
	t.Run("small buffer", func(t *testing.T) {
		data := []byte("test data")
		reader := bytes.NewReader(data)
		limiter := NewLimiter(&config.AppConfig{
			Throttle: config.ThrottleConfig{
				Enabled: true,
				Rate:    1024 * 1024, // 1MB/s
				Burst:   64 * 1024,   // 64KB
			},
		}, reader, nil, logger)

		tr := NewThrottledReader(reader, limiter, logger)
		buf := make([]byte, 4) // Below minBufferSize
		n, err := tr.Read(buf)
		assert.NoError(t, err)
		assert.Equal(t, 4, n)
		assert.Equal(t, data[:4], buf)
	})

	// Test with large buffer (uses pool)
	t.Run("large buffer", func(t *testing.T) {
		data := make([]byte, 8*1024) // 8KB
		for i := range data {
			data[i] = byte(i % 256)
		}
		reader := bytes.NewReader(data)
		limiter := NewLimiter(&config.AppConfig{
			Throttle: config.ThrottleConfig{
				Enabled: true,
				Rate:    1024 * 1024, // 1MB/s
				Burst:   64 * 1024,   // 64KB
			},
		}, reader, nil, logger)

		tr := NewThrottledReader(reader, limiter, logger)
		buf := make([]byte, len(data))
		n, err := tr.Read(buf)
		assert.NoError(t, err)
		assert.Equal(t, len(data), n)
		assert.Equal(t, data, buf)
	})

	// Test rate limiting
	t.Run("rate limiting", func(t *testing.T) {
		data := make([]byte, 4*1024) // 4KB
		reader := bytes.NewReader(data)
		limiter := NewLimiter(&config.AppConfig{
			Throttle: config.ThrottleConfig{
				Enabled: true,
				Rate:    2 * 1024, // 2KB/s
				Burst:   1024,     // 1KB burst
			},
		}, reader, nil, logger)

		tr := NewThrottledReader(reader, limiter, logger)
		buf := make([]byte, len(data))

		start := time.Now()
		n, err := tr.Read(buf)
		duration := time.Since(start)

		assert.NoError(t, err)
		assert.Equal(t, len(data), n)

		// Should take at least 1.5 seconds to read 4KB at 2KB/s
		minExpectedDuration := 1500 * time.Millisecond
		assert.True(t, duration >= minExpectedDuration,
			"Should take at least %v to read %d bytes at 2KB/s, took %v",
			minExpectedDuration, len(data), duration)
	})
}

func TestThrottledWriter(t *testing.T) {
	logger := setupTestLogger()

	// Test with small buffer (no pooling)
	t.Run("small buffer", func(t *testing.T) {
		var buf bytes.Buffer
		limiter := NewLimiter(&config.AppConfig{
			Throttle: config.ThrottleConfig{
				Enabled: true,
				Rate:    1024 * 1024, // 1MB/s
				Burst:   64 * 1024,   // 64KB
			},
		}, nil, &buf, logger)

		tw := NewThrottledWriter(&buf, limiter, logger)
		data := []byte("test data")
		n, err := tw.Write(data)
		assert.NoError(t, err)
		assert.Equal(t, len(data), n)
		assert.Equal(t, data, buf.Bytes())
	})

	// Test with large buffer (uses pool)
	t.Run("large buffer", func(t *testing.T) {
		var buf bytes.Buffer
		limiter := NewLimiter(&config.AppConfig{
			Throttle: config.ThrottleConfig{
				Enabled: true,
				Rate:    1024 * 1024, // 1MB/s
				Burst:   64 * 1024,   // 64KB
			},
		}, nil, &buf, logger)

		tw := NewThrottledWriter(&buf, limiter, logger)
		data := make([]byte, 8*1024) // 8KB
		for i := range data {
			data[i] = byte(i % 256)
		}
		n, err := tw.Write(data)
		assert.NoError(t, err)
		assert.Equal(t, len(data), n)
		assert.Equal(t, data, buf.Bytes())
	})

	// Test rate limiting
	t.Run("rate limiting", func(t *testing.T) {
		var buf bytes.Buffer
		limiter := NewLimiter(&config.AppConfig{
			Throttle: config.ThrottleConfig{
				Enabled: true,
				Rate:    2 * 1024, // 2KB/s
				Burst:   1024,     // 1KB burst
			},
		}, nil, &buf, logger)

		tw := NewThrottledWriter(&buf, limiter, logger)
		data := make([]byte, 4*1024) // 4KB

		start := time.Now()
		n, err := tw.Write(data)
		duration := time.Since(start)

		assert.NoError(t, err)
		assert.Equal(t, len(data), n)

		// Should take at least 1.5 seconds to write 4KB at 2KB/s
		minExpectedDuration := 1500 * time.Millisecond
		assert.True(t, duration >= minExpectedDuration,
			"Should take at least %v to write %d bytes at 2KB/s, took %v",
			minExpectedDuration, len(data), duration)
	})
}

func TestThrottledReadWriter(t *testing.T) {
	logger := setupTestLogger()

	t.Run("read and write", func(t *testing.T) {
		data := make([]byte, 8*1024) // 8KB
		for i := range data {
			data[i] = byte(i % 256)
		}

		// Create a ReadWriter that reads from data and writes to buf
		reader := bytes.NewReader(data)
		var writer bytes.Buffer
		rw := struct {
			io.Reader
			io.Writer
		}{reader, &writer}

		limiter := NewLimiter(&config.AppConfig{
			Throttle: config.ThrottleConfig{
				Enabled: true,
				Rate:    1024 * 1024, // 1MB/s
				Burst:   64 * 1024,   // 64KB
			},
		}, reader, &writer, logger)

		trw := NewThrottledReadWriter(rw, limiter, logger)

		// Read and write the data
		buf := make([]byte, len(data))
		n, err := trw.Read(buf)
		assert.NoError(t, err)
		assert.Equal(t, len(data), n)
		assert.Equal(t, data, buf)

		n, err = trw.Write(buf)
		assert.NoError(t, err)
		assert.Equal(t, len(data), n)
		assert.Equal(t, data, writer.Bytes())
	})
}
