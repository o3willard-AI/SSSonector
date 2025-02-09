package throttle

import (
	"bytes"
	"io"
	"testing"
	"time"

	"github.com/o3willard-AI/SSSonector/internal/config"
	"go.uber.org/zap"
)

func TestBufferPool(t *testing.T) {
	// Test buffer size constraints
	t.Run("Buffer Size Constraints", func(t *testing.T) {
		// Test minimum size
		pool := NewBufferPool(1024, 10) // Try smaller than minimum
		buf := pool.Get()
		if len(buf) != minBufferSize {
			t.Errorf("Expected minimum buffer size %d, got %d", minBufferSize, len(buf))
		}

		// Test maximum size
		pool = NewBufferPool(2*1024*1024, 10) // Try larger than maximum
		buf = pool.Get()
		if len(buf) != maxBufferSize {
			t.Errorf("Expected maximum buffer size %d, got %d", maxBufferSize, len(buf))
		}
	})

	// Test buffer reuse
	t.Run("Buffer Reuse", func(t *testing.T) {
		pool := NewBufferPool(64*1024, 10)
		buf1 := pool.Get()
		pool.Put(buf1)
		buf2 := pool.Get()

		// Should get the same buffer back (by capacity)
		if cap(buf2) != cap(buf1) {
			t.Error("Buffer not reused from pool")
		}
	})
}

func TestThrottledReader(t *testing.T) {
	logger, _ := zap.NewDevelopment()
	cfg := &config.AppConfig{
		Throttle: config.ThrottleConfig{
			Enabled: true,
			Rate:    1024 * 1024, // 1MB/s
			Burst:   64 * 1024,   // 64KB burst
		},
	}

	data := make([]byte, 256*1024) // 256KB of data
	for i := range data {
		data[i] = byte(i % 256)
	}

	t.Run("Large Read with Buffer Pool", func(t *testing.T) {
		reader := bytes.NewReader(data)
		limiter := NewLimiter(cfg, reader, nil, logger)
		throttledReader := NewThrottledReader(reader, limiter, 64*1024, 10)

		buf := make([]byte, len(data))
		start := time.Now()
		n, err := throttledReader.Read(buf)

		if err != nil {
			t.Fatalf("Read failed: %v", err)
		}
		if n != len(data) {
			t.Errorf("Expected to read %d bytes, got %d", len(data), n)
		}
		if !bytes.Equal(buf[:n], data) {
			t.Error("Read data does not match source data")
		}

		// Verify rate limiting still works with buffer pool
		elapsed := time.Since(start)
		expectedMinTime := time.Duration(float64(len(data)) / float64(cfg.Throttle.Rate) * float64(time.Second))
		if elapsed < expectedMinTime {
			t.Errorf("Read completed too quickly: %v < %v", elapsed, expectedMinTime)
		}
	})

	t.Run("Small Reads Bypass Pool", func(t *testing.T) {
		reader := bytes.NewReader(data)
		limiter := NewLimiter(cfg, reader, nil, logger)
		throttledReader := NewThrottledReader(reader, limiter, 64*1024, 10)

		// Read small chunks that should bypass the buffer pool
		buf := make([]byte, 1024)
		n, err := throttledReader.Read(buf)

		if err != nil {
			t.Fatalf("Small read failed: %v", err)
		}
		if n != 1024 {
			t.Errorf("Expected to read 1024 bytes, got %d", n)
		}
		if !bytes.Equal(buf[:n], data[:n]) {
			t.Error("Small read data does not match source data")
		}
	})
}

func TestThrottledWriter(t *testing.T) {
	logger, _ := zap.NewDevelopment()
	cfg := &config.AppConfig{
		Throttle: config.ThrottleConfig{
			Enabled: true,
			Rate:    1024 * 1024, // 1MB/s
			Burst:   64 * 1024,   // 64KB burst
		},
	}

	data := make([]byte, 256*1024) // 256KB of data
	for i := range data {
		data[i] = byte(i % 256)
	}

	t.Run("Large Write with Buffer Pool", func(t *testing.T) {
		var buf bytes.Buffer
		limiter := NewLimiter(cfg, nil, &buf, logger)
		throttledWriter := NewThrottledWriter(&buf, limiter, 64*1024, 10)

		start := time.Now()
		n, err := throttledWriter.Write(data)

		if err != nil {
			t.Fatalf("Write failed: %v", err)
		}
		if n != len(data) {
			t.Errorf("Expected to write %d bytes, got %d", len(data), n)
		}
		if !bytes.Equal(buf.Bytes(), data) {
			t.Error("Written data does not match source data")
		}

		// Verify rate limiting still works with buffer pool
		elapsed := time.Since(start)
		expectedMinTime := time.Duration(float64(len(data)) / float64(cfg.Throttle.Rate) * float64(time.Second))
		if elapsed < expectedMinTime {
			t.Errorf("Write completed too quickly: %v < %v", elapsed, expectedMinTime)
		}
	})

	t.Run("Small Writes Bypass Pool", func(t *testing.T) {
		var buf bytes.Buffer
		limiter := NewLimiter(cfg, nil, &buf, logger)
		throttledWriter := NewThrottledWriter(&buf, limiter, 64*1024, 10)

		// Write small chunk that should bypass the buffer pool
		n, err := throttledWriter.Write(data[:1024])

		if err != nil {
			t.Fatalf("Small write failed: %v", err)
		}
		if n != 1024 {
			t.Errorf("Expected to write 1024 bytes, got %d", n)
		}
		if !bytes.Equal(buf.Bytes(), data[:n]) {
			t.Error("Small write data does not match source data")
		}
	})
}

func TestThrottledReadWriter(t *testing.T) {
	logger, _ := zap.NewDevelopment()
	cfg := &config.AppConfig{
		Throttle: config.ThrottleConfig{
			Enabled: true,
			Rate:    1024 * 1024, // 1MB/s
			Burst:   64 * 1024,   // 64KB burst
		},
	}

	data := make([]byte, 128*1024) // 128KB of data
	for i := range data {
		data[i] = byte(i % 256)
	}

	t.Run("Copy Through ReadWriter", func(t *testing.T) {
		var buf bytes.Buffer
		rw := struct {
			io.Reader
			io.Writer
		}{
			Reader: bytes.NewReader(data),
			Writer: &buf,
		}

		limiter := NewLimiter(cfg, rw, rw, logger)
		throttledRW := NewThrottledReadWriter(rw, limiter, 64*1024, 10)

		start := time.Now()
		n, err := io.Copy(throttledRW, throttledRW)

		if err != nil {
			t.Fatalf("Copy failed: %v", err)
		}
		if n != int64(len(data)) {
			t.Errorf("Expected to copy %d bytes, got %d", len(data), n)
		}
		if !bytes.Equal(buf.Bytes(), data) {
			t.Error("Copied data does not match source data")
		}

		// Verify rate limiting works in both directions
		elapsed := time.Since(start)
		expectedMinTime := time.Duration(float64(len(data)*2) / float64(cfg.Throttle.Rate) * float64(time.Second))
		if elapsed < expectedMinTime {
			t.Errorf("Copy completed too quickly: %v < %v", elapsed, expectedMinTime)
		}
	})
}
