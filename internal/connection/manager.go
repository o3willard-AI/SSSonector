package connection

import (
	"context"
	"fmt"
	"net"
	"sync"
	"time"

	"go.uber.org/zap"
)

// ConnectionState represents the state of a connection
type ConnectionState string

const (
	StateConnected    ConnectionState = "Connected"
	StateDisconnected ConnectionState = "Disconnected"
)

// ConnectionInfo holds information about a connection
type ConnectionInfo struct {
	RemoteAddr    string
	State         ConnectionState
	BytesSent     int64
	BytesReceived int64
}

// Config holds connection manager configuration
type Config struct {
	MaxConnections int
	KeepAlive      bool
	KeepAliveIdle  time.Duration
	RetryAttempts  int
	RetryInterval  time.Duration
	ConnectTimeout time.Duration
}

// Manager handles connection management and tracking
type Manager struct {
	logger       *zap.Logger
	config       *Config
	mu           sync.RWMutex
	conns        map[net.Conn]*ConnectionInfo
	connCount    int
	state        ConnectionState
	onConnect    func(net.Conn)
	onDisconnect func(net.Conn, error)
}

// NewManager creates a new connection manager
func NewManager(logger *zap.Logger, cfg *Config) *Manager {
	return &Manager{
		logger: logger,
		config: cfg,
		conns:  make(map[net.Conn]*ConnectionInfo),
		state:  StateDisconnected,
	}
}

// SetCallbacks sets the connection callbacks
func (m *Manager) SetCallbacks(onConnect func(net.Conn), onDisconnect func(net.Conn, error)) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.onConnect = onConnect
	m.onDisconnect = onDisconnect
}

// Accept registers a new connection
func (m *Manager) Accept(conn net.Conn) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.connCount >= m.config.MaxConnections {
		return fmt.Errorf("maximum connections reached (%d)", m.config.MaxConnections)
	}

	if m.config.KeepAlive {
		if tcpConn, ok := conn.(*net.TCPConn); ok {
			tcpConn.SetKeepAlive(true)
			tcpConn.SetKeepAlivePeriod(m.config.KeepAliveIdle)
		}
	}

	m.conns[conn] = &ConnectionInfo{
		RemoteAddr: conn.RemoteAddr().String(),
		State:      StateConnected,
	}
	m.connCount++
	m.state = StateConnected

	m.logger.Info("Connection added",
		zap.String("remote_addr", conn.RemoteAddr().String()),
		zap.Int("total_connections", m.connCount),
	)

	if m.onConnect != nil {
		m.onConnect(conn)
	}

	return nil
}

// Remove unregisters a connection
func (m *Manager) Remove(conn net.Conn) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if info, exists := m.conns[conn]; exists {
		delete(m.conns, conn)
		m.connCount--
		info.State = StateDisconnected
		m.logger.Info("Connection removed",
			zap.String("remote_addr", conn.RemoteAddr().String()),
			zap.Int("total_connections", m.connCount),
		)

		if m.onDisconnect != nil {
			m.onDisconnect(conn, nil)
		}

		if m.connCount == 0 {
			m.state = StateDisconnected
		}
	}
}

// Connect attempts to establish a connection with retries
func (m *Manager) Connect(network, address string) (net.Conn, error) {
	ctx, cancel := context.WithTimeout(context.Background(), m.config.ConnectTimeout)
	defer cancel()

	resultCh := make(chan net.Conn, 1)
	errCh := make(chan error, 1)

	go func() {
		for attempt := 0; attempt <= m.config.RetryAttempts; attempt++ {
			if attempt > 0 {
				select {
				case <-ctx.Done():
					errCh <- ctx.Err()
					return
				case <-time.After(m.config.RetryInterval):
				}
			}

			dialer := net.Dialer{Timeout: m.config.ConnectTimeout / time.Duration(m.config.RetryAttempts+1)}
			conn, err := dialer.DialContext(ctx, network, address)
			if err == nil {
				resultCh <- conn
				return
			}

			m.logger.Debug("Connection attempt failed",
				zap.String("network", network),
				zap.String("address", address),
				zap.Int("attempt", attempt+1),
				zap.Error(err),
			)

			if attempt == m.config.RetryAttempts {
				errCh <- err
			}
		}
	}()

	select {
	case conn := <-resultCh:
		return conn, nil
	case err := <-errCh:
		return nil, err
	case <-ctx.Done():
		return nil, ctx.Err()
	}
}

// UpdateStats updates connection statistics
func (m *Manager) UpdateStats(remoteAddr string, bytesSent, bytesReceived int64) {
	m.mu.Lock()
	defer m.mu.Unlock()

	for _, info := range m.conns {
		if info.RemoteAddr == remoteAddr {
			info.BytesSent = bytesSent
			info.BytesReceived = bytesReceived
			break
		}
	}
}

// GetConnections returns information about all connections
func (m *Manager) GetConnections() []ConnectionInfo {
	m.mu.RLock()
	defer m.mu.RUnlock()

	infos := make([]ConnectionInfo, 0, len(m.conns))
	for _, info := range m.conns {
		infos = append(infos, *info)
	}
	return infos
}

// GetState returns the current connection state
func (m *Manager) GetState() ConnectionState {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.state
}

// Stop closes all managed connections
func (m *Manager) Stop() {
	m.mu.Lock()
	defer m.mu.Unlock()

	for conn := range m.conns {
		if m.onDisconnect != nil {
			m.onDisconnect(conn, nil)
		}
		conn.Close()
		delete(m.conns, conn)
	}
	m.connCount = 0
	m.state = StateDisconnected
	m.logger.Info("All connections closed")
}

// CanAcceptMore checks if more connections can be accepted
func (m *Manager) CanAcceptMore() bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.connCount < m.config.MaxConnections
}

// GetConnectionCount returns the current number of connections
func (m *Manager) GetConnectionCount() int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.connCount
}

// Add is an alias for Accept for backward compatibility
func (m *Manager) Add(conn net.Conn) error {
	return m.Accept(conn)
}

// CloseAll is an alias for Stop for backward compatibility
func (m *Manager) CloseAll() {
	m.Stop()
}
