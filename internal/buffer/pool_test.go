package buffer

import (
	"testing"
)

func TestPool(t *testing.T) {
	tests := []struct {
		name     string
		cfg      Config
		getSize  int
		wantSize int
	}{
		{
			name:     "default configuration",
			cfg:      Config{},
			getSize:  0,
			wantSize: 1500,
		},
		{
			name:     "custom min size",
			cfg:      Config{MinSize: 2000},
			getSize:  0,
			wantSize: 2000,
		},
		{
			name:     "specific size request",
			cfg:      Config{MinSize: 1500},
			getSize:  2000,
			wantSize: 2000,
		},
		{
			name:     "large size request",
			cfg:      Config{MinSize: 1500, MaxSize: 3000},
			getSize:  4000,
			wantSize: 4000,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pool := NewPool(tt.cfg)
			buf := pool.Get(tt.getSize)

			if len(buf) != tt.wantSize {
				t.Errorf("Pool.Get(%d) got buffer length = %d, want %d", tt.getSize, len(buf), tt.wantSize)
			}

			// Test Put and reuse
			pool.Put(buf)
			buf2 := pool.Get(tt.getSize)
			if len(buf2) != tt.wantSize {
				t.Errorf("Pool.Get(%d) after Put got buffer length = %d, want %d", tt.getSize, len(buf2), tt.wantSize)
			}
		})
	}
}

func TestPoolMTU(t *testing.T) {
	tests := []struct {
		name    string
		cfg     Config
		mtu     int
		wantLen int
	}{
		{
			name:    "zero mtu",
			cfg:     Config{MinSize: 1500},
			mtu:     0,
			wantLen: 1500,
		},
		{
			name:    "custom mtu",
			cfg:     Config{MinSize: 1500},
			mtu:     2000,
			wantLen: 2000,
		},
		{
			name:    "negative mtu",
			cfg:     Config{MinSize: 1500},
			mtu:     -1,
			wantLen: 1500,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pool := NewPool(tt.cfg)
			buf := pool.GetWithMTU(tt.mtu)

			if len(buf) != tt.wantLen {
				t.Errorf("Pool.GetWithMTU(%d) got buffer length = %d, want %d", tt.mtu, len(buf), tt.wantLen)
			}
		})
	}
}

func TestPoolPutInvalidSize(t *testing.T) {
	pool := NewPool(Config{MinSize: 1500, MaxSize: 3000})

	// Test putting buffer that's too small
	smallBuf := make([]byte, 1000)
	pool.Put(smallBuf)

	// Test putting buffer that's too large
	largeBuf := make([]byte, 4000)
	pool.Put(largeBuf)

	// Verify pool still works correctly
	buf := pool.Get(0)
	if len(buf) != 1500 {
		t.Errorf("Pool.Get(0) after invalid Puts got buffer length = %d, want 1500", len(buf))
	}
}

func BenchmarkPool(b *testing.B) {
	pool := NewPool(Config{MinSize: 1500, MaxSize: 3000})
	sizes := []int{1500, 2000, 2500, 3000}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		size := sizes[i%len(sizes)]
		buf := pool.Get(size)
		pool.Put(buf)
	}
}
