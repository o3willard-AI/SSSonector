package connection

import (
	"context"
	"fmt"
	"net"
	"sync"
	"time"

	"github.com/o3willard-AI/SSSonector/internal/breaker"
	"github.com/o3willard-AI/SSSonector/internal/ratelimit"
	"go.uber.org/zap"
)

// Dialer defines the interface for creating connections
type Dialer interface {
	DialContext(ctx context.Context, network, address string) (net.Conn, error)
}

// ConnectionInfo holds information about a connection
type ConnectionInfo struct {
	RemoteAddr    string
	State         ManagerState
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
	RateLimit      *ratelimit.Config
	CircuitBreaker *breaker.Config
}

// Manager handles connection management and tracking
type Manager struct {
	logger         *zap.Logger
	config         *Config
	mu             sync.RWMutex
	conns          map[net.Conn]*ConnectionInfo
	connCount      int
	state          ManagerState
	onConnect      func(net.Conn)
	onDisconnect   func(net.Conn, error)
	rateLimiter    *ratelimit.RateLimiter
	circuitBreaker *breaker.CircuitBreaker
	dialer         Dialer
}

// NewManager creates a new connection manager
func NewManager(logger *zap.Logger, cfg *Config) *Manager {
	m := &Manager{
		logger: logger,
		config: cfg,
		conns:  make(map[net.Conn]*ConnectionInfo),
		state:  ManagerStateDisconnected,
		dialer: &net.Dialer{},
	}

	if cfg.RateLimit != nil {
		m.rateLimiter = ratelimit.NewRateLimiter(cfg.RateLimit, logger)
	}

	if cfg.CircuitBreaker != nil {
		m.circuitBreaker = breaker.NewCircuitBreaker(cfg.CircuitBreaker, logger)
	}

	return m
}

// SetDialer sets a custom dialer for testing
func (m *Manager) SetDialer(dialer Dialer) {
	m.dialer = dialer
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
	// Check rate limit first
	if m.rateLimiter != nil {
		remoteAddr := conn.RemoteAddr().String()
		if !m.rateLimiter.Allow(remoteAddr) {
			return fmt.Errorf("rate limit exceeded for %s", remoteAddr)
		}
	}

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
		State:      ManagerStateConnected,
	}
	m.connCount++
	m.state = ManagerStateConnected

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
		info.State = ManagerStateDisconnected
		m.logger.Info("Connection removed",
			zap.String("remote_addr", conn.RemoteAddr().String()),
			zap.Int("total_connections", m.connCount),
		)

		if m.onDisconnect != nil {
			m.onDisconnect(conn, nil)
		}

		if m.connCount == 0 {
			m.state = ManagerStateDisconnected
		}

		// Remove rate limit for this connection
		if m.rateLimiter != nil {
			m.rateLimiter.Remove(conn.RemoteAddr().String())
		}
	}
}

// Connect attempts to establish a connection with retries
func (m *Manager) Connect(network, address string) (net.Conn, error) {
	if m.circuitBreaker != nil {
		var conn net.Conn
		var err error
		execErr := m.circuitBreaker.Execute(context.Background(), func() error {
			conn, err = m.doConnect(network, address)
			return err
		})
		if execErr != nil {
			return nil, execErr
		}
		return conn, nil
	}
	return m.doConnect(network, address)
}

// doConnect performs the actual connection attempt
func (m *Manager) doConnect(network, address string) (net.Conn, error) {
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

			conn, err := m.dialer.DialContext(ctx, network, address)
			if err == nil {
				resultCh <- conn
				return
			}

			m.logger.Debug("Connection attempt failed",
				zap.String("network", network),
				zap.String("address", address),
				zap.Int("attempt", attempt+1),
				zap.Error(err))

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

			// Update rate limiter if bytes exceed threshold
			if m.rateLimiter != nil && bytesSent+bytesReceived > 1024*1024 { // 1MB threshold
				// Adjust rate limit based on usage
				currentRate, currentBurst := m.rateLimiter.GetRate(remoteAddr)
				if bytesSent+bytesReceived > 10*1024*1024 { // 10MB threshold
					// Reduce rate for high usage
					m.rateLimiter.SetRate(remoteAddr, currentRate/2, currentBurst/2)
				} else {
					// Increase rate for moderate usage
					m.rateLimiter.SetRate(remoteAddr, currentRate*2, currentBurst*2)
				}
			}
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
func (m *Manager) GetState() ManagerState {
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
	m.state = ManagerStateDisconnected

	if m.rateLimiter != nil {
		m.rateLimiter.Stop()
	}

	m.logger.Info("All connections closed")
}

// GetCircuitBreakerStats returns circuit breaker statistics if enabled
func (m *Manager) GetCircuitBreakerStats() *breaker.Stats {
	if m.circuitBreaker != nil {
		stats := m.circuitBreaker.GetStats()
		return &stats
	}
	return nil
}

// ResetCircuitBreaker resets the circuit breaker if enabled
func (m *Manager) ResetCircuitBreaker() {
	if m.circuitBreaker != nil {
		m.circuitBreaker.Reset()
	}
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

// GetRateLimitMetrics returns rate limiting metrics if enabled
func (m *Manager) GetRateLimitMetrics() *ratelimit.Metrics {
	if m.rateLimiter != nil {
		metrics := m.rateLimiter.GetMetrics()
		return &metrics
	}
	return nil
}
