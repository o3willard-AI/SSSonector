package tunnel

import (
	"io"
	"net"
	"sync/atomic"
	"testing"
)

func TestCountingConnCountsBytes(t *testing.T) {
	c1, c2 := net.Pipe()
	defer c1.Close()
	defer c2.Close()

	var readBytes, writeBytes atomic.Int64
	cc := newCountingConn(c1, &readBytes, &writeBytes)

	payload := []byte("hello tunnel metrics")

	go func() {
		if _, err := c2.Write(payload); err != nil {
			t.Errorf("pipe write: %v", err)
		}
	}()

	buf := make([]byte, len(payload))
	if _, err := io.ReadFull(cc, buf); err != nil {
		t.Fatalf("ReadFull: %v", err)
	}
	if got := readBytes.Load(); got != int64(len(payload)) {
		t.Errorf("readBytes = %d, want %d", got, len(payload))
	}
	if got := writeBytes.Load(); got != 0 {
		t.Errorf("writeBytes = %d, want 0", got)
	}

	readDone := make(chan struct{})
	go func() {
		defer close(readDone)
		buf2 := make([]byte, len(payload))
		if _, err := io.ReadFull(c2, buf2); err != nil {
			t.Errorf("ReadFull on peer: %v", err)
		}
	}()

	if _, err := cc.Write(payload); err != nil {
		t.Fatalf("Write: %v", err)
	}
	<-readDone

	if got := writeBytes.Load(); got != int64(len(payload)) {
		t.Errorf("writeBytes = %d, want %d", got, len(payload))
	}
}

func TestTransferHalfCloseThroughCountingConn(t *testing.T) {
	c1, c2 := net.Pipe()
	defer c2.Close()

	var rb, wb atomic.Int64
	wrapped := newCountingConn(c1, &rb, &wb)

	cw, ok := interface{}(wrapped).(closeWriter)
	if !ok {
		t.Fatal("countingConn should satisfy closeWriter")
	}
	if err := cw.CloseWrite(); err != nil {
		t.Errorf("CloseWrite on non-TCP underlying conn should be a no-op, got: %v", err)
	}
}
