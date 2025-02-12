package benchmark

import (
	"context"
	"fmt"
	"net"
	"sort"
	"sync"
	"time"

	"github.com/o3willard-AI/SSSonector/internal/config/types"
	"github.com/o3willard-AI/SSSonector/internal/pool"
	"go.uber.org/zap"
)

// Result represents benchmark results
type Result struct {
	TotalRequests      int64
	SuccessfulRequests int64
	FailedRequests     int64
	Duration           time.Duration
	AverageLatency     time.Duration
	P95Latency         time.Duration
	P99Latency         time.Duration
}

// String returns a string representation of the benchmark results
func (r Result) String() string {
	return fmt.Sprintf(`
Benchmark Results:
-----------------
Total Requests:       %d
Successful Requests: %d
Failed Requests:     %d
Duration:           %v
Average Latency:    %v
P95 Latency:        %v
P99 Latency:        %v
`,
		r.TotalRequests,
		r.SuccessfulRequests,
		r.FailedRequests,
		r.Duration,
		r.AverageLatency,
		r.P95Latency,
		r.P99Latency,
	)
}

// Config represents benchmark configuration
type Config struct {
	// Concurrency is the number of concurrent workers
	Concurrency int

	// Duration is how long to run the benchmark
	Duration time.Duration

	// RateLimit is the maximum number of requests per second
	RateLimit int

	// TargetHost is the host to connect to
	TargetHost string

	// TargetPort is the port to connect to
	TargetPort int
}

// Benchmark runs a connection benchmark
func Benchmark(ctx context.Context, config Config, logger *zap.Logger) (*Result, error) {
	if config.Concurrency <= 0 {
		return nil, fmt.Errorf("concurrency must be greater than 0")
	}

	// Create connection pool
	poolConfig := pool.Config{
		InitialSize:    config.Concurrency,
		MaxSize:        config.Concurrency * 2,
		MinSize:        config.Concurrency,
		MaxIdleTime:    types.NewDuration(time.Minute),
		ConnectTimeout: types.NewDuration(5 * time.Second),
		HealthCheck: func(conn net.Conn) error {
			// Simple health check - just verify we can write/read
			deadline := time.Now().Add(time.Second)
			if err := conn.SetDeadline(deadline); err != nil {
				return fmt.Errorf("failed to set deadline: %w", err)
			}
			if _, err := conn.Write([]byte("ping")); err != nil {
				return fmt.Errorf("failed to write: %w", err)
			}
			buf := make([]byte, 4)
			if _, err := conn.Read(buf); err != nil {
				return fmt.Errorf("failed to read: %w", err)
			}
			return nil
		},
		Factory: func(ctx context.Context) (net.Conn, error) {
			addr := fmt.Sprintf("%s:%d", config.TargetHost, config.TargetPort)
			var d net.Dialer
			return d.DialContext(ctx, "tcp", addr)
		},
	}

	p, err := pool.NewPool(poolConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create connection pool: %w", err)
	}
	defer p.Close()

	// Create channels for coordinating workers
	results := make(chan time.Duration, config.Concurrency*100)
	done := make(chan struct{})

	// Create context with timeout
	ctx, cancel := context.WithTimeout(ctx, config.Duration)
	defer cancel()

	// Start workers
	var wg sync.WaitGroup
	for i := 0; i < config.Concurrency; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for {
				select {
				case <-ctx.Done():
					return
				case <-done:
					return
				default:
					start := time.Now()
					conn, err := p.Get(ctx)
					if err != nil {
						if ctx.Err() != nil {
							return
						}
						logger.Error("Failed to get connection", zap.Error(err))
						continue
					}

					// Simulate work
					time.Sleep(time.Millisecond)

					if err := p.Put(conn); err != nil {
						logger.Error("Failed to return connection", zap.Error(err))
					}

					results <- time.Since(start)

					// Rate limiting
					if config.RateLimit > 0 {
						time.Sleep(time.Second / time.Duration(config.RateLimit))
					}
				}
			}
		}()
	}

	// Wait for duration or context cancellation
	<-ctx.Done()
	close(done)

	// Wait for workers to finish
	wg.Wait()
	close(results)

	// Calculate statistics
	var latencies []time.Duration
	var totalLatency time.Duration
	var successCount int64
	for latency := range results {
		latencies = append(latencies, latency)
		totalLatency += latency
		successCount++
	}

	if len(latencies) == 0 {
		return nil, fmt.Errorf("no results collected")
	}

	// Sort latencies for percentile calculations
	sort.Slice(latencies, func(i, j int) bool {
		return latencies[i] < latencies[j]
	})

	return &Result{
		TotalRequests:      successCount,
		SuccessfulRequests: successCount,
		FailedRequests:     0, // We're not tracking failures in this implementation
		Duration:           config.Duration,
		AverageLatency:     totalLatency / time.Duration(len(latencies)),
		P95Latency:         percentile(latencies, 95),
		P99Latency:         percentile(latencies, 99),
	}, nil
}

func percentile(sorted []time.Duration, p int) time.Duration {
	if len(sorted) == 0 {
		return 0
	}
	rank := (float64(p) / 100.0) * float64(len(sorted)-1)
	index := int(rank)
	if index >= len(sorted)-1 {
		return sorted[len(sorted)-1]
	}
	frac := rank - float64(index)
	return time.Duration(float64(sorted[index]) + frac*float64(sorted[index+1]-sorted[index]))
}
