package connection

import (
	"context"
	"fmt"
	"net"
	"sync"
	"sync/atomic"
	"time"

	"go.uber.org/zap"
)

// State represents the connection state
type State int32

const (
	StateDisconnected State = iota
	StateConnecting
	StateConnected
	StateReconnecting
)

// String returns the string representation of the state
func (s State) String() string {
	switch s {
	case StateDisconnected:
		return "Disconnected"
	case StateConnecting:
		return "Connecting"
	case StateConnected:
		return "Connected"
	case StateReconnecting:
		return "Reconnecting"
	default:
		return "Unknown"
	}
}

// Config holds connection manager configuration
type Config struct {
	RetryAttempts  int           // Maximum number of retry attempts (0 = infinite)
	RetryInterval  time.Duration // Time to wait between retries
	ConnectTimeout time.Duration // Connection timeout
	KeepAlive      bool          // Enable TCP keepalive
	KeepAliveIdle  time.Duration // Time before sending keepalive probes
	MaxConnections int           // Maximum number of concurrent connections (server mode)
}

// ConnectionInfo holds information about a connection
type ConnectionInfo struct {
	RemoteAddr    string
	ConnectedAt   time.Time
	State         State
	BytesSent     uint64
	BytesReceived uint64
	LastError     error
}

// Manager handles connection lifecycle and state
type Manager struct {
	logger *zap.Logger
	config *Config

	// Connection tracking
	connections sync.Map // map[string]*ConnectionInfo
	connCount   int32

	// Connection events
	onConnect    func(net.Conn)
	onDisconnect func(net.Conn, error)

	// State
	state int32 // atomic State

	// Control
	ctx    context.Context
	cancel context.CancelFunc
	wg     sync.WaitGroup
}

// NewManager creates a new connection manager
func NewManager(logger *zap.Logger, config *Config) *Manager {
	ctx, cancel := context.WithCancel(context.Background())
	return &Manager{
		logger: logger,
		config: config,
		ctx:    ctx,
		cancel: cancel,
	}
}

// SetCallbacks sets connection event callbacks
func (m *Manager) SetCallbacks(onConnect func(net.Conn), onDisconnect func(net.Conn, error)) {
	m.onConnect = onConnect
	m.onDisconnect = onDisconnect
}

// Connect establishes a connection with retry logic
func (m *Manager) Connect(network, address string) (net.Conn, error) {
	atomic.StoreInt32((*int32)(&m.state), int32(StateConnecting))
	defer atomic.StoreInt32((*int32)(&m.state), int32(StateDisconnected))

	var attempts int
	for {
		select {
		case <-m.ctx.Done():
			return nil, m.ctx.Err()
		default:
		}

		// Create connection with timeout
		ctx, cancel := context.WithTimeout(m.ctx, m.config.ConnectTimeout)
		conn, err := (&net.Dialer{
			KeepAlive: m.config.KeepAliveIdle,
		}).DialContext(ctx, network, address)
		cancel()

		if err == nil {
			atomic.StoreInt32((*int32)(&m.state), int32(StateConnected))
			m.trackConnection(conn)
			if m.onConnect != nil {
				m.onConnect(conn)
			}
			return conn, nil
		}

		attempts++
		if m.config.RetryAttempts > 0 && attempts >= m.config.RetryAttempts {
			return nil, fmt.Errorf("max retry attempts reached: %w", err)
		}

		m.logger.Warn("Connection failed, retrying...",
			zap.Error(err),
			zap.Int("attempt", attempts),
			zap.Duration("retry_interval", m.config.RetryInterval),
		)

		select {
		case <-m.ctx.Done():
			return nil, m.ctx.Err()
		case <-time.After(m.config.RetryInterval):
		}
	}
}

// Accept handles an incoming connection
func (m *Manager) Accept(conn net.Conn) error {
	// Check connection limit
	if m.config.MaxConnections > 0 {
		if count := atomic.LoadInt32(&m.connCount); count >= int32(m.config.MaxConnections) {
			return fmt.Errorf("max connections reached: %d", m.config.MaxConnections)
		}
	}

	// Configure keepalive if enabled
	if m.config.KeepAlive {
		if tc, ok := conn.(*net.TCPConn); ok {
			tc.SetKeepAlive(true)
			tc.SetKeepAlivePeriod(m.config.KeepAliveIdle)
		}
	}

	m.trackConnection(conn)
	if m.onConnect != nil {
		m.onConnect(conn)
	}

	return nil
}

// trackConnection tracks connection state
func (m *Manager) trackConnection(conn net.Conn) {
	atomic.AddInt32(&m.connCount, 1)
	m.connections.Store(conn.RemoteAddr().String(), &ConnectionInfo{
		RemoteAddr:  conn.RemoteAddr().String(),
		ConnectedAt: time.Now(),
		State:       StateConnected,
	})

	// Start monitoring goroutine
	m.wg.Add(1)
	go func() {
		defer m.wg.Done()
		defer atomic.AddInt32(&m.connCount, -1)
		defer m.connections.Delete(conn.RemoteAddr().String())

		<-m.ctx.Done()
		conn.Close()
		if m.onDisconnect != nil {
			m.onDisconnect(conn, m.ctx.Err())
		}
	}()
}

// GetConnections returns information about all active connections
func (m *Manager) GetConnections() []ConnectionInfo {
	var infos []ConnectionInfo
	m.connections.Range(func(key, value interface{}) bool {
		if info, ok := value.(*ConnectionInfo); ok {
			infos = append(infos, *info)
		}
		return true
	})
	return infos
}

// GetConnectionCount returns the current number of active connections
func (m *Manager) GetConnectionCount() int {
	return int(atomic.LoadInt32(&m.connCount))
}

// GetState returns the current connection state
func (m *Manager) GetState() State {
	return State(atomic.LoadInt32((*int32)(&m.state)))
}

// UpdateStats updates connection statistics
func (m *Manager) UpdateStats(remoteAddr string, bytesSent, bytesReceived uint64) {
	if value, ok := m.connections.Load(remoteAddr); ok {
		if info, ok := value.(*ConnectionInfo); ok {
			atomic.StoreUint64(&info.BytesSent, bytesSent)
			atomic.StoreUint64(&info.BytesReceived, bytesReceived)
		}
	}
}

// Stop stops the connection manager and closes all connections
func (m *Manager) Stop() {
	m.cancel()
	m.wg.Wait()
}
