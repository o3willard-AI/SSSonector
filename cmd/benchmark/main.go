package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"time"

	"github.com/o3willard-AI/SSSonector/internal/benchmark"
	"go.uber.org/zap"
)

func main() {
	// Parse command line flags
	concurrency := flag.Int("c", 10, "number of concurrent workers")
	duration := flag.Duration("d", 30*time.Second, "duration to run the benchmark")
	rateLimit := flag.Int("r", 1000, "maximum requests per second")
	host := flag.String("h", "localhost", "target host")
	port := flag.Int("p", 8080, "target port")
	flag.Parse()

	// Create logger
	logger, err := zap.NewDevelopment()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to create logger: %v\n", err)
		os.Exit(1)
	}
	defer logger.Sync()

	// Create benchmark config
	config := benchmark.Config{
		Concurrency: *concurrency,
		Duration:    *duration,
		RateLimit:   *rateLimit,
		TargetHost:  *host,
		TargetPort:  *port,
	}

	// Create context that can be cancelled
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Handle interrupt signal
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt)
	go func() {
		<-sigChan
		logger.Info("Received interrupt signal, shutting down...")
		cancel()
	}()

	// Run benchmark
	result, err := benchmark.Benchmark(ctx, config, logger)
	if err != nil {
		logger.Error("Benchmark failed", zap.Error(err))
		os.Exit(1)
	}

	// Print results
	fmt.Println(result)
}
