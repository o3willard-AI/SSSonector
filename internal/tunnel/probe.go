package tunnel

import (
	"encoding/hex"
	"fmt"
	"net"
	"sync/atomic"

	"go.uber.org/zap"
)

// probeConn is a diagnostic wrapper that logs every read's payload size
// and the first bytes as hex, so we can ground the NAT architecture on
// the actual TLS-record framing the server receives. It is added
// LAST on the tunnel conn (innermost wrapping) so it observes exactly
// what the TLS layer hands up.
type probeConn struct {
	net.Conn
	tag    string
	logger *zap.Logger

	seq   atomic.Uint64
	bytes atomic.Uint64
	// fullPkt counts reads where a complete IPv4 packet (IHL*4+payload
	// per the total-length field) fits exactly in the read boundary.
	fullPkt atomic.Uint64
	// partialOrCoalesced counts reads where the boundary does NOT match
	// the IP total-length field.
	mismatch atomic.Uint64
	// multiple counts reads containing more than one IP packet.
	multiple atomic.Uint64
}

func newProbeConn(c net.Conn, tag string, logger *zap.Logger) *probeConn {
	return &probeConn{Conn: c, tag: tag, logger: logger}
}

// analyzeIP tries to walk consecutive IPv4 packets in buf and reports
// how many complete packets it contains and whether the boundary aligns.
func analyzeIP(buf []byte) (packets int, exactEnd bool, firstProto byte, ok bool) {
	if len(buf) < 20 || buf[0]>>4 != 4 {
		return 0, false, 0, false
	}
	off := 0
	n := 0
	for off+20 <= len(buf) {
		if buf[off]>>4 != 4 {
			return n, false, buf[0], true // second blob non-IPv4: coalesced garbage
		}
		total := int(buf[off+2])<<8 | int(buf[off+3])
		if total < 20 || off+total > len(buf) {
			return n, false, buf[0], true // partial packet at end
		}
		off += total
		n++
		if off == len(buf) {
			return n, true, buf[0], true
		}
	}
	return n, false, buf[0], true
}

func (p *probeConn) Read(b []byte) (int, error) {
	n, err := p.Conn.Read(b)
	if err != nil || n == 0 {
		return n, err
	}
	seq := p.seq.Add(1)
	p.bytes.Add(uint64(n))

	pkts, exact, proto, ok := analyzeIP(b[:n])
	if ok {
		if exact && pkts == 1 {
			p.fullPkt.Add(1)
		} else if pkts > 1 {
			p.multiple.Add(1)
		} else {
			p.mismatch.Add(1)
		}
	}

	// Log the first N reads in full, then only mismatches (to bound log
	// volume while still catching every boundary anomaly).
	hexLen := 48
	if n < hexLen {
		hexLen = n
	}
	head := hex.EncodeToString(b[:hexLen])
	if seq <= 8 || (ok && pkts == 1 && exact && seq <= 40) {
		p.logger.Info("PROBE tunnel read",
			zap.String("tag", p.tag),
			zap.Uint64("seq", seq),
			zap.Int("n", n),
			zap.Bool("ipv4", ok),
			zap.Int("pkts", pkts),
			zap.Bool("exact", exact),
			zap.Uint8("proto", proto),
			zap.String("head_hex", head),
		)
	} else if !ok || !exact || pkts != 1 {
		if seq%20 == 0 { // sample mismatches, not all
			p.logger.Warn("PROBE boundary mismatch",
				zap.String("tag", p.tag),
				zap.Uint64("seq", seq),
				zap.Int("n", n),
				zap.Int("pkts", pkts),
				zap.Bool("exact", exact),
				zap.Uint8("proto", proto),
				zap.String("head_hex", head),
			)
		}
	}
	return n, err
}

// Snapshot returns the accumulated counters.
func (p *probeConn) Snapshot() string {
	return fmt.Sprintf("reads=%d bytes=%d exact_single=%d multi=%d mismatch=%d",
		p.seq.Load(), p.bytes.Load(), p.fullPkt.Load(), p.multiple.Load(), p.mismatch.Load())
}
