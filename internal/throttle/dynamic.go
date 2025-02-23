package throttle

import (
	"sync"
	"time"

	"github.com/o3willard-AI/SSSonector/internal/config/types"
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
	cooldown    types.Duration
	logger      *zap.Logger
}

// NewDynamicLimiter creates a new dynamic rate limiter
func NewDynamicLimiter(cfg *types.AppConfig, limiter *Limiter, logger *zap.Logger) *DynamicLimiter {
	baseRate := float64(cfg.Throttle.Rate)
	dl := &DynamicLimiter{
		Limiter:    limiter,
		minRate:    baseRate / 2,                   // Default min rate is half of base rate
		maxRate:    baseRate * 2,                   // Default max rate is double base rate
		lastAdjust: time.Now().Add(-time.Hour),     // Start with expired cooldown
		cooldown:   types.NewDuration(time.Second), // Default 1 second cooldown
		logger:     logger,
	}

	// Start with base rate
	dl.setRate(baseRate)
	return dl
}

// IncreaseRate increases the rate by the specified percentage
func (dl *DynamicLimiter) IncreaseRate(percent float64) bool {
	dl.adjustMu.Lock()
	defer dl.adjustMu.Unlock()

	if time.Since(dl.lastAdjust) < dl.cooldown.Duration {
		return false // Prevent too frequent adjustments
	}

	// Get current rate from metrics
	dl.Limiter.mu.RLock()
	currentRate := dl.Limiter.inMetrics.Rate
	dl.Limiter.mu.RUnlock()

	// Calculate new rate
	newRate := currentRate * (1 + percent/100)

	// Cap at maximum rate
	if newRate > dl.maxRate {
		newRate = dl.maxRate
	}

	// Apply new rate and update adjustment tracking
	dl.setRate(newRate)
	dl.lastAdjust = time.Now()
	dl.adjustCount++

	dl.logger.Info("Increased rate",
		zap.Float64("old_rate", currentRate),
		zap.Float64("new_rate", newRate),
		zap.Float64("percent", percent),
	)

	return true
}

// DecreaseRate decreases the rate by the specified percentage
func (dl *DynamicLimiter) DecreaseRate(percent float64) bool {
	dl.adjustMu.Lock()
	defer dl.adjustMu.Unlock()

	if time.Since(dl.lastAdjust) < dl.cooldown.Duration {
		return false // Prevent too frequent adjustments
	}

	// Get current rate from metrics
	dl.Limiter.mu.RLock()
	currentRate := dl.Limiter.inMetrics.Rate
	dl.Limiter.mu.RUnlock()

	// Calculate new rate
	newRate := currentRate * (1 - percent/100)

	// Cap at minimum rate
	if newRate < dl.minRate {
		newRate = dl.minRate
	}

	// Apply new rate and update adjustment tracking
	dl.setRate(newRate)
	dl.lastAdjust = time.Now()
	dl.adjustCount++

	dl.logger.Info("Decreased rate",
		zap.Float64("old_rate", currentRate),
		zap.Float64("new_rate", newRate),
		zap.Float64("percent", percent),
	)

	return true
}

// setRate updates the rate in the underlying limiter
func (dl *DynamicLimiter) setRate(rate float64) {
	// Calculate burst as 10% of rate
	burst := rate * 0.1

	// Apply TCP overhead factor
	adjustedRate := rate * tcpOverheadFactor
	adjustedBurst := burst * tcpOverheadFactor

	// Lock the base limiter
	dl.Limiter.mu.Lock()
	defer dl.Limiter.mu.Unlock()

	// Update token buckets with adjusted values
	if dl.inBucket != nil {
		dl.inBucket.Update(adjustedRate, adjustedBurst)
	} else {
		dl.inBucket = NewTokenBucket(adjustedRate, adjustedBurst)
	}

	if dl.outBucket != nil {
		dl.outBucket.Update(adjustedRate, adjustedBurst)
	} else {
		dl.outBucket = NewTokenBucket(adjustedRate, adjustedBurst)
	}

	// Update base limiter metrics with base (unadjusted) values
	dl.Limiter.inMetrics = LimiterMetrics{
		Rate:      rate,
		Burst:     burst,
		LimitHits: dl.Limiter.inMetrics.LimitHits,
	}
	dl.Limiter.outMetrics = LimiterMetrics{
		Rate:      rate,
		Burst:     burst,
		LimitHits: dl.Limiter.outMetrics.LimitHits,
	}
	dl.Limiter.enabled = true

	dl.logger.Info("Updated rate limiter",
		zap.Float64("base_rate", rate),
		zap.Float64("base_burst", burst),
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
	dl.lastAdjust = time.Now().Add(-time.Hour) // Reset cooldown
}

// SetCooldown sets the cooldown duration between rate adjustments
func (dl *DynamicLimiter) SetCooldown(d types.Duration) {
	dl.adjustMu.Lock()
	defer dl.adjustMu.Unlock()
	dl.cooldown = d
	dl.lastAdjust = time.Now().Add(-d.Duration) // Reset cooldown
}

// GetCooldown returns the current cooldown duration
func (dl *DynamicLimiter) GetCooldown() types.Duration {
	dl.adjustMu.RLock()
	defer dl.adjustMu.RUnlock()
	return dl.cooldown
}
