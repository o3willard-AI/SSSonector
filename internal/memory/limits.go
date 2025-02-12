package memory

import (
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"go.uber.org/zap"
)

const (
	// Default memory limits
	DefaultMaxMemoryMB     = 512
	DefaultMonitorInterval = time.Second * 5

	// Memory units
	MB = 1024 * 1024
)

// Manager handles memory allocation and limits
type Manager struct {
	maxBytes     uint64
	currentBytes uint64
	interval     time.Duration
	logger       *zap.Logger
	cleanupFuncs []func()
	mu           sync.RWMutex
	stopCh       chan struct{}
}

// NewManager creates a new memory manager
func NewManager(maxMemoryMB uint64, interval time.Duration, logger *zap.Logger) *Manager {
	if maxMemoryMB == 0 {
		maxMemoryMB = DefaultMaxMemoryMB
	}
	if interval == 0 {
		interval = DefaultMonitorInterval
	}
	if logger == nil {
		var err error
		logger, err = zap.NewDevelopment()
		if err != nil {
			panic(fmt.Sprintf("failed to create logger: %v", err))
		}
	}

	m := &Manager{
		maxBytes:     maxMemoryMB * MB,
		interval:     interval,
		logger:       logger,
		cleanupFuncs: make([]func(), 0),
		stopCh:       make(chan struct{}),
	}

	go m.monitor()

	return m
}

// CheckAndReserve checks if memory can be allocated and reserves it
func (m *Manager) CheckAndReserve(bytes uint64) bool {
	current := atomic.LoadUint64(&m.currentBytes)
	if current+bytes > m.maxBytes {
		m.logger.Warn("Memory allocation rejected",
			zap.Uint64("requested_bytes", bytes),
			zap.Uint64("current_mb", current/MB),
			zap.Uint64("max_mb", m.maxBytes/MB))
		return false
	}

	atomic.AddUint64(&m.currentBytes, bytes)
	return true
}

// Release releases previously reserved memory
func (m *Manager) Release(bytes uint64) {
	atomic.AddUint64(&m.currentBytes, ^(bytes - 1))
}

// RegisterCleanup registers a cleanup function
func (m *Manager) RegisterCleanup(cleanup func()) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.cleanupFuncs = append(m.cleanupFuncs, cleanup)
}

// Stop stops the memory manager
func (m *Manager) Stop() {
	close(m.stopCh)
}

// monitor periodically checks memory usage
func (m *Manager) monitor() {
	ticker := time.NewTicker(m.interval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			current := atomic.LoadUint64(&m.currentBytes)
			if current > m.maxBytes {
				m.logger.Info("Running memory cleanup",
					zap.Uint64("current_mb", current/MB),
					zap.Uint64("max_mb", m.maxBytes/MB),
					zap.Int("cleanup_funcs", len(m.cleanupFuncs)))

				m.runCleanup()
			}
		case <-m.stopCh:
			return
		}
	}
}

// runCleanup executes registered cleanup functions
func (m *Manager) runCleanup() {
	m.mu.RLock()
	defer m.mu.RUnlock()

	for _, cleanup := range m.cleanupFuncs {
		cleanup()
	}
}

// GetStats returns current memory statistics
func (m *Manager) GetStats() string {
	current := atomic.LoadUint64(&m.currentBytes)
	return fmt.Sprintf("Memory Usage: %d/%d MB (%.1f%%)",
		current/MB,
		m.maxBytes/MB,
		float64(current)/float64(m.maxBytes)*100)
}

// SetMaxMemory updates the maximum memory limit
func (m *Manager) SetMaxMemory(maxMemoryMB uint64) {
	atomic.StoreUint64(&m.maxBytes, maxMemoryMB*MB)
}

// GetMaxMemory returns the maximum memory limit in MB
func (m *Manager) GetMaxMemory() uint64 {
	return atomic.LoadUint64(&m.maxBytes) / MB
}

// GetCurrentMemory returns the current memory usage in MB
func (m *Manager) GetCurrentMemory() uint64 {
	return atomic.LoadUint64(&m.currentBytes) / MB
}
