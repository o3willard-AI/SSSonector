package throttle

import (
	"bytes"
	"testing"
	"time"

	"github.com/o3willard-AI/SSSonector/internal/config/types"
	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"
)

func setupLimiter(cfg *types.AppConfig) (*Limiter, *bytes.Buffer, *bytes.Buffer) {
	reader := &bytes.Buffer{}
	writer := &bytes.Buffer{}
	logger, _ := zap.NewDevelopment()

	if cfg == nil {
		cfg = &types.AppConfig{
			Throttle: types.NewThrottleConfig(),
		}
	}

	return NewLimiter(cfg, reader, writer, logger), reader, writer
}

func TestNewLimiter(t *testing.T) {
	// Test with nil config
	limiter, _, _ := setupLimiter(nil)
	assert.NotNil(t, limiter)
	assert.False(t, limiter.enabled)

	// Get metrics to check rate and burst
	inMetrics, outMetrics := limiter.GetMetrics()
	assert.Equal(t, float64(0), inMetrics.Rate)
	assert.Equal(t, float64(0), inMetrics.Burst)
	assert.Equal(t, float64(0), outMetrics.Rate)
	assert.Equal(t, float64(0), outMetrics.Burst)

	// Test with disabled config
	cfg := &types.AppConfig{
		Throttle: &types.ThrottleConfig{
			Enabled: false,
			Rate:    1000,
			Burst:   100,
		},
	}
	limiter, _, _ = setupLimiter(cfg)
	assert.NotNil(t, limiter)
	assert.False(t, limiter.enabled)

	// Get metrics to check rate and burst
	inMetrics, outMetrics = limiter.GetMetrics()
	assert.Equal(t, float64(0), inMetrics.Rate)
	assert.Equal(t, float64(0), inMetrics.Burst)
	assert.Equal(t, float64(0), outMetrics.Rate)
	assert.Equal(t, float64(0), outMetrics.Burst)

	// Test with enabled config
	cfg = &types.AppConfig{
		Throttle: &types.ThrottleConfig{
			Enabled: true,
			Rate:    1000,
			Burst:   100,
		},
	}
	limiter, _, _ = setupLimiter(cfg)
	assert.NotNil(t, limiter)
	assert.True(t, limiter.enabled)

	// Get metrics to check rate and burst
	inMetrics, outMetrics = limiter.GetMetrics()
	assert.Equal(t, float64(1000)*tcpOverheadFactor, inMetrics.Rate)
	assert.Equal(t, float64(100)*tcpOverheadFactor, inMetrics.Burst)
	assert.Equal(t, float64(1000)*tcpOverheadFactor, outMetrics.Rate)
	assert.Equal(t, float64(100)*tcpOverheadFactor, outMetrics.Burst)
}

func TestLimiterUpdate(t *testing.T) {
	// Test with nil config
	limiter, _, _ := setupLimiter(nil)
	assert.NotNil(t, limiter)
	assert.False(t, limiter.enabled)

	// Get initial metrics
	inMetrics, outMetrics := limiter.GetMetrics()
	assert.Equal(t, float64(0), inMetrics.Rate)
	assert.Equal(t, float64(0), inMetrics.Burst)
	assert.Equal(t, float64(0), outMetrics.Rate)
	assert.Equal(t, float64(0), outMetrics.Burst)

	// Update with enabled config
	cfg := &types.AppConfig{
		Throttle: &types.ThrottleConfig{
			Enabled: true,
			Rate:    1000,
			Burst:   100,
		},
	}
	limiter.Update(cfg)
	assert.True(t, limiter.enabled)

	// Get updated metrics
	inMetrics, outMetrics = limiter.GetMetrics()
	assert.Equal(t, float64(1000)*tcpOverheadFactor, inMetrics.Rate)
	assert.Equal(t, float64(100)*tcpOverheadFactor, inMetrics.Burst)
	assert.Equal(t, float64(1000)*tcpOverheadFactor, outMetrics.Rate)
	assert.Equal(t, float64(100)*tcpOverheadFactor, outMetrics.Burst)

	// Update with disabled config
	cfg = &types.AppConfig{
		Throttle: &types.ThrottleConfig{
			Enabled: false,
			Rate:    2000,
			Burst:   200,
		},
	}
	limiter.Update(cfg)
	assert.False(t, limiter.enabled)

	// Get updated metrics
	inMetrics, outMetrics = limiter.GetMetrics()
	assert.Equal(t, float64(2000)*tcpOverheadFactor, inMetrics.Rate)
	assert.Equal(t, float64(200)*tcpOverheadFactor, inMetrics.Burst)
	assert.Equal(t, float64(2000)*tcpOverheadFactor, outMetrics.Rate)
	assert.Equal(t, float64(200)*tcpOverheadFactor, outMetrics.Burst)
}

func TestLimiterWait(t *testing.T) {
	// Test with disabled limiter
	limiter, _, _ := setupLimiter(nil)
	start := time.Now()
	err := limiter.Wait(true, 1000)
	assert.NoError(t, err)
	elapsed := time.Since(start)
	assert.Less(t, elapsed, 10*time.Millisecond)

	// Test with enabled limiter
	cfg := &types.AppConfig{
		Throttle: &types.ThrottleConfig{
			Enabled: true,
			Rate:    1000,
			Burst:   100,
		},
	}
	limiter, _, _ = setupLimiter(cfg)
	start = time.Now()
	err = limiter.Wait(true, 1000)
	assert.NoError(t, err)
	elapsed = time.Since(start)
	assert.GreaterOrEqual(t, elapsed, time.Second)
}
