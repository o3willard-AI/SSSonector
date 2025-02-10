package pool

import (
	"context"
	"errors"
	"net"
	"sync"
	"time"

	"go.uber.org/zap"
)

const (
	defaultIdleTimeout   = 5 * time.Minute
	defaultMaxIdle       = 100
	defaultMaxActive     = 500
	defaultRetryInterval = 5 * time.Second
	defaultMaxRetries    = 3
)

// Config holds pool configuration
type Config struct {
	IdleTimeout   time.Duration
	MaxIdle       int
	MaxActive     int
	RetryInterval time.Duration
	MaxRetries    int
}

// DefaultConfig returns default pool configuration
func DefaultConfig() *Config {
	return &Config{
		IdleTimeout:   defaultIdleTimeout,
		MaxIdle:       defaultMaxIdle,
		MaxActive:     defaultMaxActive,
		RetryInterval: defaultRetryInterval,
		MaxRetries:    defaultMaxRetries,
	}
}

// Factory creates new connections
type Factory func(ctx context.Context) (net.Conn, error)

// Pool manages a pool of connections
type Pool struct {
	factory Factory
	config  *Config
	logger  *zap.Logger

	mu          sync.Mutex
	closed      bool
	active      int
	idle        []*poolConn
	activeConns map[*poolConn]struct{}

	// Health check
	healthCheck chan struct{}
}

// poolConn wraps a connection with pool metadata
type poolConn struct {
	net.Conn
	pool      *Pool
	timeUsed  time.Time
	timeAdded time.Time
	failCount int
}

// NewPool creates a new connection pool
func NewPool(factory Factory, cfg *Config, logger *zap.Logger) *Pool {
	if cfg == nil {
		cfg = DefaultConfig()
	}

	p := &Pool{
		factory:     factory,
		config:      cfg,
		logger:      logger,
		activeConns: make(map[*poolConn]struct{}),
		healthCheck: make(chan struct{}, 1),
	}

	// Start health checker
	go p.healthChecker()

	return p
}

// Get gets a connection from the pool
func (p *Pool) Get(ctx context.Context) (net.Conn, error) {
	p.mu.Lock()
	if p.closed {
		p.mu.Unlock()
		return nil, ErrPoolClosed
	}

	// Check for idle connections
	for i := len(p.idle) - 1; i >= 0; i-- {
		conn := p.idle[i]
		p.idle = p.idle[:i]

		// Remove stale connection
		if time.Since(conn.timeUsed) > p.config.IdleTimeout {
			conn.Conn.Close()
			continue
		}

		// Verify connection is still alive
		if err := p.testConnection(conn.Conn); err != nil {
			conn.Conn.Close()
			continue
		}

		p.active++
		p.activeConns[conn] = struct{}{}
		p.mu.Unlock()
		return conn, nil
	}

	// Check if we can create new connection
	if p.active >= p.config.MaxActive {
		p.mu.Unlock()
		return nil, ErrPoolExhausted
	}

	// Create new connection
	p.active++
	p.mu.Unlock()

	var conn net.Conn
	var err error
	for i := 0; i < p.config.MaxRetries; i++ {
		conn, err = p.factory(ctx)
		if err == nil {
			break
		}
		time.Sleep(p.config.RetryInterval)
	}
	if err != nil {
		p.mu.Lock()
		p.active--
		p.mu.Unlock()
		return nil, err
	}

	pc := &poolConn{
		Conn:      conn,
		pool:      p,
		timeUsed:  time.Now(),
		timeAdded: time.Now(),
	}

	p.mu.Lock()
	p.activeConns[pc] = struct{}{}
	p.mu.Unlock()

	return pc, nil
}

// Put returns a connection to the pool
func (p *Pool) Put(conn net.Conn) error {
	pc, ok := conn.(*poolConn)
	if !ok {
		return errors.New("connection was not created by this pool")
	}

	p.mu.Lock()
	defer p.mu.Unlock()

	if p.closed {
		conn.Close()
		return ErrPoolClosed
	}

	delete(p.activeConns, pc)
	p.active--

	// Don't cache connections that have failed too many times
	if pc.failCount > p.config.MaxRetries {
		conn.Close()
		return nil
	}

	// Only cache if we have room
	if len(p.idle) < p.config.MaxIdle {
		pc.timeUsed = time.Now()
		p.idle = append(p.idle, pc)
	} else {
		conn.Close()
	}

	return nil
}

// Close closes the pool and all its connections
func (p *Pool) Close() error {
	p.mu.Lock()
	if p.closed {
		p.mu.Unlock()
		return nil
	}
	p.closed = true

	// Close idle connections
	for _, conn := range p.idle {
		conn.Conn.Close()
	}
	p.idle = nil

	// Close active connections
	for conn := range p.activeConns {
		conn.Conn.Close()
	}
	p.activeConns = nil

	close(p.healthCheck)
	p.mu.Unlock()
	return nil
}

// testConnection tests if a connection is still alive
func (p *Pool) testConnection(conn net.Conn) error {
	deadline := time.Now().Add(time.Second)
	if err := conn.SetDeadline(deadline); err != nil {
		return err
	}

	// Try to write 1 byte
	if _, err := conn.Write([]byte{0}); err != nil {
		return err
	}

	return conn.SetDeadline(time.Time{})
}

// healthChecker periodically checks connection health
func (p *Pool) healthChecker() {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			p.mu.Lock()
			if p.closed {
				p.mu.Unlock()
				return
			}

			// Check idle connections
			var remaining []*poolConn
			for _, conn := range p.idle {
				if time.Since(conn.timeUsed) > p.config.IdleTimeout {
					conn.Conn.Close()
					continue
				}
				if err := p.testConnection(conn.Conn); err != nil {
					conn.Conn.Close()
					continue
				}
				remaining = append(remaining, conn)
			}
			p.idle = remaining

			// Check active connections
			for conn := range p.activeConns {
				if err := p.testConnection(conn.Conn); err != nil {
					conn.failCount++
					if conn.failCount > p.config.MaxRetries {
						delete(p.activeConns, conn)
						p.active--
						conn.Conn.Close()
					}
				}
			}
			p.mu.Unlock()

		case <-p.healthCheck:
			return
		}
	}
}

// Stats returns pool statistics
type Stats struct {
	ActiveCount int
	IdleCount   int
	WaitCount   int64
	WaitTime    time.Duration
}

// Stats returns current pool statistics
func (p *Pool) Stats() Stats {
	p.mu.Lock()
	stats := Stats{
		ActiveCount: p.active,
		IdleCount:   len(p.idle),
	}
	p.mu.Unlock()
	return stats
}
