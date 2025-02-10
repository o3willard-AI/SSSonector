package throttle

import (
	"sync"
	"time"

	"github.com/o3willard-AI/SSSonector/internal/config"
	"go.uber.org/zap"
)

// DynamicLimiter extends the base Limiter with dynamic rate adjustment
type DynamicLimiter struct {
	*Limiter
	minRate     float64
	maxRate     float64
	adjustMu    sync.RWMutex
	lastAdjust  time.Time
	adjustCount int
	logger      *zap.Logger
}

// NewDynamicLimiter creates a new dynamic rate limiter
func NewDynamicLimiter(cfg *config.AppConfig, limiter *Limiter, logger *zap.Logger) *DynamicLimiter {
	baseRate := float64(cfg.Throttle.Rate)
	dl := &DynamicLimiter{
		Limiter:    limiter,
		minRate:    baseRate / 2, // Default min rate is half of base rate
		maxRate:    baseRate * 2, // Default max rate is double base rate
		lastAdjust: time.Now(),
		logger:     logger,
	}

	// Start with base rate
	dl.setRate(baseRate)
	return dl
}

// IncreaseRate increases the rate by the specified percentage
func (dl *DynamicLimiter) IncreaseRate(percent float64) {
	dl.adjustMu.Lock()
	defer dl.adjustMu.Unlock()

	if time.Since(dl.lastAdjust) < time.Second {
		return // Prevent too frequent adjustments
	}

	inMetrics, _ := dl.GetMetrics()
	currentRate := inMetrics.Rate
	newRate := currentRate * (1 + percent/100)

	// Cap at maximum rate
	if newRate > dl.maxRate {
		newRate = dl.maxRate
	}

	dl.setRate(newRate)
	dl.lastAdjust = time.Now()
	dl.adjustCount++

	dl.logger.Info("Increased rate",
		zap.Float64("old_rate", currentRate),
		zap.Float64("new_rate", newRate),
		zap.Float64("percent", percent),
	)
}

// DecreaseRate decreases the rate by the specified percentage
func (dl *DynamicLimiter) DecreaseRate(percent float64) {
	dl.adjustMu.Lock()
	defer dl.adjustMu.Unlock()

	if time.Since(dl.lastAdjust) < time.Second {
		return // Prevent too frequent adjustments
	}

	inMetrics, _ := dl.GetMetrics()
	currentRate := inMetrics.Rate
	newRate := currentRate * (1 - percent/100)

	// Cap at minimum rate
	if newRate < dl.minRate {
		newRate = dl.minRate
	}

	dl.setRate(newRate)
	dl.lastAdjust = time.Now()
	dl.adjustCount++

	dl.logger.Info("Decreased rate",
		zap.Float64("old_rate", currentRate),
		zap.Float64("new_rate", newRate),
		zap.Float64("percent", percent),
	)
}

// setRate updates the rate in the underlying limiter
func (dl *DynamicLimiter) setRate(rate float64) {
	// Calculate burst as 10% of rate
	burst := rate * 0.1

	// Apply TCP overhead factor
	adjustedRate := rate * tcpOverheadFactor
	adjustedBurst := burst * tcpOverheadFactor

	dl.mu.Lock()
	defer dl.mu.Unlock()

	// Update token buckets directly
	dl.inBucket.Update(adjustedRate, adjustedBurst)
	dl.outBucket.Update(adjustedRate, adjustedBurst)

	// Update metrics (store the pre-adjusted values)
	dl.inMetrics = LimiterMetrics{
		Rate:      rate,
		Burst:     burst,
		LimitHits: dl.inMetrics.LimitHits,
	}
	dl.outMetrics = LimiterMetrics{
		Rate:      rate,
		Burst:     burst,
		LimitHits: dl.outMetrics.LimitHits,
	}

	// Update base limiter state
	dl.enabled = true

	dl.logger.Info("Updated rate limiter",
		zap.Float64("rate", rate),
		zap.Float64("burst", burst),
		zap.Float64("adjusted_rate", adjustedRate),
		zap.Float64("adjusted_burst", adjustedBurst),
	)
}

// GetAdjustCount returns the number of rate adjustments made
func (dl *DynamicLimiter) GetAdjustCount() int {
	dl.adjustMu.RLock()
	defer dl.adjustMu.RUnlock()
	return dl.adjustCount
}

// ResetAdjustCount resets the adjustment counter
func (dl *DynamicLimiter) ResetAdjustCount() {
	dl.adjustMu.Lock()
	defer dl.adjustMu.Unlock()
	dl.adjustCount = 0
}
