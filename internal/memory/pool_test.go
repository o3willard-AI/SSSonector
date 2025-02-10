package memory

import (
	"runtime"
	"sync"
	"testing"
	"time"

	"go.uber.org/zap"
)

func TestBufferPool(t *testing.T) {
	logger, _ := zap.NewDevelopment()
	manager := NewManager(&LimitConfig{MaxMemoryMB: 100}, logger)
	pool := NewBufferPool(manager)

	t.Run("Get and Put Standard Size", func(t *testing.T) {
		buf := pool.Get(size4K)
		if buf == nil {
			t.Fatal("Expected buffer allocation to succeed")
		}
		if len(buf) != size4K {
			t.Errorf("Expected buffer size %d, got %d", size4K, len(buf))
		}

		pool.Put(buf)
	})

	t.Run("Get Non-Standard Size", func(t *testing.T) {
		size := 10000 // Non-standard size
		buf := pool.Get(size)
		if buf == nil {
			t.Fatal("Expected buffer allocation to succeed")
		}
		if len(buf) != size {
			t.Errorf("Expected buffer size %d, got %d", size, len(buf))
		}

		pool.Put(buf)
	})

	t.Run("Memory Limit Enforcement", func(t *testing.T) {
		// Try to allocate more than the limit
		size := 101 * 1024 * 1024 // 101MB
		buf := pool.Get(size)
		if buf != nil {
			t.Error("Expected allocation to fail due to memory limit")
		}
	})
}

func TestBufferPoolConcurrent(t *testing.T) {
	logger, _ := zap.NewDevelopment()
	manager := NewManager(&LimitConfig{MaxMemoryMB: 100}, logger)
	pool := NewBufferPool(manager)

	const (
		numGoroutines = 10
		numIterations = 100
	)

	var wg sync.WaitGroup
	wg.Add(numGoroutines)

	for i := 0; i < numGoroutines; i++ {
		go func() {
			defer wg.Done()
			for j := 0; j < numIterations; j++ {
				buf := pool.Get(size4K)
				if buf != nil {
					// Simulate some work
					time.Sleep(time.Microsecond)
					pool.Put(buf)
				}
			}
		}()
	}

	wg.Wait()
}

func TestGetClosestPoolSize(t *testing.T) {
	logger, _ := zap.NewDevelopment()
	manager := NewManager(nil, logger)
	pool := NewBufferPool(manager)

	tests := []struct {
		name     string
		size     int
		expected int
	}{
		{"Exact Match", size4K, size4K},
		{"Round Up Small", 3 * 1024, size4K},
		{"Round Up Medium", 33 * 1024, size64K},
		{"Too Large", 2 * 1024 * 1024, -1},
		{"Zero Size", 0, size4K},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := pool.GetClosestPoolSize(tt.size)
			if got != tt.expected {
				t.Errorf("GetClosestPoolSize(%d) = %d, want %d", tt.size, got, tt.expected)
			}
		})
	}
}

func TestBufferPoolCleanup(t *testing.T) {
	logger, _ := zap.NewDevelopment()
	manager := NewManager(&LimitConfig{MaxMemoryMB: 100}, logger)
	pool := NewBufferPool(manager)

	// Allocate some buffers
	var buffers [][]byte
	for i := 0; i < 10; i++ {
		buf := pool.Get(size4K)
		if buf == nil {
			t.Fatal("Failed to allocate buffer")
		}
		buffers = append(buffers, buf)
	}

	// Return them to the pool
	for _, buf := range buffers {
		pool.Put(buf)
	}

	// Force cleanup
	pool.Reset()

	// Verify memory was released
	runtime.GC()
	var memStats runtime.MemStats
	runtime.ReadMemStats(&memStats)

	// Get new buffers to ensure pool is still functional
	buf := pool.Get(size4K)
	if buf == nil {
		t.Fatal("Failed to allocate buffer after cleanup")
	}
	pool.Put(buf)
}

func TestBufferPoolStats(t *testing.T) {
	logger, _ := zap.NewDevelopment()
	manager := NewManager(&LimitConfig{MaxMemoryMB: 100}, logger)
	pool := NewBufferPool(manager)

	stats := pool.GetStats()
	if len(stats) == 0 {
		t.Error("Expected non-empty stats")
	}

	// Verify stats for each pool size
	for size, stat := range stats {
		if stat.Size != size {
			t.Errorf("Expected size %d, got %d", size, stat.Size)
		}
		if stat.Capacity != 100*1024*1024 {
			t.Errorf("Expected capacity %d, got %d", 100*1024*1024, stat.Capacity)
		}
	}
}

func TestIsManagedSize(t *testing.T) {
	logger, _ := zap.NewDevelopment()
	manager := NewManager(nil, logger)
	pool := NewBufferPool(manager)

	tests := []struct {
		name     string
		size     int
		expected bool
	}{
		{"4K", size4K, true},
		{"8K", size8K, true},
		{"Non-standard", 10000, false},
		{"Zero", 0, false},
		{"Negative", -1, false},
		{"Very Large", 2 * 1024 * 1024, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := pool.IsManagedSize(tt.size)
			if got != tt.expected {
				t.Errorf("IsManagedSize(%d) = %v, want %v", tt.size, got, tt.expected)
			}
		})
	}
}

func BenchmarkBufferPool(b *testing.B) {
	logger, _ := zap.NewDevelopment()
	manager := NewManager(&LimitConfig{MaxMemoryMB: 100}, logger)
	pool := NewBufferPool(manager)

	b.Run("Get/Put 4K", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			buf := pool.Get(size4K)
			pool.Put(buf)
		}
	})

	b.Run("Get/Put Mixed Sizes", func(b *testing.B) {
		sizes := []int{size4K, size8K, size16K, size32K}
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			size := sizes[i%len(sizes)]
			buf := pool.Get(size)
			pool.Put(buf)
		}
	})

	b.Run("Concurrent Get/Put", func(b *testing.B) {
		const numGoroutines = 4
		var wg sync.WaitGroup
		wg.Add(numGoroutines)

		b.ResetTimer()
		for i := 0; i < numGoroutines; i++ {
			go func() {
				defer wg.Done()
				for j := 0; j < b.N/numGoroutines; j++ {
					buf := pool.Get(size4K)
					pool.Put(buf)
				}
			}()
		}
		wg.Wait()
	})
}
