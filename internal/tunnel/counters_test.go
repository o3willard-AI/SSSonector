package tunnel

import (
	"errors"
	"io"
	"net"
	"os"
	"sync/atomic"
	"testing"
	"time"
)

func TestCountingConnCountsBytes(t *testing.T) {
	c1, c2 := net.Pipe()
	defer c1.Close()
	defer c2.Close()

	var readBytes, writeBytes atomic.Int64
	cc := newCountingConn(c1, 0, &readBytes, &writeBytes)

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
	wrapped := newCountingConn(c1, 0, &rb, &wb)

	cw, ok := interface{}(wrapped).(closeWriter)
	if !ok {
		t.Fatal("countingConn should satisfy closeWriter")
	}
	if err := cw.CloseWrite(); err != nil {
		t.Errorf("CloseWrite on non-TCP underlying conn should be a no-op, got: %v", err)
	}
}

// TestIdleDeadlineAbortsSilentRead proves the dead-peer mechanism at the
// connection level: with an idle window armed and no incoming bytes, a
// blocked Read fails with a deadline error inside the window instead of
// hanging forever.
func TestIdleDeadlineAbortsSilentRead(t *testing.T) {
	c1, c2 := net.Pipe()
	defer c1.Close()
	defer c2.Close()

	cc := newCountingConn(c1, 150*time.Millisecond, &atomic.Int64{}, &atomic.Int64{})

	done := make(chan error, 1)
	start := time.Now()
	go func() {
		_, err := cc.Read(make([]byte, 64))
		done <- err
	}()

	select {
	case err := <-done:
		if !errors.Is(err, os.ErrDeadlineExceeded) {
			t.Errorf("expected deadline exceeded, got %v", err)
		}
		if elapsed := time.Since(start); elapsed < 100*time.Millisecond || elapsed > 2*time.Second {
			t.Errorf("abort fired outside expected window: %v", elapsed)
		}
	case <-time.After(3 * time.Second):
		t.Fatal("idle deadline never fired")
	}
}

// TestIdleWindowExtendsOnActivity proves traffic keeps pushing the deadline:
// periodic small writes keep a reader alive far longer than one window.
func TestIdleWindowExtendsOnActivity(t *testing.T) {
	listener, clientEnd, err := servePipeLike()
	if err != nil {
		t.Fatalf("setup: %v", err)
	}
	defer listener.Close()

	cc := newCountingConn(clientEnd, 200*time.Millisecond, &atomic.Int64{}, &atomic.Int64{})

	readDone := make(chan error, 1)
	go func() {
		buf := make([]byte, 8)
		for i := 0; i < 5; i++ {
			if _, err := cc.Read(buf); err != nil {
				readDone <- err
				return
			}
		}
		readDone <- nil
	}()

	writeDone := make(chan struct{})
	go func() {
		defer close(writeDone)
		conn, err := listener.Accept()
		if err != nil {
			return
		}
		defer conn.Close()
		for i := 0; i < 5; i++ {
			time.Sleep(80 * time.Millisecond) // well inside the 200ms window
			if _, err := conn.Write([]byte("ping")); err != nil {
				return
			}
		}
	}()

	select {
	case err := <-readDone:
		if err != nil {
			t.Errorf("active connection should not hit idle deadline: %v", err)
		}
	case <-time.After(3 * time.Second):
		t.Fatal("reader did not finish")
	}
	<-writeDone
}

// servePipeLike exposes a real TCP loopback listener so the writing side can
// be an ordinary conn while the reading side is deadline-instrumented.
func servePipeLike() (net.Listener, net.Conn, error) {
	ln, err := net.Listen("tcp4", "127.0.0.1:0")
	if err != nil {
		return nil, nil, err
	}
	conn, err := net.Dial(ln.Addr().Network(), ln.Addr().String())
	if err != nil {
		ln.Close()
		return nil, nil, err
	}
	return ln, conn, nil
}
