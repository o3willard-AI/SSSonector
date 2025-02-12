package balancer

import (
	"context"
	"errors"
	"net"
	"sync"
	"sync/atomic"
	"time"

	"go.uber.org/zap"
)

// Strategy represents a load balancing strategy
type Strategy int

const (
	// RoundRobin distributes connections in a circular order
	RoundRobin Strategy = iota
	// LeastConnections assigns to the endpoint with fewest active connections
	LeastConnections
	// WeightedRoundRobin distributes based on endpoint weights
	WeightedRoundRobin
)

// Dialer defines the interface for creating connections
type Dialer interface {
	DialContext(ctx context.Context, network, address string) (net.Conn, error)
}

// Endpoint represents a backend endpoint
type Endpoint struct {
	Address            string
	Weight             int
	ActiveConns        int64
	TotalConns         int64
	LastError          error
	LastErrorTime      time.Time
	SuccessStreak      int64
	FailureStreak      int64
	HealthyThreshold   int64
	UnhealthyThreshold int64
	mu                 sync.RWMutex
}

// Config holds load balancer configuration
type Config struct {
	Strategy           Strategy
	HealthCheck        time.Duration
	RetryTimeout       time.Duration
	MaxRetries         int
	HealthyThreshold   int64
	UnhealthyThreshold int64
}

// Balancer implements connection load balancing
type Balancer struct {
	logger    *zap.Logger
	config    *Config
	endpoints []*Endpoint
	mu        sync.RWMutex
	current   uint64
	dialer    Dialer
	ctx       context.Context
	cancel    context.CancelFunc
}

// NewBalancer creates a new load balancer
func NewBalancer(logger *zap.Logger, cfg *Config) *Balancer {
	ctx, cancel := context.WithCancel(context.Background())
	b := &Balancer{
		logger: logger,
		config: cfg,
		ctx:    ctx,
		cancel: cancel,
		dialer: &net.Dialer{},
	}

	if cfg.HealthCheck > 0 {
		go b.healthCheck()
	}

	return b
}

// SetDialer sets a custom dialer
func (b *Balancer) SetDialer(dialer Dialer) {
	b.dialer = dialer
}

// AddEndpoint adds a new endpoint
func (b *Balancer) AddEndpoint(addr string, weight int) {
	b.mu.Lock()
	defer b.mu.Unlock()

	endpoint := &Endpoint{
		Address:            addr,
		Weight:             weight,
		HealthyThreshold:   b.config.HealthyThreshold,
		UnhealthyThreshold: b.config.UnhealthyThreshold,
	}
	b.endpoints = append(b.endpoints, endpoint)

	b.logger.Info("Added endpoint",
		zap.String("address", addr),
		zap.Int("weight", weight))
}

// RemoveEndpoint removes an endpoint
func (b *Balancer) RemoveEndpoint(addr string) {
	b.mu.Lock()
	defer b.mu.Unlock()

	for i, ep := range b.endpoints {
		if ep.Address == addr {
			b.endpoints = append(b.endpoints[:i], b.endpoints[i+1:]...)
			b.logger.Info("Removed endpoint", zap.String("address", addr))
			return
		}
	}
}

// Next returns the next endpoint based on the balancing strategy
func (b *Balancer) Next() (*Endpoint, error) {
	b.mu.RLock()
	defer b.mu.RUnlock()

	if len(b.endpoints) == 0 {
		return nil, errors.New("no endpoints available")
	}

	switch b.config.Strategy {
	case RoundRobin:
		return b.nextRoundRobin()
	case LeastConnections:
		return b.nextLeastConnections()
	case WeightedRoundRobin:
		return b.nextWeightedRoundRobin()
	default:
		return b.nextRoundRobin()
	}
}

func (b *Balancer) nextRoundRobin() (*Endpoint, error) {
	current := atomic.AddUint64(&b.current, 1)
	idx := int(current % uint64(len(b.endpoints)))
	return b.endpoints[idx], nil
}

func (b *Balancer) nextLeastConnections() (*Endpoint, error) {
	var minConns int64 = 1<<63 - 1
	var selected *Endpoint

	for _, ep := range b.endpoints {
		conns := atomic.LoadInt64(&ep.ActiveConns)
		if conns < minConns {
			minConns = conns
			selected = ep
		}
	}

	return selected, nil
}

func (b *Balancer) nextWeightedRoundRobin() (*Endpoint, error) {
	var totalWeight int
	for _, ep := range b.endpoints {
		totalWeight += ep.Weight
	}

	current := atomic.AddUint64(&b.current, 1)
	targetWeight := int(current % uint64(totalWeight))

	var accumWeight int
	for _, ep := range b.endpoints {
		accumWeight += ep.Weight
		if accumWeight > targetWeight {
			return ep, nil
		}
	}

	return b.endpoints[0], nil
}

// Connect establishes a connection using the next endpoint
func (b *Balancer) Connect(ctx context.Context) (net.Conn, error) {
	var lastErr error

	for i := 0; i <= b.config.MaxRetries; i++ {
		ep, err := b.Next()
		if err != nil {
			return nil, err
		}

		conn, err := b.connectToEndpoint(ctx, ep)
		if err == nil {
			atomic.AddInt64(&ep.ActiveConns, 1)
			atomic.AddInt64(&ep.TotalConns, 1)
			atomic.AddInt64(&ep.SuccessStreak, 1)
			atomic.StoreInt64(&ep.FailureStreak, 0)
			return conn, nil
		}

		lastErr = err
		atomic.AddInt64(&ep.FailureStreak, 1)
		atomic.StoreInt64(&ep.SuccessStreak, 0)
		ep.mu.Lock()
		ep.LastError = err
		ep.LastErrorTime = time.Now()
		ep.mu.Unlock()

		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-time.After(b.config.RetryTimeout):
		}
	}

	return nil, lastErr
}

func (b *Balancer) connectToEndpoint(ctx context.Context, ep *Endpoint) (net.Conn, error) {
	return b.dialer.DialContext(ctx, "tcp", ep.Address)
}

// ReleaseConn decrements the active connection count for an endpoint
func (b *Balancer) ReleaseConn(addr string) {
	b.mu.RLock()
	defer b.mu.RUnlock()

	for _, ep := range b.endpoints {
		if ep.Address == addr {
			atomic.AddInt64(&ep.ActiveConns, -1)
			return
		}
	}
}

// healthCheck performs periodic health checks on endpoints
func (b *Balancer) healthCheck() {
	ticker := time.NewTicker(b.config.HealthCheck)
	defer ticker.Stop()

	for {
		select {
		case <-b.ctx.Done():
			return
		case <-ticker.C:
			b.checkEndpoints()
		}
	}
}

func (b *Balancer) checkEndpoints() {
	b.mu.RLock()
	endpoints := make([]*Endpoint, len(b.endpoints))
	copy(endpoints, b.endpoints)
	b.mu.RUnlock()

	for _, ep := range endpoints {
		ctx, cancel := context.WithTimeout(context.Background(), b.config.HealthCheck/2)
		conn, err := b.connectToEndpoint(ctx, ep)
		cancel()

		if err == nil {
			conn.Close()
			atomic.AddInt64(&ep.SuccessStreak, 1)
			atomic.StoreInt64(&ep.FailureStreak, 0)
		} else {
			atomic.AddInt64(&ep.FailureStreak, 1)
			atomic.StoreInt64(&ep.SuccessStreak, 0)
			ep.mu.Lock()
			ep.LastError = err
			ep.LastErrorTime = time.Now()
			ep.mu.Unlock()
		}
	}
}

// Stop stops the load balancer
func (b *Balancer) Stop() {
	b.cancel()
}

// GetStats returns statistics for all endpoints
type EndpointStats struct {
	Address       string
	ActiveConns   int64
	TotalConns    int64
	SuccessStreak int64
	FailureStreak int64
	LastError     error
	LastErrorTime time.Time
}

func (b *Balancer) GetStats() []EndpointStats {
	b.mu.RLock()
	defer b.mu.RUnlock()

	stats := make([]EndpointStats, len(b.endpoints))
	for i, ep := range b.endpoints {
		ep.mu.RLock()
		stats[i] = EndpointStats{
			Address:       ep.Address,
			ActiveConns:   atomic.LoadInt64(&ep.ActiveConns),
			TotalConns:    atomic.LoadInt64(&ep.TotalConns),
			SuccessStreak: atomic.LoadInt64(&ep.SuccessStreak),
			FailureStreak: atomic.LoadInt64(&ep.FailureStreak),
			LastError:     ep.LastError,
			LastErrorTime: ep.LastErrorTime,
		}
		ep.mu.RUnlock()
	}

	return stats
}
