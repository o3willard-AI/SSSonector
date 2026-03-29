package facade

import (
	"context"
	"fmt"
	"io"
	"net"
	"sync"

	"go.uber.org/zap"
)

// Proxy bridges a hijacked HTTP connection to a local tunnel port.
// It performs bidirectional byte copying between the client connection
// (arriving via the HTTPS facade on port 443) and the local tunnel
// listener (on the configured tunnel port, e.g. 8443).
//
// The proxy does not perform rate limiting -- that is handled by the
// tunnel instance itself once it accepts the proxied connection.
func Proxy(ctx context.Context, clientConn net.Conn, tunnelAddr string, logger *zap.Logger) error {
	// Dial the local tunnel port
	tunnelConn, err := net.Dial("tcp", tunnelAddr)
	if err != nil {
		return fmt.Errorf("failed to connect to tunnel at %s: %w", tunnelAddr, err)
	}

	logger.Debug("Proxy established",
		zap.String("client", clientConn.RemoteAddr().String()),
		zap.String("tunnel", tunnelAddr),
	)

	// Bidirectional copy
	var wg sync.WaitGroup
	errChan := make(chan error, 2)

	wg.Add(2)

	// Client -> Tunnel
	go func() {
		defer wg.Done()
		_, err := io.Copy(tunnelConn, clientConn)
		// When one direction finishes, close write side to signal the other end
		if tc, ok := tunnelConn.(*net.TCPConn); ok {
			tc.CloseWrite()
		}
		errChan <- err
	}()

	// Tunnel -> Client
	go func() {
		defer wg.Done()
		_, err := io.Copy(clientConn, tunnelConn)
		if tc, ok := clientConn.(*net.TCPConn); ok {
			tc.CloseWrite()
		}
		errChan <- err
	}()

	// Wait for context cancellation or both directions to complete
	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
	}()

	select {
	case <-ctx.Done():
		// Context cancelled -- close both connections to unblock io.Copy
		clientConn.Close()
		tunnelConn.Close()
		<-done
	case <-done:
		// Both directions completed normally
	}

	// Clean up
	clientConn.Close()
	tunnelConn.Close()

	// Collect errors (non-nil errors from io.Copy after close are expected)
	var proxyErr error
	for i := 0; i < 2; i++ {
		select {
		case e := <-errChan:
			if e != nil && proxyErr == nil {
				proxyErr = e
			}
		default:
		}
	}

	logger.Debug("Proxy closed",
		zap.String("client", clientConn.RemoteAddr().String()),
		zap.String("tunnel", tunnelAddr),
	)

	return proxyErr
}
