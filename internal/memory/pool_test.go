package memory

import (
	"testing"
	"time"

	"go.uber.org/zap"
)

const (
	size4K  = 4 * 1024
	size8K  = 8 * 1024
	size16K = 16 * 1024
	size32K = 32 * 1024
	size64K = 64 * 1024
)

func TestBufferPool_BasicOperations(t *testing.T) {
	logger, _ := zap.NewDevelopment()
	manager := NewManager(100, time.Second, logger)
	pool := NewBufferPool(manager, logger)

	// Test Get
	buf := pool.Get(size4K)
	if buf == nil {
		t.Fatal("Expected buffer, got nil")
	}
	if len(buf) != size4K {
		t.Errorf("Expected buffer size %d, got %d", size4K, len(buf))
	}

	// Test Put
	pool.Put(buf)
}

func TestBufferPool_SizeRange(t *testing.T) {
	logger, _ := zap.NewDevelopment()
	manager := NewManager(100, time.Second, logger)
	pool := NewBufferPool(manager, logger)

	// Test default size range
	minSize, maxSize := pool.GetSizeRange()
	if minSize != DefaultMinSize {
		t.Errorf("Expected min size %d, got %d", DefaultMinSize, minSize)
	}
	if maxSize != DefaultMaxSize {
		t.Errorf("Expected max size %d, got %d", DefaultMaxSize, maxSize)
	}

	// Test setting size range
	err := pool.SetSizeRange(size8K, size32K)
	if err != nil {
		t.Fatalf("Failed to set size range: %v", err)
	}

	minSize, maxSize = pool.GetSizeRange()
	if minSize != size8K {
		t.Errorf("Expected min size %d, got %d", size8K, minSize)
	}
	if maxSize != size32K {
		t.Errorf("Expected max size %d, got %d", size32K, maxSize)
	}
}

func TestBufferPool_MemoryLimit(t *testing.T) {
	logger, _ := zap.NewDevelopment()
	manager := NewManager(1, time.Second, logger) // 1MB limit
	pool := NewBufferPool(manager, logger)

	// Test allocation within limit
	buf := pool.Get(size4K)
	if buf == nil {
		t.Fatal("Expected buffer, got nil")
	}

	// Test allocation exceeding limit
	buf2 := pool.Get(2 * 1024 * 1024) // 2MB
	if buf2 != nil {
		t.Error("Expected nil buffer for allocation exceeding limit")
	}

	// Release memory
	pool.Put(buf)
}

func TestBufferPool_Cleanup(t *testing.T) {
	logger, _ := zap.NewDevelopment()
	manager := NewManager(100, time.Second, logger)
	pool := NewBufferPool(manager, logger)

	// Allocate some buffers
	bufs := make([][]byte, 10)
	for i := 0; i < 10; i++ {
		bufs[i] = pool.Get(size4K)
		if bufs[i] == nil {
			t.Fatalf("Failed to allocate buffer %d", i)
		}
	}

	// Return buffers to pool
	for i := 0; i < 10; i++ {
		pool.Put(bufs[i])
	}

	// Trigger cleanup
	pool.cleanup()

	// Verify memory is released
	if manager.GetCurrentMemory() != 0 {
		t.Errorf("Expected 0 current memory, got %d", manager.GetCurrentMemory())
	}
}

func TestBufferPool_Concurrent(t *testing.T) {
	logger, _ := zap.NewDevelopment()
	manager := NewManager(100, time.Second, logger)
	pool := NewBufferPool(manager, logger)

	done := make(chan bool)
	const goroutines = 10

	for i := 0; i < goroutines; i++ {
		go func() {
			for j := 0; j < 100; j++ {
				buf := pool.Get(size4K)
				if buf != nil {
					pool.Put(buf)
				}
			}
			done <- true
		}()
	}

	// Wait for all goroutines
	for i := 0; i < goroutines; i++ {
		<-done
	}

	// Verify memory is released
	if manager.GetCurrentMemory() != 0 {
		t.Errorf("Expected 0 current memory, got %d", manager.GetCurrentMemory())
	}
}
