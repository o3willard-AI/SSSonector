package tunnel

import (
	"net"

	"go.uber.org/zap"
)

// reverseProbeConn is a temporary QA diagnostic wrapper on the Transfer src
// (tunnel side): logs every Read/Write passing through the reverse io.Copy
// path and its outcome, so a silent stall between TUN read and tunnel write
// becomes visible. Remove after the forward/reverse NAT QA cycle.
type reverseProbeConn struct {
	net.Conn
	tag    string
	logger *zap.Logger
}

func (r *reverseProbeConn) Write(p []byte) (int, error) {
	n, err := r.Conn.Write(p)
	r.logger.Info("PROBE reverse write",
		zap.String("tag", r.tag),
		zap.Int("n", n),
		zap.Error(err),
		zap.String("head_hex", headHex(p, n)),
	)
	return n, err
}

func (r *reverseProbeConn) Read(p []byte) (int, error) {
	n, err := r.Conn.Read(p)
	r.logger.Info("PROBE reverse read",
		zap.String("tag", r.tag),
		zap.Int("n", n),
		zap.Error(err),
		zap.String("head_hex", headHex(p, n)),
	)
	return n, err
}

func headHex(b []byte, n int) string {
	hexLen := n
	if hexLen > 16 {
		hexLen = 16
	}
	const hexDigits = "0123456789abcdef"
	out := make([]byte, 0, hexLen*2)
	for i := 0; i < hexLen; i++ {
		out = append(out, hexDigits[b[i]>>4], hexDigits[b[i]&0x0F])
	}
	return string(out)
}
