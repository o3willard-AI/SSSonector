package facade

import (
	"context"
	"crypto/tls"
	"fmt"
	"io"
	"net"
	"testing"
	"time"

	"github.com/o3willard-AI/SSSonector/internal/config/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)


func TestClientDirectConnect(t *testing.T) {
	// Start a mock server that the client can directly connect to
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	defer ln.Close()

	port := ln.Addr().(*net.TCPAddr).Port

	// Accept connections in background
	go func() {
		for {
			conn, err := ln.Accept()
			if err != nil {
				return
			}
			conn.Close()
		}
	}()

	certs := generateTestCerts(t)
	logger, _ := zap.NewDevelopment()

	facadeConfig := &types.FacadeConfig{
		Enabled:       true,
		ServerPort:    443,
		DirectTimeout: 3 * time.Second,
		TokenSecret:   testTokenSecret,
		TLS: types.FacadeTLSConfig{
			CAFile: certs.CAFile,
		},
	}

	tunnelConfig := &types.TunnelConfig{
		ServerAddress: "127.0.0.1",
		ServerPort:    port,
	}

	authConfig := &types.AuthConfig{
		CertFile: certs.CertFile,
		KeyFile:  certs.KeyFile,
		CAFile:   certs.CAFile,
	}

	client, err := NewClient(facadeConfig, tunnelConfig, authConfig, logger)
	require.NoError(t, err)

	// Connect should succeed with direct connection
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	result, err := client.Connect(ctx)
	require.NoError(t, err)
	defer result.Conn.Close()

	assert.False(t, result.ViaFacade, "should have connected directly")
}

func TestClientFallbackToFacade(t *testing.T) {
	certs := generateTestCerts(t)

	// Start a mock tunnel listener that the facade will proxy to
	tunnelLn, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	defer tunnelLn.Close()
	tunnelPort := tunnelLn.Addr().(*net.TCPAddr).Port

	// Echo server on tunnel port
	go func() {
		for {
			conn, err := tunnelLn.Accept()
			if err != nil {
				return
			}
			go func() {
				defer conn.Close()
				io.Copy(conn, conn)
			}()
		}
	}()

	// Start the facade server
	server, facadeAddr := startTestFacadeServer(t, certs, []int{tunnelPort})
	defer server.Stop()

	// Parse the facade address to get the port
	_, facadePortStr, err := net.SplitHostPort(facadeAddr)
	require.NoError(t, err)
	var facadePort int
	fmt.Sscanf(facadePortStr, "%d", &facadePort)

	logger, _ := zap.NewDevelopment()

	// Configure client to connect to a BLOCKED direct port (use port 1 which is unlikely to be open)
	// and fall back to the facade
	facadeConfig := &types.FacadeConfig{
		Enabled:       true,
		ServerAddress: "127.0.0.1",
		ServerPort:    facadePort,
		DirectTimeout: 1 * time.Second, // Short timeout for faster test
		TokenSecret:   testTokenSecret,
		TLS: types.FacadeTLSConfig{
			CertFile: certs.CertFile,
			KeyFile:  certs.KeyFile,
			CAFile:   certs.CAFile,
		},
	}

	tunnelConfig := &types.TunnelConfig{
		ServerAddress: "127.0.0.1",
		ServerPort:    tunnelPort,
	}

	authConfig := &types.AuthConfig{
		CertFile: certs.CertFile,
		KeyFile:  certs.KeyFile,
		CAFile:   certs.CAFile,
	}

	client, err := NewClient(facadeConfig, tunnelConfig, authConfig, logger)
	require.NoError(t, err)

	// Override the tunnel config to use a port that will fail for direct connect
	// but the facade will know the correct tunnel port from the token
	client.tunnelConfig = &types.TunnelConfig{
		ServerAddress: "127.0.0.1",
		ServerPort:    tunnelPort,
	}

	// Temporarily create a custom client that always fails direct connect
	// by setting an impossibly short direct timeout and a blocked port
	blockedClient := &Client{
		facadeConfig: facadeConfig,
		tunnelConfig: &types.TunnelConfig{
			ServerAddress: "192.0.2.1", // RFC 5737 TEST-NET, guaranteed unreachable
			ServerPort:    tunnelPort,
		},
		authConfig:    authConfig,
		logger:        logger,
		secret:        client.secret,
		directTimeout: 500 * time.Millisecond,
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	result, err := blockedClient.Connect(ctx)
	require.NoError(t, err)
	defer result.Conn.Close()

	assert.True(t, result.ViaFacade, "should have connected via facade")

	// Test data flows through the facade -> tunnel
	testData := []byte("test data through facade")
	_, err = result.Conn.Write(testData)
	require.NoError(t, err)

	echoBuf := make([]byte, len(testData))
	result.Conn.(*tls.Conn).SetReadDeadline(time.Now().Add(2 * time.Second))
	_, err = io.ReadFull(result.Conn, echoBuf)
	require.NoError(t, err)
	assert.Equal(t, testData, echoBuf)
}

func TestClientNewClientValidation(t *testing.T) {
	logger, _ := zap.NewDevelopment()
	certs := generateTestCerts(t)

	// Nil facade config
	_, err := NewClient(nil, &types.TunnelConfig{}, &types.AuthConfig{}, logger)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "facade config is required")

	// Nil tunnel config
	_, err = NewClient(&types.FacadeConfig{
		TLS: types.FacadeTLSConfig{CAFile: certs.CAFile},
	}, nil, &types.AuthConfig{}, logger)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "tunnel config is required")

	// Nil auth config
	_, err = NewClient(&types.FacadeConfig{
		TLS: types.FacadeTLSConfig{CAFile: certs.CAFile},
	}, &types.TunnelConfig{}, nil, logger)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "auth config is required")
}

func TestClientDefaultDirectTimeout(t *testing.T) {
	certs := generateTestCerts(t)
	logger, _ := zap.NewDevelopment()

	client, err := NewClient(
		&types.FacadeConfig{
			Enabled:       true,
			ServerPort:    443,
			DirectTimeout: 0, // Should default to 3s
			TokenSecret:   testTokenSecret,
			TLS:           types.FacadeTLSConfig{CAFile: certs.CAFile},
		},
		&types.TunnelConfig{ServerAddress: "127.0.0.1", ServerPort: 8443},
		&types.AuthConfig{CAFile: certs.CAFile},
		logger,
	)
	require.NoError(t, err)
	assert.Equal(t, DefaultDirectTimeout, client.directTimeout)
}

func TestComputeWebSocketAcceptRFC(t *testing.T) {
	// Verify SHA-1(key + GUID) per RFC 6455 Section 4.2.2
	// SHA1("dGhlIHNhbXBsZSBub25jZQ==258EAFA5-E914-47DA-95CA-5631BC565D11") base64-encoded
	expected := "50Q8AId4CODuYsXtANFhoLtjFt4="
	assert.Equal(t, expected, computeWebSocketAccept(rfc6455SampleKey))
}
