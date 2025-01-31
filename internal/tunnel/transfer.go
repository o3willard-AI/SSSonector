package tunnel

import (
"context"
"io"
"net"
"sync"
"sync/atomic"
"time"

"github.com/o3willard-AI/SSSonector/internal/throttle"
"go.uber.org/zap"
)

// Statistics holds tunnel transfer statistics
type Statistics struct {
BytesReceived uint64
BytesSent     uint64
StartTime     time.Time
LastActivity  time.Time
RemoteAddr    string
RemoteFQDN    string
}

// Transfer handles data transfer between network interfaces
type Transfer struct {
logger       *zap.Logger
iface        io.ReadWriteCloser
conn         net.Conn
limiter      *throttle.Limiter
stats        Statistics
ctx          context.Context
cancel       context.CancelFunc
wg           sync.WaitGroup
mu           sync.RWMutex
uploadKbps   float64
downloadKbps float64
}

// NewTransfer creates a new transfer instance
func NewTransfer(logger *zap.Logger, iface io.ReadWriteCloser, conn net.Conn, uploadKbps, downloadKbps float64) *Transfer {
ctx, cancel := context.WithCancel(context.Background())
t := &Transfer{
logger:       logger,
iface:        iface,
conn:         conn,
ctx:          ctx,
cancel:       cancel,
uploadKbps:   uploadKbps,
downloadKbps: downloadKbps,
}

// Initialize statistics
t.stats.StartTime = time.Now()
t.stats.LastActivity = time.Now()
t.stats.RemoteAddr = conn.RemoteAddr().String()
if addr, ok := conn.RemoteAddr().(*net.TCPAddr); ok {
names, err := net.LookupAddr(addr.IP.String())
if err == nil && len(names) > 0 {
t.stats.RemoteFQDN = names[0]
}
}

// Initialize bandwidth limiter if throttling is enabled
if uploadKbps > 0 || downloadKbps > 0 {
t.limiter = throttle.NewLimiter(conn, conn, int64(uploadKbps), int64(downloadKbps))
}

return t
}

// Start starts the transfer process
func (t *Transfer) Start() {
t.wg.Add(2)
go t.tunnelToInterface()
go t.interfaceToTunnel()
}

// Stop stops the transfer process
func (t *Transfer) Stop() {
t.cancel()
t.wg.Wait()

t.mu.Lock()
defer t.mu.Unlock()

if t.conn != nil {
t.conn.Close()
t.conn = nil
}
}

// GetStatistics returns current transfer statistics
func (t *Transfer) GetStatistics() Statistics {
t.mu.RLock()
defer t.mu.RUnlock()
return t.stats
}

// tunnelToInterface transfers data from tunnel to interface
func (t *Transfer) tunnelToInterface() {
defer t.wg.Done()
buffer := make([]byte, 65536)

for {
select {
case <-t.ctx.Done():
return
default:
var n int
var err error

t.mu.RLock()
if t.limiter != nil {
n, err = t.limiter.Read(buffer)
} else if t.conn != nil {
n, err = t.conn.Read(buffer)
}
t.mu.RUnlock()

if err != nil {
if err != io.EOF && !isClosedError(err) {
t.logger.Error("Failed to read from tunnel",
zap.Error(err),
)
}
return
}

if n > 0 {
t.mu.RLock()
if t.iface != nil {
_, err = t.iface.Write(buffer[:n])
}
t.mu.RUnlock()

if err != nil {
if !isClosedError(err) {
t.logger.Error("Failed to write to interface",
zap.Error(err),
)
}
return
}

atomic.AddUint64(&t.stats.BytesReceived, uint64(n))
t.updateLastActivity()
}
}
}
}

// interfaceToTunnel transfers data from interface to tunnel
func (t *Transfer) interfaceToTunnel() {
defer t.wg.Done()
buffer := make([]byte, 65536)

for {
select {
case <-t.ctx.Done():
return
default:
t.mu.RLock()
n, err := t.iface.Read(buffer)
t.mu.RUnlock()

if err != nil {
if err != io.EOF && !isClosedError(err) {
t.logger.Error("Failed to read from interface",
zap.Error(err),
)
}
return
}

if n > 0 {
var err error
t.mu.RLock()
if t.limiter != nil {
_, err = t.limiter.Write(buffer[:n])
} else if t.conn != nil {
_, err = t.conn.Write(buffer[:n])
}
t.mu.RUnlock()

if err != nil {
if !isClosedError(err) {
t.logger.Error("Failed to write to tunnel",
zap.Error(err),
)
}
return
}

atomic.AddUint64(&t.stats.BytesSent, uint64(n))
t.updateLastActivity()
}
}
}
}

func (t *Transfer) updateLastActivity() {
t.mu.Lock()
t.stats.LastActivity = time.Now()
t.mu.Unlock()
}

func isClosedError(err error) bool {
if err == nil {
return false
}
return err == io.EOF || err == io.ErrClosedPipe ||
err.Error() == "use of closed network connection"
}
