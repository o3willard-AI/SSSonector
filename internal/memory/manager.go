package memory

import (
	"context"
	"fmt"
	"runtime"
	"runtime/debug"
	"sync"
	"sync/atomic"
	"time"

	"go.uber.org/zap"
)

// MemoryManager manages memory usage and optimization
type MemoryManager struct {
	// Configuration
	maxMemoryBytes  uint64
	gcThreshold     float64
	monitorInterval time.Duration

	// State
	currentMemory   uint64
	peakMemory      uint64
	gcCount         uint32
	allocationCount uint64

	// Pools
	bufferPool interface{} // We'll integrate with our existing buffer pool

	// Callbacks
	onMemoryLimit    func(current, limit uint64) error
	onGCTrigger      func()
	onMemoryPressure func(level float64) error

	// Metrics
	lock        sync.RWMutex
	lastGCTime  time.Time
	memoryStats MemoryStats
	logger      *zap.Logger
	stopCh      chan struct{}
}

// MemoryStats represents current memory statistics
type MemoryStats struct {
	CurrentMemory  uint64
	PeakMemory     uint64
	SystemMemory   uint64
	GCTriggerRatio float64
	GCPauseAvg     time.Duration
	AllocCount     uint64
	LastGC         time.Time
	GCCount        uint32
	GCFraction     float64
	ObjectCount    uint64
	PauseTimeNs    []uint64
}

// MemPressure describes memory pressure levels
type MemPressure int

const (
	MemPressureNone MemPressure = iota
	MemPressureLow
	MemPressureMedium
	MemPressureHigh
	MemPressureCritical
)

// MemManagerConfig represents memory manager configuration
type MemManagerConfig struct {
	MaxMemoryBytes   uint64
	GCThreshold      float64 // GC trigger threshold (0.6-0.9 recommended)
	MonitorInterval  time.Duration
	OnMemoryLimit    func(uint64, uint64) error
	OnGCTrigger      func()
	OnMemoryPressure func(float64) error
	Logger           *zap.Logger
}

// NewMemoryManager creates a new memory manager
func NewMemoryManager(cfg *MemManagerConfig) *MemoryManager {
	if cfg == nil {
		cfg = getDefaultMemConfig()
	}

	mgr := &MemoryManager{
		maxMemoryBytes:   cfg.MaxMemoryBytes,
		gcThreshold:      cfg.GCThreshold,
		monitorInterval:  cfg.MonitorInterval,
		onMemoryLimit:    cfg.OnMemoryLimit,
		onGCTrigger:      cfg.OnGCTrigger,
		onMemoryPressure: cfg.OnMemoryPressure,
		logger:           cfg.Logger,
		stopCh:           make(chan struct{}),
		memoryStats:      MemoryStats{},
	}

	// Initialize stats
	mgr.updateStats()

	if cfg.Logger == nil {
		mgr.logger = zap.NewNop()
	}

	return mgr
}

// Start begins memory monitoring
func (m *MemoryManager) Start(ctx context.Context) error {
	go m.monitorLoop(ctx)

	m.logger.Info("Memory manager started",
		zap.Uint64("max_memory", m.maxMemoryBytes),
		zap.Float64("gc_threshold", m.gcThreshold),
		zap.Duration("monitor_interval", m.monitorInterval))

	return nil
}

// Stop stops memory monitoring
func (m *MemoryManager) Stop() {
	close(m.stopCh)
}

// monitorLoop continuously monitors memory usage
func (m *MemoryManager) monitorLoop(ctx context.Context) {
	ticker := time.NewTicker(m.monitorInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-m.stopCh:
			return
		case <-ticker.C:
			m.checkMemoryUsage()
			m.updateStats()
		}
	}
}

// checkMemoryUsage monitors current memory usage and takes action if needed
func (m *MemoryManager) checkMemoryUsage() {
	stats := runtime.MemStats{}
	runtime.ReadMemStats(&stats)

	currentMemory := stats.Alloc

	// Update peak memory
	atomic.StoreUint64(&m.peakMemory, max(m.peakMemory, currentMemory))
	atomic.StoreUint64(&m.currentMemory, currentMemory)

	// Check memory limit
	if m.maxMemoryBytes > 0 && currentMemory > m.maxMemoryBytes {
		m.handleMemoryLimit(currentMemory, m.maxMemoryBytes)
		return
	}

	// Check GC threshold
	if stats.GCCPUFraction > m.gcThreshold {
		m.handleGCTrigger(stats.GCCPUFraction)
	}

	// Check memory pressure
	pressure := m.calculatePressure()
	if pressure >= MemPressureMedium {
		m.handleMemoryPressure(float64(pressure) / 4.0) // Normalize to 0-1
	}

	// Force GC if necessary
	if stats.NumGC > 0 && time.Since(m.lastGCTime) > 60*time.Second {
		if stats.GCCPUFraction > 0.5 { // High GC pressure
			m.triggerGC()
		}
	}
}

// handleMemoryLimit is called when memory limit is exceeded
func (m *MemoryManager) handleMemoryLimit(current, limit uint64) {
	m.logger.Warn("Memory limit exceeded",
		zap.Uint64("current", current),
		zap.Uint64("limit", limit),
		zap.String("excess", fmt.Sprintf("%.2f%%", float64(current-limit)/float64(limit)*100)))

	if m.onMemoryLimit != nil {
		if err := m.onMemoryLimit(current, limit); err != nil {
			m.logger.Error("Memory limit callback failed",
				zap.Error(err))
		}
	}

	// Force GC as first action
	m.triggerGC()
}

// handleGCTrigger is called when GC threshold is exceeded
func (m *MemoryManager) handleGCTrigger(gcFraction float64) {
	if m.onGCTrigger != nil {
		m.onGCTrigger()
	}

	debug.SetGCPercent(int(gcFraction * 100))
	m.triggerGC()

	m.logger.Info("GC trigger handled",
		zap.Float64("gc_fraction", gcFraction))
}

// handleMemoryPressure is called when memory pressure is detected
func (m *MemoryManager) handleMemoryPressure(level float64) {
	if m.onMemoryPressure != nil {
		if err := m.onMemoryPressure(level); err != nil {
			m.logger.Error("Memory pressure callback failed",
				zap.Error(err),
				zap.Float64("pressure_level", level))
		}
	}

	// Additional actions based on pressure level
	if level > 0.8 { // Very high pressure
		m.forceGCAndConcurrentGCOff()
	} else if level > 0.6 {
		m.triggerGC()
	}
}

// calculatePressure calculates current memory pressure level
func (m *MemoryManager) calculatePressure() MemPressure {
	stats := runtime.MemStats{}
	runtime.ReadMemStats(&stats)

	// Calculate pressure based on multiple factors
	memoryPressure := float64(stats.Alloc) / float64(stats.Sys) // Allocation vs system memory
	gcPressure := stats.GCCPUFraction                           // GC CPU fraction

	combinedPressure := (memoryPressure + gcPressure) / 2

	switch {
	case combinedPressure > 0.8:
		return MemPressureCritical
	case combinedPressure > 0.6:
		return MemPressureHigh
	case combinedPressure > 0.4:
		return MemPressureMedium
	case combinedPressure > 0.2:
		return MemPressureLow
	default:
		return MemPressureNone
	}
}

// triggerGC forces a garbage collection
func (m *MemoryManager) triggerGC() {
	runtime.GC()
	m.lastGCTime = time.Now()
	atomic.AddUint32(&m.gcCount, 1)
}

// forceGCAndConcurrentGCOff forces GC and disables concurrent GC temporarily
func (m *MemoryManager) forceGCAndConcurrentGCOff() {
	// Temporarily disable concurrent GC
	runtime.GC()
	runtime.GC()
}

// updateStats updates current memory statistics
func (m *MemoryManager) updateStats() {
	m.lock.Lock()
	defer m.lock.Unlock()

	stats := runtime.MemStats{}
	runtime.ReadMemStats(&stats)

	m.memoryStats = MemoryStats{
		CurrentMemory:  stats.Alloc,
		PeakMemory:     atomic.LoadUint64(&m.peakMemory),
		SystemMemory:   stats.Sys,
		GCTriggerRatio: float64(stats.NextGC) / float64(stats.Sys),
		GCPauseAvg:     time.Duration(avgPauseTime(stats.PauseNs[:])),
		AllocCount:     atomic.LoadUint64(&m.allocationCount),
		LastGC:         m.lastGCTime,
		GCCount:        stats.NumGC,
		GCFraction:     stats.GCCPUFraction,
		ObjectCount:    stats.HeapObjects,
		PauseTimeNs:    make([]uint64, len(stats.PauseNs)),
	}

	copy(m.memoryStats.PauseTimeNs, stats.PauseNs[:])
}

// GetMemoryStats returns current memory statistics
func (m *MemoryManager) GetMemoryStats() MemoryStats {
	m.lock.RLock()
	defer m.lock.RUnlock()
	return m.memoryStats
}

// GetCurrentMemory returns current memory usage
func (m *MemoryManager) GetCurrentMemory() uint64 {
	stats := runtime.MemStats{}
	runtime.ReadMemStats(&stats)
	return stats.Alloc
}

// GetPeakMemory returns peak memory usage
func (m *MemoryManager) GetPeakMemory() uint64 {
	return atomic.LoadUint64(&m.peakMemory)
}

// GetPressureLevel returns current memory pressure level
func (m *MemoryManager) GetPressureLevel() MemPressure {
	return m.calculatePressure()
}

// Alloc allocates memory and tracks allocation
func (m *MemoryManager) Alloc(size uint64) []byte {
	atomic.AddUint64(&m.allocationCount, 1)
	buf := make([]byte, size)
	return buf
}

// SetMaxMemory sets the maximum memory limit
func (m *MemoryManager) SetMaxMemory(limit uint64) {
	atomic.StoreUint64(&m.maxMemoryBytes, limit)
}

// SetGCThreshold sets the GC trigger threshold
func (m *MemoryManager) SetGCThreshold(threshold float64) {
	m.gcThreshold = threshold
}

// getDefaultMemConfig returns default memory manager configuration
func getDefaultMemConfig() *MemManagerConfig {
	return &MemManagerConfig{
		MaxMemoryBytes:  1024 * 1024 * 1024, // 1GB default
		GCThreshold:     0.7,                // 70% GC threshold
		MonitorInterval: 1 * time.Second,
		OnMemoryLimit: func(curr, limit uint64) error {
			// Default action: log warning
			return fmt.Errorf("memory limit exceeded: %d > %d", curr, limit)
		},
		OnGCTrigger: func() {
			// Default: no action
		},
		OnMemoryPressure: func(level float64) error {
			// Default: no action
			return nil
		},
	}
}

// Helper functions
func max(a, b uint64) uint64 {
	if a > b {
		return a
	}
	return b
}

func avgPauseTime(pauses []uint64) uint64 {
	if len(pauses) == 0 {
		return 0
	}

	var sum uint64
	for _, pause := range pauses {
		sum += pause
	}
	return sum / uint64(len(pauses))
}
