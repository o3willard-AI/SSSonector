package tunnel

import (
"context"
"fmt"
"net"
"sync"
"time"

"github.com/o3willard-AI/SSSonector/internal/adapter"
"github.com/o3willard-AI/SSSonector/internal/cert"
"github.com/o3willard-AI/SSSonector/internal/config"
"go.uber.org/zap"
)

// Tunnel represents an SSL tunnel
type Tunnel struct {
logger     *zap.Logger
config     *config.TunnelConfig
iface      adapter.Interface
listener   net.Listener
transfer   *Transfer
tlsManager *TLSManager
ctx        context.Context
cancel     context.CancelFunc
wg         sync.WaitGroup
mu         sync.Mutex
}

// NewTunnel creates a new tunnel instance
func NewTunnel(logger *zap.Logger, cfg *config.TunnelConfig, iface adapter.Interface) *Tunnel {
ctx, cancel := context.WithCancel(context.Background())
return &Tunnel{
logger:   logger,
config:   cfg,
iface:    iface,
ctx:      ctx,
cancel:   cancel,
}
}

// Start starts the tunnel in either server or client mode
func (t *Tunnel) Start(mode string) error {
t.mu.Lock()
defer t.mu.Unlock()

// Initialize certificate manager
certManager, err := cert.NewManager(t.config.CertFile, t.config.KeyFile)
if err != nil {
return fmt.Errorf("failed to initialize certificate manager: %w", err)
}

if t.config.CAFile != "" {
if err := certManager.LoadCACertificate(t.config.CAFile); err != nil {
return fmt.Errorf("failed to load CA certificate: %w", err)
}
}

t.tlsManager = NewTLSManager(t.logger, t.config)
t.tlsManager.SetCertManager(certManager)

switch mode {
case "server":
return t.startServer()
case "client":
return t.startClient()
default:
return ErrInvalidMode
}
}

// startServer starts the tunnel in server mode
func (t *Tunnel) startServer() error {
if t.iface == nil {
return ErrInterfaceNotInitialized
}

var err error
addr := fmt.Sprintf("%s:%d", t.config.ListenAddress, t.config.ListenPort)

t.listener, err = net.Listen("tcp", addr)
if err != nil {
return fmt.Errorf("failed to create listener: %w", err)
}

t.listener, err = t.tlsManager.WrapListener(t.listener)
if err != nil {
t.listener.Close()
return fmt.Errorf("failed to create TLS listener: %w", err)
}

t.logger.Info("Server listening",
zap.String("address", addr),
zap.Int("max_clients", t.config.MaxClients),
)

t.wg.Add(1)
go func() {
defer t.wg.Done()
t.acceptConnections()
}()

return nil
}

func (t *Tunnel) acceptConnections() {
for {
select {
case <-t.ctx.Done():
return
default:
conn, err := t.listener.Accept()
if err != nil {
if t.ctx.Err() != nil {
return
}
t.logger.Error("Failed to accept connection", zap.Error(err))
continue
}

t.logger.Info("Client connected",
zap.String("remote_addr", conn.RemoteAddr().String()),
)

t.mu.Lock()
if t.transfer != nil {
t.transfer.Stop()
}
t.transfer = NewTransfer(t.logger, t.iface, conn, float64(t.config.UploadKbps), float64(t.config.DownloadKbps))
t.mu.Unlock()

t.wg.Add(1)
go func() {
defer t.wg.Done()
t.transfer.Start()
}()
}
}
}

// startClient starts the tunnel in client mode
func (t *Tunnel) startClient() error {
if t.iface == nil {
return ErrInterfaceNotInitialized
}

addr := fmt.Sprintf("%s:%d", t.config.ServerAddress, t.config.ServerPort)

// Implement retry logic
var conn net.Conn
var err error
for attempt := 0; attempt < t.config.RetryAttempts; attempt++ {
conn, err = net.Dial("tcp", addr)
if err == nil {
break
}
if attempt < t.config.RetryAttempts-1 {
t.logger.Info("Connection attempt failed, retrying...",
zap.String("address", addr),
zap.Int("attempt", attempt+1),
zap.Error(err),
)
time.Sleep(time.Duration(t.config.RetryInterval) * time.Second)
}
}
if err != nil {
return fmt.Errorf("failed to connect to server after %d attempts: %w", t.config.RetryAttempts, err)
}

conn, err = t.tlsManager.WrapConn(conn)
if err != nil {
conn.Close()
return fmt.Errorf("failed to create TLS connection: %w", err)
}

t.logger.Info("Connected to server",
zap.String("address", addr),
)

t.mu.Lock()
if t.transfer != nil {
t.transfer.Stop()
}
t.transfer = NewTransfer(t.logger, t.iface, conn, float64(t.config.UploadKbps), float64(t.config.DownloadKbps))
t.mu.Unlock()

t.wg.Add(1)
go func() {
defer t.wg.Done()
t.transfer.Start()
}()

return nil
}

// Stop stops the tunnel
func (t *Tunnel) Stop() error {
t.mu.Lock()
defer t.mu.Unlock()

t.cancel()

if t.transfer != nil {
t.transfer.Stop()
}

if t.listener != nil {
t.listener.Close()
}

t.wg.Wait()
return nil
}

// GetStatistics returns current tunnel statistics
func (t *Tunnel) GetStatistics() *Statistics {
t.mu.Lock()
defer t.mu.Unlock()

if t.transfer != nil {
stats := t.transfer.GetStatistics()
return &stats
}
return nil
}
