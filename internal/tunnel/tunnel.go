package tunnel

import (
"fmt"
"net"

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
}

// NewTunnel creates a new tunnel instance
func NewTunnel(logger *zap.Logger, cfg *config.TunnelConfig, iface adapter.Interface) *Tunnel {
return &Tunnel{
logger: logger,
config: cfg,
iface:  iface,
}
}

// Start starts the tunnel in either server or client mode
func (t *Tunnel) Start(mode string) error {
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
return fmt.Errorf("failed to create TLS listener: %w", err)
}

t.logger.Info("Server listening",
zap.String("address", addr),
zap.Int("max_clients", t.config.MaxClients),
)

for {
conn, err := t.listener.Accept()
if err != nil {
t.logger.Error("Failed to accept connection", zap.Error(err))
continue
}

t.logger.Info("Client connected",
zap.String("remote_addr", conn.RemoteAddr().String()),
)

t.transfer = NewTransfer(t.logger, t.iface, conn, float64(t.config.UploadKbps), float64(t.config.DownloadKbps))
go t.transfer.Start()
}
}

// startClient starts the tunnel in client mode
func (t *Tunnel) startClient() error {
if t.iface == nil {
return ErrInterfaceNotInitialized
}

addr := fmt.Sprintf("%s:%d", t.config.ServerAddress, t.config.ServerPort)

conn, err := net.Dial("tcp", addr)
if err != nil {
return fmt.Errorf("failed to connect to server: %w", err)
}

conn, err = t.tlsManager.WrapConn(conn)
if err != nil {
return fmt.Errorf("failed to create TLS connection: %w", err)
}

t.logger.Info("Connected to server",
zap.String("address", addr),
)

t.transfer = NewTransfer(t.logger, t.iface, conn, float64(t.config.UploadKbps), float64(t.config.DownloadKbps))
t.transfer.Start()
return nil
}

// Stop stops the tunnel
func (t *Tunnel) Stop() error {
if t.transfer != nil {
t.transfer.Stop()
}

if t.listener != nil {
return t.listener.Close()
}

return nil
}

// GetStatistics returns current tunnel statistics
func (t *Tunnel) GetStatistics() *Statistics {
if t.transfer != nil {
stats := t.transfer.GetStatistics()
return &stats
}
return nil
}
