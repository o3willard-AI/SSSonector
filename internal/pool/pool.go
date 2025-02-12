package pool

import (
	"context"
	"fmt"
	"net"
	"sync"
)

// Pool represents a pool of connections
type Pool struct {
	config Config
	conns  chan net.Conn
	mu     sync.RWMutex
}

// NewPool creates a new connection pool
func NewPool(config Config) (*Pool, error) {
	if config.Factory == nil {
		return nil, fmt.Errorf("factory is required")
	}

	if config.MaxSize < config.MinSize {
		return nil, fmt.Errorf("max size must be greater than or equal to min size")
	}

	if config.InitialSize < 0 || config.InitialSize > config.MaxSize {
		return nil, fmt.Errorf("initial size must be between 0 and max size")
	}

	p := &Pool{
		config: config,
		conns:  make(chan net.Conn, config.MaxSize),
	}

	// Initialize the pool with the initial connections
	ctx, cancel := context.WithTimeout(context.Background(), config.ConnectTimeout.Duration)
	defer cancel()

	for i := 0; i < config.InitialSize; i++ {
		conn, err := config.Factory(ctx)
		if err != nil {
			return nil, fmt.Errorf("failed to create initial connection: %w", err)
		}
		p.conns <- conn
	}

	return p, nil
}

// Get gets a connection from the pool
func (p *Pool) Get(ctx context.Context) (net.Conn, error) {
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	case conn := <-p.conns:
		if p.config.HealthCheck != nil {
			if err := p.config.HealthCheck(conn); err != nil {
				conn.Close()
				return p.config.Factory(ctx)
			}
		}
		return conn, nil
	default:
		// Pool is empty, create a new connection if we haven't reached max size
		p.mu.RLock()
		currentSize := len(p.conns)
		p.mu.RUnlock()

		if currentSize < p.config.MaxSize {
			return p.config.Factory(ctx)
		}

		// Wait for a connection to become available or context cancellation
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case conn := <-p.conns:
			if p.config.HealthCheck != nil {
				if err := p.config.HealthCheck(conn); err != nil {
					conn.Close()
					select {
					case <-ctx.Done():
						return nil, ctx.Err()
					default:
						return p.config.Factory(ctx)
					}
				}
			}
			return conn, nil
		}
	}
}

// Put returns a connection to the pool
func (p *Pool) Put(conn net.Conn) error {
	select {
	case p.conns <- conn:
		return nil
	default:
		// Pool is full, close the connection
		if p.config.OnClose != nil {
			p.config.OnClose(conn)
		}
		return conn.Close()
	}
}

// Close closes all connections in the pool
func (p *Pool) Close() error {
	p.mu.Lock()
	defer p.mu.Unlock()

	close(p.conns)
	var lastErr error
	for conn := range p.conns {
		if p.config.OnClose != nil {
			p.config.OnClose(conn)
		}
		if err := conn.Close(); err != nil {
			lastErr = err
		}
	}
	return lastErr
}
