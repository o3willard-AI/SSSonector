package nat

import (
	"sync"

	"gvisor.dev/gvisor/pkg/tcpip/header"
)

// FrameAssembler reassembles raw TLS-read chunks into complete IPv4
// frames. TLS record boundaries do not align with IP packet boundaries
// (QA: a coalesced or split chunk made the proto byte land mid-header,
// so per-Read delivery misparsed frames). Feed every chunk via
// Deliver; each fully-assembled frame is passed to fn.
type FrameAssembler struct {
	mu        sync.Mutex
	buf       []byte
	chunks    uint64
	chunkByt  uint64
	framesOut uint64
	frameByt  uint64
}

// maxBuf caps the reassembly buffer: a stream that never yields valid
// frames must not grow without bound.
const maxBuf = 256 * 1024

// Deliver appends a raw chunk and invokes fn for every complete IPv4
// frame now available.
func (a *FrameAssembler) Deliver(pkt []byte, fn func(frame []byte)) {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.buf = append(a.buf, pkt...)
	a.chunks++
	a.chunkByt += uint64(len(pkt))
	for {
		frame, ok := a.nextFrame()
		if !ok {
			break
		}
		a.framesOut++
		a.frameByt += uint64(len(frame))
		fn(frame)
	}
	if len(a.buf) > maxBuf {
		a.buf = append(a.buf[:0], a.buf[len(a.buf)-maxBuf:]...)
	}
}

// Reset discards any buffered partial frame: call when the underlying
// stream restarts (new tunnel connection), since buffered bytes from the
// old stream would otherwise corrupt the new one's framing.
func (a *FrameAssembler) Reset() {
	a.mu.Lock()
	a.buf = a.buf[:0]
	a.mu.Unlock()
}

// DebugStats returns chunk/frame counters for QA diagnostics.
func (a *FrameAssembler) DebugStats() (chunks, chunkByt, framesOut uint64) {
	a.mu.Lock()
	defer a.mu.Unlock()
	return a.chunks, a.chunkByt, a.framesOut
}

// nextFrame pops one complete IPv4 frame from the buffer, or returns
// ok=false if more bytes are needed. Unparsable prefixes are skipped
// byte-by-byte until a plausible IPv4 header is found.
func (a *FrameAssembler) nextFrame() ([]byte, bool) {
	for {
		if len(a.buf) < header.IPv4MinimumSize {
			return nil, false
		}
		if a.buf[0]>>4 != 4 {
			// Resync: drop one byte and scan on.
			a.buf = a.buf[1:]
			continue
		}
		ihl := int(a.buf[0]&0x0F) * 4
		if ihl < header.IPv4MinimumSize {
			a.buf = a.buf[1:]
			continue
		}
		total := int(a.buf[2])<<8 | int(a.buf[3])
		if total < ihl {
			a.buf = a.buf[1:] // bogus length: resync
			continue
		}
		if total > len(a.buf) {
			return nil, false // need more bytes
		}
		frame := make([]byte, total)
		copy(frame, a.buf[:total])
		a.buf = a.buf[total:]
		return frame, true
	}
}
