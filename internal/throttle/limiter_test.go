package throttle

import (
	"bytes"
	"testing"
	"time"

	"github.com/o3willard-AI/SSSonector/internal/config"
	"go.uber.org/zap"
)

func TestLimiterWithTCPOverhead(t *testing.T) {
	logger, _ := zap.NewDevelopment()
	cfg := &config.AppConfig{
		Throttle: config.ThrottleConfig{
			Enabled: true,
			Rate:    1000, // 1000 bytes/s
			Burst:   100,  // 100 bytes burst
		},
	}

	reader := bytes.NewReader(make([]byte, 2000))
	writer := &bytes.Buffer{}
	limiter := NewLimiter(cfg, reader, writer, logger)

	// Test that effective rate includes TCP overhead
	metrics := limiter.GetMetrics()
	expectedRate := float64(cfg.Throttle.Rate) * tcpOverheadFactor
	if metrics.InRate != expectedRate {
		t.Errorf("Expected effective rate %f, got %f", expectedRate, metrics.InRate)
	}

	// Test burst is 100ms worth of data
	expectedBurst := expectedRate * 0.1 // 100ms
	if metrics.InBurst != expectedBurst {
		t.Errorf("Expected burst %f, got %f", expectedBurst, metrics.InBurst)
	}
}

func TestLimiterRateEnforcement(t *testing.T) {
	logger, _ := zap.NewDevelopment()
	cfg := &config.AppConfig{
		Throttle: config.ThrottleConfig{
			Enabled: true,
			Rate:    1000, // 1000 bytes/s
			Burst:   100,  // 100 bytes burst
		},
	}

	data := make([]byte, 2000)
	reader := bytes.NewReader(data)
	writer := &bytes.Buffer{}
	limiter := NewLimiter(cfg, reader, writer, logger)

	// Test read rate limiting
	start := time.Now()
	buf := make([]byte, 1500)
	n, err := limiter.Read(buf)
	elapsed := time.Since(start)

	if err != nil {
		t.Fatalf("Read failed: %v", err)
	}

	// Should take at least 1.5 seconds to read 1500 bytes at 1000 bytes/s
	if elapsed < 1500*time.Millisecond {
		t.Errorf("Rate limit not enforced: read %d bytes in %v", n, elapsed)
	}
}

func TestLimiterTimeout(t *testing.T) {
	logger, _ := zap.NewDevelopment()
	cfg := &config.AppConfig{
		Throttle: config.ThrottleConfig{
			Enabled: true,
			Rate:    10, // Very low rate to trigger timeout
			Burst:   1,
		},
	}

	data := make([]byte, 1000)
	reader := bytes.NewReader(data)
	writer := &bytes.Buffer{}
	limiter := NewLimiter(cfg, reader, writer, logger)

	// Try to read more data than possible within timeout
	buf := make([]byte, 1000)
	_, err := limiter.Read(buf)

	if err == nil {
		t.Error("Expected timeout error, got nil")
	}
}

func TestLimiterMetrics(t *testing.T) {
	logger, _ := zap.NewDevelopment()
	cfg := &config.AppConfig{
		Throttle: config.ThrottleConfig{
			Enabled: true,
			Rate:    1000,
			Burst:   100,
		},
	}

	reader := bytes.NewReader(make([]byte, 2000))
	writer := &bytes.Buffer{}
	limiter := NewLimiter(cfg, reader, writer, logger)

	// Trigger some limit hits
	buf := make([]byte, 2000)
	limiter.Read(buf)

	metrics := limiter.GetMetrics()
	if metrics.InLimitHits == 0 {
		t.Error("Expected non-zero limit hits")
	}

	// Test rate values
	baseRate := float64(cfg.Throttle.Rate)
	expectedEffectiveRate := baseRate * tcpOverheadFactor
	if metrics.InRate != expectedEffectiveRate {
		t.Errorf("Expected effective rate %f, got %f", expectedEffectiveRate, metrics.InRate)
	}
}

func TestLimiterUpdate(t *testing.T) {
	logger, _ := zap.NewDevelopment()
	cfg := &config.AppConfig{
		Throttle: config.ThrottleConfig{
			Enabled: true,
			Rate:    1000,
			Burst:   100,
		},
	}

	reader := bytes.NewReader(make([]byte, 2000))
	writer := &bytes.Buffer{}
	limiter := NewLimiter(cfg, reader, writer, logger)

	// Update configuration
	newCfg := &config.AppConfig{
		Throttle: config.ThrottleConfig{
			Enabled: true,
			Rate:    2000,
			Burst:   200,
		},
	}
	limiter.Update(newCfg)

	metrics := limiter.GetMetrics()
	expectedRate := float64(newCfg.Throttle.Rate) * tcpOverheadFactor
	if metrics.InRate != expectedRate {
		t.Errorf("Expected updated rate %f, got %f", expectedRate, metrics.InRate)
	}

	expectedBurst := expectedRate * 0.1 // 100ms
	if metrics.InBurst != expectedBurst {
		t.Errorf("Expected updated burst %f, got %f", expectedBurst, metrics.InBurst)
	}
}

func TestLimiterDisabled(t *testing.T) {
	logger, _ := zap.NewDevelopment()
	cfg := &config.AppConfig{
		Throttle: config.ThrottleConfig{
			Enabled: false,
			Rate:    1000,
			Burst:   100,
		},
	}

	data := make([]byte, 2000)
	reader := bytes.NewReader(data)
	writer := &bytes.Buffer{}
	limiter := NewLimiter(cfg, reader, writer, logger)

	// Read should complete immediately when disabled
	start := time.Now()
	buf := make([]byte, 1500)
	_, err := limiter.Read(buf)
	elapsed := time.Since(start)

	if err != nil {
		t.Fatalf("Read failed: %v", err)
	}

	if elapsed > 100*time.Millisecond {
		t.Error("Rate limiting applied when disabled")
	}
}
