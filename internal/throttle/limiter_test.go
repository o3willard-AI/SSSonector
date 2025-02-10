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
	inMetrics, _ := limiter.GetMetrics()
	expectedRate := float64(cfg.Throttle.Rate) * tcpOverheadFactor
	if inMetrics.Rate != expectedRate {
		t.Errorf("Expected effective rate %f, got %f", expectedRate, inMetrics.Rate)
	}

	// Test burst is 100ms worth of data
	expectedBurst := expectedRate * 0.1 // 100ms
	if inMetrics.Burst != expectedBurst {
		t.Errorf("Expected burst %f, got %f", expectedBurst, inMetrics.Burst)
	}
}

func TestLimiterRateEnforcement(t *testing.T) {
	logger, _ := zap.NewDevelopment()
	cfg := &config.AppConfig{
		Throttle: config.ThrottleConfig{
			Enabled: true,
			Rate:    100,  // 100 bytes/s
			Burst:   1000, // Large enough burst to not interfere
		},
	}

	// Create a small test data set
	data := make([]byte, 300)
	reader := bytes.NewReader(data)
	writer := &bytes.Buffer{}
	limiter := NewLimiter(cfg, reader, writer, logger)

	// Test read rate limiting
	start := time.Now()
	err := limiter.Wait(true, 200) // Wait for 200 bytes
	elapsed := time.Since(start)

	if err != nil {
		t.Fatalf("Wait failed: %v", err)
	}

	// Should take at least 1.8 seconds to read 200 bytes at 100 bytes/s
	minExpectedDuration := 1800 * time.Millisecond
	if elapsed < minExpectedDuration {
		t.Errorf("Rate limit not enforced: waited %v, expected at least %v", elapsed, minExpectedDuration)
	}
}

func TestLimiterTimeout(t *testing.T) {
	logger, _ := zap.NewDevelopment()
	cfg := &config.AppConfig{
		Throttle: config.ThrottleConfig{
			Enabled: true,
			Rate:    10,  // Very low rate
			Burst:   100, // Small burst
		},
	}

	// Create data that will trigger timeout
	data := make([]byte, 1000)
	reader := bytes.NewReader(data)
	writer := &bytes.Buffer{}
	limiter := NewLimiter(cfg, reader, writer, logger)

	// Try to read more data than possible within timeout
	start := time.Now()
	err := limiter.Wait(true, 1000)
	elapsed := time.Since(start)

	if err == nil || elapsed < defaultTimeout {
		t.Errorf("Expected timeout error after %v, got %v after %v", defaultTimeout, err, elapsed)
	}
}

func TestLimiterMetrics(t *testing.T) {
	logger, _ := zap.NewDevelopment()
	cfg := &config.AppConfig{
		Throttle: config.ThrottleConfig{
			Enabled: true,
			Rate:    100,
			Burst:   50,
		},
	}

	// Create data that will trigger rate limiting
	data := make([]byte, 200)
	reader := bytes.NewReader(data)
	writer := &bytes.Buffer{}
	limiter := NewLimiter(cfg, reader, writer, logger)

	// Trigger rate limiting
	_ = limiter.Wait(true, 150)
	time.Sleep(100 * time.Millisecond) // Give time for metrics to update

	inMetrics, _ := limiter.GetMetrics()
	if inMetrics.LimitHits == 0 {
		t.Error("Expected non-zero limit hits")
	}

	// Test rate values
	baseRate := float64(cfg.Throttle.Rate)
	expectedEffectiveRate := baseRate * tcpOverheadFactor
	if inMetrics.Rate != expectedEffectiveRate {
		t.Errorf("Expected effective rate %f, got %f", expectedEffectiveRate, inMetrics.Rate)
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

	inMetrics, _ := limiter.GetMetrics()
	expectedRate := float64(newCfg.Throttle.Rate) * tcpOverheadFactor
	if inMetrics.Rate != expectedRate {
		t.Errorf("Expected updated rate %f, got %f", expectedRate, inMetrics.Rate)
	}

	expectedBurst := expectedRate * 0.1 // 100ms
	if inMetrics.Burst != expectedBurst {
		t.Errorf("Expected updated burst %f, got %f", expectedBurst, inMetrics.Burst)
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

	// Wait should complete immediately when disabled
	start := time.Now()
	err := limiter.Wait(true, 1500)
	elapsed := time.Since(start)

	if err != nil {
		t.Fatalf("Wait failed: %v", err)
	}

	if elapsed > 100*time.Millisecond {
		t.Error("Rate limiting applied when disabled")
	}
}
