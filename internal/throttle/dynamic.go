package throttle

import (
	"fmt"
	"math"
	"sync"
	"time"

	"github.com/o3willard-AI/SSSonector/internal/config/types"
	"go.uber.org/zap"
)

// DynamicLimiter implements advanced rate limiting with dynamic adjustment
type DynamicLimiter struct {
	limiter *Limiter
	config  *types.ThrottleConfig

	// Dynamic adjustment state
	targetUtilization  float64
	minRate            float64
	maxRate            float64
	adjustmentInterval time.Duration
	adjustmentStep     float64
	smoothingFactor    float64

	// TCP overhead compensation
	tcpOverheadFactor float64
	baselineOverhead  int // bytes
	estimatedMSS      int // maximum segment size

	// Burst control with 100ms windows
	burstWindow      time.Duration
	burstTokens      int64
	burstTokensPerMs float64

	// Monitoring and metrics
	metrics        DynamicMetrics
	lastAdjustment time.Time

	// Synchronization
	mu     sync.RWMutex
	stopCh chan struct{}
	logger *zap.Logger
}

// DynamicMetrics tracks advanced rate limiting statistics
type DynamicMetrics struct {
	CurrentRate       float64
	CurrentBurst      int64
	TargetUtilization float64
	ActualUtilization float64
	AdjustmentCount   int64
	BurstHitCount     int64
	Efficiency        float64
	TCPEfficiency     float64
	LastAdjustment    time.Time
}

// NewDynamicLimiter creates a new dynamic rate limiter with enhanced features
func NewDynamicLimiter(cfg *types.ThrottleConfig, reader Reader, writer Writer, logger *zap.Logger) *DynamicLimiter {
	limiter := &Limiter{
		enabled: cfg.Enabled,
		reader:  reader,
		writer:  writer,
		logger:  logger,
	}

	// Initialize with TCP overhead compensation
	baseRate := float64(cfg.Rate)
	limiter.inBucket = NewTokenBucket(baseRate*tcpOverheadFactor, float64(cfg.Burst*tcpOverheadFactor))
	limiter.outBucket = NewTokenBucket(baseRate*tcpOverheadFactor, float64(cfg.Burst*tcpOverheadFactor))

	dl := &DynamicLimiter{
		limiter:            limiter,
		config:             cfg,
		targetUtilization:  0.8,                     // 80% target utilization
		minRate:            float64(cfg.Rate) * 0.5, // 50% of configured rate
		maxRate:            float64(cfg.Rate) * 1.5, // 150% of configured rate
		adjustmentInterval: time.Minute,
		adjustmentStep:     0.1, // 10% adjustment per step
		smoothingFactor:    0.3, // exponential moving average

		// TCP overhead configuration
		tcpOverheadFactor: 1.05, // 5% TCP overhead (more accurate than 10%)
		baselineOverhead:  40,   // basic TCP/IP headerSize
		estimatedMSS:      1460, // typical MSS for Ethernet

		// Burst control (100ms windows)
		burstWindow: 100 * time.Millisecond,
		stopCh:      make(chan struct{}),

		logger: logger,
	}

	// Initialize metrics
	dl.metrics = DynamicMetrics{
		CurrentRate:       baseRate,
		CurrentBurst:      int64(cfg.Burst),
		TargetUtilization: dl.targetUtilization,
		LastAdjustment:    time.Now(),
	}

	return dl
}

// Start starts the dynamic adjustment routine
func (dl *DynamicLimiter) Start() {
	go dl.dynamicAdjustmentLoop()
}

// Stop stops the dynamic adjustment routine
func (dl *DynamicLimiter) Stop() {
	close(dl.stopCh)
}

// dynamicAdjustmentLoop continuously adjusts rate limits based on utilization
func (dl *DynamicLimiter) dynamicAdjustmentLoop() {
	ticker := time.NewTicker(dl.adjustmentInterval)
	defer ticker.Stop()

	for {
		select {
		case <-dl.stopCh:
			return
		case <-ticker.C:
			dl.adjustRateLimits()
		}
	}
}

// adjustRateLimits dynamically adjusts rate limits based on current utilization
func (dl *DynamicLimiter) adjustRateLimits() {
	dl.mu.Lock()
	defer dl.mu.Unlock()

	now := time.Now()
	inMetrics, outMetrics := dl.limiter.GetMetrics()

	// Calculate current utilization (hits / tokens per second)
	inUtilization := calculateUtilization(inMetrics)
	outUtilization := calculateUtilization(outMetrics)

	// Use the higher utilization for adjustment
	utilization := math.Max(inUtilization, outUtilization)

	// Apply exponential moving average smoothing
	dl.metrics.ActualUtilization = (dl.smoothingFactor * utilization) +
		((1 - dl.smoothingFactor) * dl.metrics.ActualUtilization)

	// Calculate adjustment factor
	adjustment := dl.calculateAdjustment()

	if math.Abs(adjustment) >= 0.01 { // Only adjust if change > 1%
		dl.applyAdjustment(adjustment)
		dl.metrics.AdjustmentCount++
		dl.metrics.LastAdjustment = now

		dl.logger.Info("Rate limit adjusted dynamically",
			zap.Float64("old_rate", dl.metrics.CurrentRate),
			zap.Float64("new_rate", dl.metrics.CurrentRate*(1+adjustment)),
			zap.Float64("utilization", dl.metrics.ActualUtilization),
			zap.Float64("target_utilization", dl.targetUtilization),
			zap.Float64("adjustment_factor", adjustment),
		)
	}

	// Update burst control
	dl.updateBurstControl()
}

// calculateAdjustment calculates the adjustment factor based on utilization
func (dl *DynamicLimiter) calculateAdjustment() float64 {
	actual := dl.metrics.ActualUtilization
	target := dl.targetUtilization

	var adjustment float64

	if actual > target+0.1 { // Over 10% above target - reduce rate
		adjustment = -dl.adjustmentStep
	} else if actual < target-0.1 { // Over 10% below target - increase rate
		adjustment = dl.adjustmentStep
	}

	// Limit adjustment bounds
	if adjustment < 0 {
		minAllowed := (dl.minRate - dl.metrics.CurrentRate) / dl.metrics.CurrentRate
		adjustment = math.Max(adjustment, minAllowed)
	} else if adjustment > 0 {
		maxAllowed := (dl.maxRate - dl.metrics.CurrentRate) / dl.metrics.CurrentRate
		adjustment = math.Min(adjustment, maxAllowed)
	}

	return adjustment
}

// applyAdjustment applies the calculated adjustment to the token buckets
func (dl *DynamicLimiter) applyAdjustment(factor float64) {
	newRate := dl.metrics.CurrentRate * (1 + factor)
	newBurst := int64(float64(dl.metrics.CurrentBurst) * (1 + factor))

	// Update token buckets
	dl.limiter.inBucket.Update(newRate*dl.tcpOverheadFactor, float64(newBurst)*dl.tcpOverheadFactor)
	dl.limiter.outBucket.Update(newRate*dl.tcpOverheadFactor, float64(newBurst)*dl.tcpOverheadFactor)

	// Update metrics
	dl.metrics.CurrentRate = newRate
	dl.metrics.CurrentBurst = newBurst
}

// updateBurstControl updates burst control using 100ms windows
func (dl *DynamicLimiter) updateBurstControl() {
	// Calculate tokens per millisecond for burst control
	burstTokensPerMs := dl.metrics.CurrentRate / 1000 // tokens per millisecond
	dl.burstTokensPerMs = burstTokensPerMs

	// Calculate maximum burst tokens for 100ms window
	dl.burstTokens = int64(burstTokensPerMs * float64(dl.burstWindow.Milliseconds()))
}

// calculateUtilization calculates utilization from limiter metrics
func calculateUtilization(metrics LimiterMetrics) float64 {
	if metrics.Rate <= 0 {
		return 0
	}

	// Estimate utilization based on limit hits vs available burst
	totalTokensPerSecond := metrics.Rate * 1.5 // rate + burst allowance
	usedRate := totalTokensPerSecond * 0.01    // rough estimate from limit hits

	return usedRate / metrics.Rate
}

// Read implements io.Reader with dynamic rate limiting
func (dl *DynamicLimiter) Read(p []byte) (n int, err error) {
	if !dl.limiter.enabled {
		return dl.limiter.reader.Read(p)
	}

	// Get buffer from pool with burst control
	buf := dl.getBurstControlledBuffer(len(p))
	defer dl.putBuffer(buf)

	// Read into buffer
	n, err = dl.limiter.reader.Read(buf)
	if err != nil {
		return n, err
	}

	data := buf[:n]

	// Apply TCP overhead compensation
	tcpOverhead := dl.calculateTCPOoverhead(n)
	actualTokens := float64(n + tcpOverhead)

	// Wait for tokens (including TCP overhead)
	if err := dl.limiter.Wait(true, int(actualTokens)); err != nil {
		return 0, err
	}

	// Copy data
	copy(p, data)
	return n, nil
}

// Write implements io.Writer with dynamic rate limiting
func (dl *DynamicLimiter) Write(p []byte) (n int, err error) {
	if !dl.limiter.enabled {
		return dl.limiter.writer.Write(p)
	}

	// Calculate TCP overhead compensation
	tcpOverhead := dl.calculateTCPOoverhead(len(p))
	actualTokens := float64(len(p) + tcpOverhead)

	// Wait for tokens (including TCP overhead)
	if err := dl.limiter.Wait(false, int(actualTokens)); err != nil {
		return 0, err
	}

	// Write data
	return dl.limiter.writer.Write(p)
}

// calculateTCPOoverhead calculates the estimated TCP/IP overhead for a payload
func (dl *DynamicLimiter) calculateTCPOoverhead(payloadSize int) int {
	if payloadSize == 0 {
		return dl.baselineOverhead
	}

	// Calculate number of segments needed
	segments := int(math.Ceil(float64(payloadSize) / float64(dl.estimatedMSS)))

	// Each TCP segment has IP headers + TCP headers
	overheadPerSegment := dl.baselineOverhead // 40 bytes

	return segments * overheadPerSegment
}

// getBurstControlledBuffer gets a buffer with burst control
func (dl *DynamicLimiter) getBurstControlledBuffer(size int) []byte {
	// Clamp size to prevent excessive bursting
	maxBurstSize := int(dl.burstTokensPerMs * float64(dl.burstWindow.Milliseconds()))
	if size > maxBurstSize {
		size = maxBurstSize
	}

	return dl.limiter.GetBuffer(size)
}

// putBuffer returns a buffer to the pool
func (dl *DynamicLimiter) putBuffer(buf []byte) {
	dl.limiter.PutBuffer(buf)
}

// GetDynamicMetrics returns current dynamic metrics
func (dl *DynamicLimiter) GetDynamicMetrics() DynamicMetrics {
	dl.mu.RLock()
	defer dl.mu.RUnlock()
	return dl.metrics
}

// GetEfficiency calculates system efficiency
func (dl *DynamicLimiter) GetEfficiency() float64 {
	dl.mu.RLock()
	defer dl.mu.RUnlock()

	// Calculate TCP efficiency (payload vs overhead)
	inMetrics, outMetrics := dl.limiter.GetMetrics()
	totalRate := inMetrics.Rate + outMetrics.Rate

	if totalRate == 0 {
		return 0
	}

	// Estimate actual data throughput vs allocated tokens
	dataThroughput := totalRate / dl.tcpOverheadFactor
	efficiency := dataThroughput / totalRate

	dl.metrics.Efficiency = efficiency
	dl.metrics.TCPEfficiency = efficiency

	return efficiency
}

// SetTargetUtilization sets the target utilization for dynamic adjustment
func (dl *DynamicLimiter) SetTargetUtilization(utilization float64) error {
	if utilization <= 0 || utilization > 1 {
		return fmt.Errorf("utilization must be between 0 and 1, got %f", utilization)
	}

	dl.mu.Lock()
	dl.targetUtilization = utilization
	dl.mu.Unlock()

	dl.logger.Info("Updated target utilization",
		zap.Float64("new_target", utilization))

	return nil
}

// GetCurrentConfig returns the current throttling configuration
func (dl *DynamicLimiter) GetCurrentConfig() (float64, int64) {
	dl.mu.RLock()
	defer dl.mu.RUnlock()
	return dl.metrics.CurrentRate, dl.metrics.CurrentBurst
}

// ForceAdjustment forces an immediate rate limit adjustment
func (dl *DynamicLimiter) ForceAdjustment() {
	go dl.adjustRateLimits()
}
