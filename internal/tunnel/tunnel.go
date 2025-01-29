package tunnel

import (
	"crypto/tls"
	"fmt"
	"io"
	"net"
	"sync"
	"time"

	"github.com/o3willard-AI/SSSonector/internal/adapter"
	"github.com/o3willard-AI/SSSonector/internal/config"
	"github.com/o3willard-AI/SSSonector/internal/monitor"
	"github.com/o3willard-AI/SSSonector/internal/throttle"
	"go.uber.org/zap"
)

// Tunnel represents an SSL tunnel
type Tunnel struct {
	logger   *zap.Logger
	config   *config.TunnelConfig
	certMgr  *CertManager
	iface    adapter.Interface
	monitor  *monitor.Monitor
	throttle *throttle.Limiter

	listener net.Listener
	conns    sync.Map
	done     chan struct{}
}

// NewTunnel creates a new SSL tunnel
func NewTunnel(logger *zap.Logger, cfg *config.TunnelConfig, iface adapter.Interface, mon *monitor.Monitor) (*Tunnel, error) {
	certMgr := NewCertManager(logger, cfg)
	if err := certMgr.VerifyCertificates(); err != nil {
		return nil, fmt.Errorf("certificate verification failed: %w", err)
	}

	return &Tunnel{
		logger:   logger,
		config:   cfg,
		certMgr:  certMgr,
		iface:    iface,
		monitor:  mon,
		throttle: throttle.NewLimiter(cfg.BandwidthLimit),
		done:     make(chan struct{}),
	}, nil
}

// Start starts the tunnel in either server or client mode
func (t *Tunnel) Start() error {
	if t.config.ListenAddress != "" {
		return t.startServer()
	}
	return t.startClient()
}

// startServer starts the tunnel in server mode
func (t *Tunnel) startServer() error {
	tlsConfig, err := t.certMgr.GetServerTLSConfig()
	if err != nil {
		return fmt.Errorf("failed to get server TLS config: %w", err)
	}

	addr := fmt.Sprintf("%s:%d", t.config.ListenAddress, t.config.ListenPort)
	listener, err := tls.Listen("tcp", addr, tlsConfig)
	if err != nil {
		return fmt.Errorf("failed to start listener: %w", err)
	}
	t.listener = listener

	t.logger.Info("Server started",
		zap.String("address", addr),
	)

	go t.acceptConnections()
	return nil
}

// startClient starts the tunnel in client mode
func (t *Tunnel) startClient() error {
	tlsConfig, err := t.certMgr.GetClientTLSConfig()
	if err != nil {
		return fmt.Errorf("failed to get client TLS config: %w", err)
	}

	go t.maintainConnection(tlsConfig)
	return nil
}

// acceptConnections accepts incoming connections in server mode
func (t *Tunnel) acceptConnections() {
	for {
		conn, err := t.listener.Accept()
		if err != nil {
			select {
			case <-t.done:
				return
			default:
				t.logger.Error("Failed to accept connection",
					zap.Error(err),
				)
				continue
			}
		}

		tlsConn := conn.(*tls.Conn)
		if err := tlsConn.Handshake(); err != nil {
			t.logger.Error("TLS handshake failed",
				zap.Error(err),
			)
			conn.Close()
			continue
		}

		go t.handleConnection(conn)
	}
}

// maintainConnection maintains a persistent connection in client mode
func (t *Tunnel) maintainConnection(tlsConfig *tls.Config) {
	var retryCount int
	for {
		select {
		case <-t.done:
			return
		default:
			addr := fmt.Sprintf("%s:%d", t.config.ServerAddress, t.config.ServerPort)
			conn, err := tls.Dial("tcp", addr, tlsConfig)
			if err != nil {
				retryCount++
				t.monitor.LogConnectionEvent("connect_failed", addr, err)
				if t.config.RetryAttempts > 0 && retryCount >= t.config.RetryAttempts {
					t.logger.Error("Max retry attempts reached",
						zap.Int("attempts", retryCount),
					)
					return
				}
				time.Sleep(time.Duration(t.config.RetryInterval) * time.Second)
				continue
			}

			retryCount = 0
			t.monitor.LogConnectionEvent("connected", addr, nil)
			t.handleConnection(conn)

			select {
			case <-t.done:
				return
			default:
				time.Sleep(time.Second)
				continue
			}
		}
	}
}

// handleConnection handles a tunnel connection
func (t *Tunnel) handleConnection(conn net.Conn) {
	defer conn.Close()

	// Create throttled reader/writer
	reader := t.throttle.NewReader(conn)
	writer := t.throttle.NewWriter(conn)

	// Start bidirectional copy
	var wg sync.WaitGroup
	wg.Add(2)

	// Interface -> Connection
	go func() {
		defer wg.Done()
		if _, err := t.throttle.ThrottledCopy(writer, t.iface); err != nil {
			if err != io.EOF {
				t.logger.Error("Interface -> Connection copy failed",
					zap.Error(err),
				)
			}
		}
	}()

	// Connection -> Interface
	go func() {
		defer wg.Done()
		if _, err := t.throttle.ThrottledCopy(t.iface, reader); err != nil {
			if err != io.EOF {
				t.logger.Error("Connection -> Interface copy failed",
					zap.Error(err),
				)
			}
		}
	}()

	wg.Wait()
}

// Close closes the tunnel and all connections
func (t *Tunnel) Close() error {
	close(t.done)

	if t.listener != nil {
		t.listener.Close()
	}

	t.conns.Range(func(key, value interface{}) bool {
		if conn, ok := value.(net.Conn); ok {
			conn.Close()
		}
		return true
	})

	return nil
}
