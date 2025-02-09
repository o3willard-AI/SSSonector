package throttle

import (
	"sync"
	"time"

	"go.uber.org/zap"
)

const (
	defaultAdjustmentInterval = 5 * time.Second
	defaultIncreaseThreshold  = 0.8               // 80% utilization
	defaultDecreaseThreshold  = 0.2               // 20% utilization
	defaultIncreaseRatio      = 1.25              // 25% increase
	defaultDecreaseRatio      = 0.8               // 20% decrease
	minRate                   = 1024 * 512        // 512KB/s minimum
	maxRate                   = 1024 * 1024 * 100 // 100MB/s maximum
)

// DynamicAdjuster implements dynamic rate adjustment based on utilization
type DynamicAdjuster struct {
	enabled            bool
	minRate            float64
	maxRate            float64
	adjustmentInterval time.Duration
	increaseThreshold  float64
	decreaseThreshold  float64
	increaseRatio      float64
	decreaseRatio      float64
	lastAdjustment     time.Time
	mu                 sync.Mutex
	logger             *zap.Logger
}

// NewDynamicAdjuster creates a new dynamic rate adjuster
func NewDynamicAdjuster(logger *zap.Logger) *DynamicAdjuster {
	return &DynamicAdjuster{
		enabled:            true,
		minRate:            minRate,
		maxRate:            maxRate,
		adjustmentInterval: defaultAdjustmentInterval,
		increaseThreshold:  defaultIncreaseThreshold,
		decreaseThreshold:  defaultDecreaseThreshold,
		increaseRatio:      defaultIncreaseRatio,
		decreaseRatio:      defaultDecreaseRatio,
		lastAdjustment:     time.Now(),
		logger:             logger,
	}
}

// Configure sets the dynamic adjuster parameters
func (d *DynamicAdjuster) Configure(minRate, maxRate float64, interval time.Duration) {
	d.mu.Lock()
	defer d.mu.Unlock()

	if minRate > 0 {
		d.minRate = minRate
	}
	if maxRate > 0 {
		d.maxRate = maxRate
	}
	if interval > 0 {
		d.adjustmentInterval = interval
	}
}

// SetEnabled enables or disables dynamic adjustment
func (d *DynamicAdjuster) SetEnabled(enabled bool) {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.enabled = enabled
}

// AdjustRate adjusts the rate based on current utilization
func (d *DynamicAdjuster) AdjustRate(currentRate float64, utilization float64) float64 {
	d.mu.Lock()
	defer d.mu.Unlock()

	if !d.enabled {
		return currentRate
	}

	now := time.Now()
	if now.Sub(d.lastAdjustment) < d.adjustmentInterval {
		return currentRate
	}
	d.lastAdjustment = now

	var newRate float64
	switch {
	case utilization > d.increaseThreshold:
		// High utilization - increase rate
		newRate = currentRate * d.increaseRatio
		d.logger.Info("Increasing rate due to high utilization",
			zap.Float64("utilization", utilization),
			zap.Float64("old_rate", currentRate),
			zap.Float64("new_rate", newRate))

	case utilization < d.decreaseThreshold:
		// Low utilization - decrease rate
		newRate = currentRate * d.decreaseRatio
		d.logger.Info("Decreasing rate due to low utilization",
			zap.Float64("utilization", utilization),
			zap.Float64("old_rate", currentRate),
			zap.Float64("new_rate", newRate))

	default:
		// Utilization within acceptable range
		return currentRate
	}

	// Ensure rate stays within bounds
	if newRate < d.minRate {
		newRate = d.minRate
	}
	if newRate > d.maxRate {
		newRate = d.maxRate
	}

	return newRate
}

// UtilizationStats tracks utilization statistics
type UtilizationStats struct {
	bytesProcessed uint64
	startTime      time.Time
	mu             sync.Mutex
}

// NewUtilizationStats creates a new utilization stats tracker
func NewUtilizationStats() *UtilizationStats {
	return &UtilizationStats{
		startTime: time.Now(),
	}
}

// AddBytes records processed bytes
func (s *UtilizationStats) AddBytes(n uint64) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.bytesProcessed += n
}

// GetUtilization calculates current utilization ratio
func (s *UtilizationStats) GetUtilization(targetRate float64) float64 {
	s.mu.Lock()
	defer s.mu.Unlock()

	elapsed := time.Since(s.startTime).Seconds()
	if elapsed == 0 {
		return 0
	}

	actualRate := float64(s.bytesProcessed) / elapsed
	utilization := actualRate / targetRate

	// Reset stats for next period
	s.bytesProcessed = 0
	s.startTime = time.Now()

	return utilization
}
