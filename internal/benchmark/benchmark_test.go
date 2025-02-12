package benchmark

import (
	"context"
	"net"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap/zaptest"
)

type mockServer struct {
	listener net.Listener
	done     chan struct{}
	wg       sync.WaitGroup
}

func newMockServer(t *testing.T) *mockServer {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)

	server := &mockServer{
		listener: listener,
		done:     make(chan struct{}),
	}

	server.wg.Add(1)
	go server.serve()
	return server
}

func (s *mockServer) serve() {
	defer s.wg.Done()
	for {
		select {
		case <-s.done:
			return
		default:
			conn, err := s.listener.Accept()
			if err != nil {
				continue
			}
			s.wg.Add(1)
			go s.handleConnection(conn)
		}
	}
}

func (s *mockServer) handleConnection(conn net.Conn) {
	defer s.wg.Done()
	defer conn.Close()

	buf := make([]byte, 4)
	for {
		select {
		case <-s.done:
			return
		default:
			if err := conn.SetDeadline(time.Now().Add(100 * time.Millisecond)); err != nil {
				return
			}
			_, err := conn.Read(buf)
			if err != nil {
				return
			}
			_, err = conn.Write(buf)
			if err != nil {
				return
			}
		}
	}
}

func (s *mockServer) close() {
	close(s.done)
	s.listener.Close()
	s.wg.Wait()
}

func (s *mockServer) addr() string {
	return s.listener.Addr().String()
}

func TestBenchmark(t *testing.T) {
	logger := zaptest.NewLogger(t)

	t.Run("basic benchmark", func(t *testing.T) {
		server := newMockServer(t)
		defer server.close()

		host, port, err := net.SplitHostPort(server.addr())
		require.NoError(t, err)

		config := Config{
			Concurrency: 2,
			Duration:    100 * time.Millisecond,
			RateLimit:   1000,
			TargetHost:  host,
			TargetPort:  atoi(port),
		}

		result, err := Benchmark(context.Background(), config, logger)
		require.NoError(t, err)
		assert.Greater(t, result.TotalRequests, int64(0))
		assert.Equal(t, result.SuccessfulRequests, result.TotalRequests)
		assert.Equal(t, int64(0), result.FailedRequests)
		assert.Equal(t, config.Duration, result.Duration)
		assert.Greater(t, result.AverageLatency, time.Duration(0))
		assert.Greater(t, result.P95Latency, time.Duration(0))
		assert.Greater(t, result.P99Latency, time.Duration(0))
	})

	t.Run("zero concurrency", func(t *testing.T) {
		config := Config{
			Concurrency: 0,
			Duration:    100 * time.Millisecond,
			RateLimit:   1000,
			TargetHost:  "localhost",
			TargetPort:  8080,
		}

		result, err := Benchmark(context.Background(), config, logger)
		assert.Error(t, err)
		assert.Nil(t, result)
	})

	t.Run("zero duration", func(t *testing.T) {
		server := newMockServer(t)
		defer server.close()

		host, port, err := net.SplitHostPort(server.addr())
		require.NoError(t, err)

		config := Config{
			Concurrency: 2,
			Duration:    0,
			RateLimit:   1000,
			TargetHost:  host,
			TargetPort:  atoi(port),
		}

		result, err := Benchmark(context.Background(), config, logger)
		assert.Error(t, err)
		assert.Nil(t, result)
	})

	t.Run("connection failures", func(t *testing.T) {
		config := Config{
			Concurrency: 2,
			Duration:    100 * time.Millisecond,
			RateLimit:   1000,
			TargetHost:  "invalid-host",
			TargetPort:  0,
		}

		result, err := Benchmark(context.Background(), config, logger)
		assert.Error(t, err)
		assert.Nil(t, result)
	})

	t.Run("context cancellation", func(t *testing.T) {
		server := newMockServer(t)
		defer server.close()

		host, port, err := net.SplitHostPort(server.addr())
		require.NoError(t, err)

		config := Config{
			Concurrency: 2,
			Duration:    time.Second,
			RateLimit:   1000,
			TargetHost:  host,
			TargetPort:  atoi(port),
		}

		ctx, cancel := context.WithCancel(context.Background())
		cancel() // Cancel immediately

		result, err := Benchmark(ctx, config, logger)
		assert.Error(t, err)
		assert.Nil(t, result)
	})
}

func TestResult_String(t *testing.T) {
	result := &Result{
		TotalRequests:      100,
		SuccessfulRequests: 95,
		FailedRequests:     5,
		Duration:           time.Second,
		AverageLatency:     10 * time.Millisecond,
		P95Latency:         20 * time.Millisecond,
		P99Latency:         30 * time.Millisecond,
	}

	str := result.String()
	assert.Contains(t, str, "Total Requests:       100")
	assert.Contains(t, str, "Successful Requests: 95")
	assert.Contains(t, str, "Failed Requests:     5")
	assert.Contains(t, str, "Duration:           1s")
	assert.Contains(t, str, "Average Latency:    10ms")
	assert.Contains(t, str, "P95 Latency:        20ms")
	assert.Contains(t, str, "P99 Latency:        30ms")
}

func atoi(s string) int {
	var n int
	for _, c := range s {
		n = n*10 + int(c-'0')
	}
	return n
}
