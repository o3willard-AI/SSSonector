package throttle

import (
	"bytes"
	"io"
	"testing"
	"time"

	"github.com/o3willard-AI/SSSonector/internal/config/types"
	"go.uber.org/zap"
)

func TestThrottledReader(t *testing.T) {
	logger, _ := zap.NewDevelopment()

	t.Run("small buffer", func(t *testing.T) {
		cfg := &types.AppConfig{
			Throttle: &types.ThrottleConfig{
				Enabled: true,
				Rate:    1024 * 1024, // 1MB/s
				Burst:   1024 * 100,  // 100KB burst
			},
		}

		data := []byte("Hello, World!")
		reader := bytes.NewBuffer(data)
		limiter := NewLimiter(cfg, reader, nil, logger)

		buf := make([]byte, len(data))
		n, err := limiter.Read(buf)

		if err != nil {
			t.Errorf("Expected no error, got %v", err)
		}
		if n != len(data) {
			t.Errorf("Expected to read %d bytes, got %d", len(data), n)
		}
		if !bytes.Equal(buf[:n], data) {
			t.Errorf("Expected %q, got %q", data, buf[:n])
		}
	})

	t.Run("large buffer", func(t *testing.T) {
		cfg := &types.AppConfig{
			Throttle: &types.ThrottleConfig{
				Enabled: true,
				Rate:    1024 * 1024, // 1MB/s
				Burst:   1024 * 100,  // 100KB burst
			},
		}

		data := make([]byte, 1024*1024) // 1MB
		for i := range data {
			data[i] = byte(i % 256)
		}

		reader := bytes.NewBuffer(data)
		limiter := NewLimiter(cfg, reader, nil, logger)

		buf := make([]byte, len(data))
		n, err := limiter.Read(buf)

		if err != nil {
			t.Errorf("Expected no error, got %v", err)
		}
		if n != len(data) {
			t.Errorf("Expected to read %d bytes, got %d", len(data), n)
		}
		if !bytes.Equal(buf[:n], data) {
			t.Error("Data mismatch")
		}
	})

	t.Run("rate limiting", func(t *testing.T) {
		cfg := &types.AppConfig{
			Throttle: &types.ThrottleConfig{
				Enabled: true,
				Rate:    2048, // 2KB/s
				Burst:   1024, // 1KB burst
			},
		}

		data := make([]byte, 4096) // 4KB
		for i := range data {
			data[i] = byte(i % 256)
		}

		reader := bytes.NewBuffer(data)
		limiter := NewLimiter(cfg, reader, nil, logger)

		// Read should take at least 1.5s (4KB at 2KB/s + 10% buffer)
		start := time.Now()
		buf := make([]byte, len(data))
		n, err := limiter.Read(buf)

		elapsed := time.Since(start)
		if err != nil {
			t.Errorf("Expected no error, got %v", err)
		}
		if n != len(data) {
			t.Errorf("Expected to read %d bytes, got %d", len(data), n)
		}
		if elapsed < 1500*time.Millisecond {
			t.Errorf("Should take at least 1.5s to read 4096 bytes at 2KB/s took %v", elapsed)
		}
	})
}

func TestThrottledWriter(t *testing.T) {
	logger, _ := zap.NewDevelopment()

	t.Run("small buffer", func(t *testing.T) {
		cfg := &types.AppConfig{
			Throttle: &types.ThrottleConfig{
				Enabled: true,
				Rate:    1024 * 1024, // 1MB/s
				Burst:   1024 * 100,  // 100KB burst
			},
		}

		data := []byte("Hello, World!")
		writer := &bytes.Buffer{}
		limiter := NewLimiter(cfg, nil, writer, logger)

		n, err := limiter.Write(data)

		if err != nil {
			t.Errorf("Expected no error, got %v", err)
		}
		if n != len(data) {
			t.Errorf("Expected to write %d bytes, got %d", len(data), n)
		}
		if !bytes.Equal(writer.Bytes(), data) {
			t.Errorf("Expected %q, got %q", data, writer.Bytes())
		}
	})

	t.Run("large buffer", func(t *testing.T) {
		cfg := &types.AppConfig{
			Throttle: &types.ThrottleConfig{
				Enabled: true,
				Rate:    1024 * 1024, // 1MB/s
				Burst:   1024 * 100,  // 100KB burst
			},
		}

		data := make([]byte, 1024*1024) // 1MB
		for i := range data {
			data[i] = byte(i % 256)
		}

		writer := &bytes.Buffer{}
		limiter := NewLimiter(cfg, nil, writer, logger)

		n, err := limiter.Write(data)

		if err != nil {
			t.Errorf("Expected no error, got %v", err)
		}
		if n != len(data) {
			t.Errorf("Expected to write %d bytes, got %d", len(data), n)
		}
		if !bytes.Equal(writer.Bytes(), data) {
			t.Error("Data mismatch")
		}
	})

	t.Run("rate limiting", func(t *testing.T) {
		cfg := &types.AppConfig{
			Throttle: &types.ThrottleConfig{
				Enabled: true,
				Rate:    2048, // 2KB/s
				Burst:   1024, // 1KB burst
			},
		}

		data := make([]byte, 4096) // 4KB
		for i := range data {
			data[i] = byte(i % 256)
		}

		writer := &bytes.Buffer{}
		limiter := NewLimiter(cfg, nil, writer, logger)

		// Write should take at least 1.5s (4KB at 2KB/s + 10% buffer)
		start := time.Now()
		n, err := limiter.Write(data)

		elapsed := time.Since(start)
		if err != nil {
			t.Errorf("Expected no error, got %v", err)
		}
		if n != len(data) {
			t.Errorf("Expected to write %d bytes, got %d", len(data), n)
		}
		if elapsed < 1500*time.Millisecond {
			t.Errorf("Should take at least 1.5s to write 4096 bytes at 2KB/s took %v", elapsed)
		}
	})
}

func TestThrottledReadWriter(t *testing.T) {
	logger, _ := zap.NewDevelopment()

	t.Run("read and write", func(t *testing.T) {
		cfg := &types.AppConfig{
			Throttle: &types.ThrottleConfig{
				Enabled: true,
				Rate:    1024 * 1024, // 1MB/s
				Burst:   1024 * 100,  // 100KB burst
			},
		}

		data := []byte("Hello, World!")
		reader := bytes.NewBuffer(data)
		writer := &bytes.Buffer{}

		limiter := NewLimiter(cfg, reader, writer, logger)

		// Copy data through the limiter
		n, err := io.Copy(limiter, limiter)

		if err != nil {
			t.Errorf("Expected no error, got %v", err)
		}
		if n != int64(len(data)) {
			t.Errorf("Expected to copy %d bytes, got %d", len(data), n)
		}
		if !bytes.Equal(writer.Bytes(), data) {
			t.Errorf("Expected %q, got %q", data, writer.Bytes())
		}
	})
}
