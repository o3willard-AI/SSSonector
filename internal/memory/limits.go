package memory

import (
	"fmt"
	"runtime"
	"sync"
	"sync/atomic"
	"time"

	"go.uber.org/zap"
)

const (
	defaultMaxMemoryMB     = 512
	defaultCheckInterval   = 30 * time.Second
	defaultCleanupInterval = 5 * time.Second
	defaultSoftLimitRatio  = 0.85 // 85% of max memory
)

// LimitConfig holds memory limit configuration
type LimitConfig struct {
	// Maximum memory usage in MB
	MaxMemoryMB int64

	// Interval between memory usage checks
	CheckInterval time.Duration

	// Interval between cleanup attempts when near limit
	CleanupInterval time.Duration

	// Ratio of max memory that triggers cleanup (0.0-1.0)
	SoftLimitRatio float64
}

// DefaultLimitConfig returns default memory limit configuration
func DefaultLimitConfig() *LimitConfig {
	return &LimitConfig{
		MaxMemoryMB:     defaultMaxMemoryMB,
		CheckInterval:   defaultCheckInterval,
		CleanupInterval: defaultCleanupInterval,
		SoftLimitRatio:  defaultSoftLimitRatio,
	}
}

// Manager manages memory limits and cleanup
type Manager struct {
	config *LimitConfig
	logger *zap.Logger

	// Current memory stats
	stats struct {
		sync.RWMutex
		current    int64
		max        int64
		cleanups   int64
		rejections int64
	}

	// Cleanup callback functions
	cleanupFuncs []func()

	// Control channels
	stop    chan struct{}
	stopped chan struct{}
}

// NewManager creates a new memory limit manager
func NewManager(config *LimitConfig, logger *zap.Logger) *Manager {
	if config == nil {
		config = DefaultLimitConfig()
	}

	m := &Manager{
		config:  config,
		logger:  logger,
		stop:    make(chan struct{}),
		stopped: make(chan struct{}),
	}

	go m.monitor()
	return m
}

// RegisterCleanupFunc registers a function to be called during cleanup
func (m *Manager) RegisterCleanupFunc(f func()) {
	m.stats.Lock()
	m.cleanupFuncs = append(m.cleanupFuncs, f)
	m.stats.Unlock()
}

// CheckAndReserve checks if allocating size bytes would exceed limits
// Returns true if allocation is allowed, false if it would exceed limits
func (m *Manager) CheckAndReserve(size int64) bool {
	current := atomic.LoadInt64(&m.stats.current)
	if current+size > m.config.MaxMemoryMB*1024*1024 {
		atomic.AddInt64(&m.stats.rejections, 1)
		m.logger.Warn("Memory allocation rejected",
			zap.Int64("requested_bytes", size),
			zap.Int64("current_mb", current/(1024*1024)),
			zap.Int64("max_mb", m.config.MaxMemoryMB),
		)
		return false
	}

	atomic.AddInt64(&m.stats.current, size)
	return true
}

// Release releases previously reserved memory
func (m *Manager) Release(size int64) {
	atomic.AddInt64(&m.stats.current, -size)
}

// GetStats returns current memory statistics
func (m *Manager) GetStats() (current, max, cleanups, rejections int64) {
	m.stats.RLock()
	defer m.stats.RUnlock()
	return m.stats.current, m.stats.max, m.stats.cleanups, m.stats.rejections
}

// Stop stops the memory monitor
func (m *Manager) Stop() {
	close(m.stop)
	<-m.stopped
}

// monitor periodically checks memory usage and triggers cleanup if needed
func (m *Manager) monitor() {
	defer close(m.stopped)

	checkTicker := time.NewTicker(m.config.CheckInterval)
	defer checkTicker.Stop()

	cleanupTicker := time.NewTicker(m.config.CleanupInterval)
	defer cleanupTicker.Stop()

	var memStats runtime.MemStats

	for {
		select {
		case <-m.stop:
			return

		case <-checkTicker.C:
			runtime.ReadMemStats(&memStats)
			current := int64(memStats.Alloc)
			atomic.StoreInt64(&m.stats.current, current)

			if current > atomic.LoadInt64(&m.stats.max) {
				atomic.StoreInt64(&m.stats.max, current)
			}

			// Check if we're approaching the limit
			if float64(current) > float64(m.config.MaxMemoryMB*1024*1024)*m.config.SoftLimitRatio {
				m.cleanup()
			}

		case <-cleanupTicker.C:
			// Periodically run cleanup regardless of current usage
			m.cleanup()
		}
	}
}

// cleanup runs registered cleanup functions
func (m *Manager) cleanup() {
	m.stats.RLock()
	funcs := make([]func(), len(m.cleanupFuncs))
	copy(funcs, m.cleanupFuncs)
	m.stats.RUnlock()

	if len(funcs) == 0 {
		return
	}

	m.logger.Info("Running memory cleanup",
		zap.Int64("current_mb", atomic.LoadInt64(&m.stats.current)/(1024*1024)),
		zap.Int64("max_mb", m.config.MaxMemoryMB),
		zap.Int("cleanup_funcs", len(funcs)),
	)

	for _, f := range funcs {
		f()
	}

	atomic.AddInt64(&m.stats.cleanups, 1)

	// Force garbage collection after cleanup
	runtime.GC()
}

// FormatStats formats current memory statistics as a string
func (m *Manager) FormatStats() string {
	current, max, cleanups, rejections := m.GetStats()
	return fmt.Sprintf(
		"Memory Stats:\n"+
			"  Current: %d MB\n"+
			"  Max: %d MB\n"+
			"  Limit: %d MB\n"+
			"  Cleanups: %d\n"+
			"  Rejections: %d",
		current/(1024*1024),
		max/(1024*1024),
		m.config.MaxMemoryMB,
		cleanups,
		rejections,
	)
}
