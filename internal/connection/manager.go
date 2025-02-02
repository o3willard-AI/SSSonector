package connection

import (
	"fmt"
	"net"
	"sync"

	"go.uber.org/zap"
)

// Config holds connection manager configuration
type Config struct {
	MaxConnections int
}

// Manager handles connection management and tracking
type Manager struct {
	logger    *zap.Logger
	config    *Config
	mu        sync.RWMutex
	conns     map[net.Conn]struct{}
	connCount int
}

// NewManager creates a new connection manager
func NewManager(logger *zap.Logger, cfg *Config) *Manager {
	return &Manager{
		logger: logger,
		config: cfg,
		conns:  make(map[net.Conn]struct{}),
	}
}

// Add registers a new connection
func (m *Manager) Add(conn net.Conn) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.connCount >= m.config.MaxConnections {
		return fmt.Errorf("maximum connections reached (%d)", m.config.MaxConnections)
	}

	m.conns[conn] = struct{}{}
	m.connCount++
	m.logger.Info("Connection added",
		zap.String("remote_addr", conn.RemoteAddr().String()),
		zap.Int("total_connections", m.connCount),
	)

	return nil
}

// Remove unregisters a connection
func (m *Manager) Remove(conn net.Conn) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if _, exists := m.conns[conn]; exists {
		delete(m.conns, conn)
		m.connCount--
		m.logger.Info("Connection removed",
			zap.String("remote_addr", conn.RemoteAddr().String()),
			zap.Int("total_connections", m.connCount),
		)
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

// CloseAll closes all managed connections
func (m *Manager) CloseAll() {
	m.mu.Lock()
	defer m.mu.Unlock()

	for conn := range m.conns {
		conn.Close()
		delete(m.conns, conn)
	}
	m.connCount = 0
	m.logger.Info("All connections closed")
}
