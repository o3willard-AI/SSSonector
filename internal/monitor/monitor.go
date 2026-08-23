package monitor

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"runtime"
	"sync"
	"time"

	"go.uber.org/zap"
)

// Config holds monitoring configuration
type Config struct {
	SNMPEnabled   bool
	SNMPAddress   string
	SNMPPort      int
	SNMPCommunity string
	PromEnabled   bool
	PromPort      int
	PromPath      string
}

// Monitor handles system monitoring and metrics exposition
type Monitor struct {
	logger     *zap.Logger
	config     *Config
	metrics    *Metrics
	snmpAgent  *SNMPAgent
	promSrv    *http.Server
	promLn     net.Listener
	sysMetrics *SystemMetricsCollector
	startTime  time.Time
	mu         sync.RWMutex
	shutdownCh chan struct{}
	shutdownWg sync.WaitGroup
}

// New creates a new monitor instance. The logger is owned by the caller
// (the daemon's configured zap logger); the monitor no longer builds its own.
func New(logger *zap.Logger, cfg *Config) (*Monitor, error) {
	if logger == nil {
		return nil, fmt.Errorf("logger is required")
	}
	if cfg == nil {
		return nil, fmt.Errorf("monitor config is required")
	}

	m := &Monitor{
		logger:     logger,
		config:     cfg,
		metrics:    NewMetrics(),
		sysMetrics: NewSystemMetricsCollector(),
		startTime:  time.Now(),
		shutdownCh: make(chan struct{}),
	}

	// Initialize SNMP agent if enabled
	if cfg.SNMPEnabled {
		var err error
		m.snmpAgent, err = NewSNMPAgent(cfg, m.metrics, logger)
		if err != nil {
			return nil, fmt.Errorf("failed to create SNMP agent: %w", err)
		}
	}

	return m, nil
}

// Start initializes monitoring. Endpoint bind failures are logged and do not
// prevent the rest of monitoring from running (degraded, non-fatal).
func (m *Monitor) Start() error {
	if m.config.SNMPEnabled && m.snmpAgent != nil {
		if err := m.snmpAgent.Start(); err != nil {
			m.logger.Error("Failed to start SNMP agent", zap.Error(err))
		} else {
			m.logger.Info("SNMP monitoring started",
				zap.String("address", m.config.SNMPAddress),
				zap.Int("port", m.config.SNMPPort))
		}
	}

	if m.config.PromEnabled && m.config.PromPort > 0 {
		if err := m.startPrometheus(); err != nil {
			m.logger.Error("Failed to start Prometheus endpoint", zap.Error(err))
		}
	}

	// Start system metrics collection
	m.shutdownWg.Add(1)
	go m.collectSystemMetrics()

	return nil
}

// Stop shuts down monitoring
func (m *Monitor) Stop() {
	select {
	case <-m.shutdownCh:
		// Already shutting down
		return
	default:
		close(m.shutdownCh)
	}

	if m.promSrv != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := m.promSrv.Shutdown(ctx); err != nil {
			m.logger.Error("Prometheus endpoint shutdown error", zap.Error(err))
		}
		m.mu.Lock()
		m.promSrv = nil
		m.promLn = nil
		m.mu.Unlock()
	}

	if m.config.SNMPEnabled && m.snmpAgent != nil {
		m.snmpAgent.Stop()
		m.logger.Info("SNMP monitoring stopped")
	}

	m.shutdownWg.Wait()

	// Sync errors from console/stdout sinks are benign (EINVAL on Linux);
	// surface anything else at debug level only.
	if err := m.logger.Sync(); err != nil {
		m.logger.Debug("Logger sync returned error", zap.Error(err))
	}
}

// PromEndpoint returns the address the Prometheus endpoint is serving on,
// or an empty string when it is disabled or not yet started.
func (m *Monitor) PromEndpoint() string {
	m.mu.RLock()
	defer m.mu.RUnlock()
	if m.promLn == nil {
		return ""
	}
	return m.promLn.Addr().String()
}

// startPrometheus binds and serves the /metrics text exposition endpoint
func (m *Monitor) startPrometheus() error {
	path := m.config.PromPath
	if path == "" {
		path = "/metrics"
	}

	mux := http.NewServeMux()
	mux.HandleFunc(path, m.handleMetrics)

	ln, err := net.Listen("tcp", fmt.Sprintf(":%d", m.config.PromPort))
	if err != nil {
		return fmt.Errorf("failed to bind Prometheus listener on port %d: %w", m.config.PromPort, err)
	}

	srv := &http.Server{
		Handler:           mux,
		ReadHeaderTimeout: 5 * time.Second,
	}
	m.mu.Lock()
	m.promLn = ln
	m.promSrv = srv
	m.mu.Unlock()

	m.shutdownWg.Add(1)
	go func() {
		defer m.shutdownWg.Done()
		if err := srv.Serve(ln); err != nil && err != http.ErrServerClosed {
			m.logger.Error("Prometheus endpoint server error", zap.Error(err))
		}
	}()

	m.logger.Info("Prometheus metrics endpoint started",
		zap.String("address", ln.Addr().String()),
		zap.String("path", path),
	)
	return nil
}

// handleMetrics renders current metrics in Prometheus text exposition format
func (m *Monitor) handleMetrics(w http.ResponseWriter, r *http.Request) {
	snap := m.GetMetrics()

	w.Header().Set("Content-Type", "text/plain; version=0.0.4; charset=utf-8")
	fmt.Fprintf(w, "# HELP sssonector_bytes_in_total Total bytes received from tunnel peers.\n")
	fmt.Fprintf(w, "# TYPE sssonector_bytes_in_total counter\n")
	fmt.Fprintf(w, "sssonector_bytes_in_total %d\n", snap.BytesIn)
	fmt.Fprintf(w, "# HELP sssonector_bytes_out_total Total bytes sent to tunnel peers.\n")
	fmt.Fprintf(w, "# TYPE sssonector_bytes_out_total counter\n")
	fmt.Fprintf(w, "sssonector_bytes_out_total %d\n", snap.BytesOut)
	fmt.Fprintf(w, "# HELP sssonector_packets_in_total Total packets received.\n")
	fmt.Fprintf(w, "# TYPE sssonector_packets_in_total counter\n")
	fmt.Fprintf(w, "sssonector_packets_in_total %d\n", snap.PacketsIn)
	fmt.Fprintf(w, "# HELP sssonector_packets_out_total Total packets sent.\n")
	fmt.Fprintf(w, "# TYPE sssonector_packets_out_total counter\n")
	fmt.Fprintf(w, "sssonector_packets_out_total %d\n", snap.PacketsOut)
	fmt.Fprintf(w, "# HELP sssonector_errors_total Total errors recorded.\n")
	fmt.Fprintf(w, "# TYPE sssonector_errors_total counter\n")
	fmt.Fprintf(w, "sssonector_errors_total %d\n", snap.Errors)
	fmt.Fprintf(w, "# HELP sssonector_byte_rate Current byte throughput per second.\n")
	fmt.Fprintf(w, "# TYPE sssonector_byte_rate gauge\n")
	fmt.Fprintf(w, "sssonector_byte_rate %f\n", snap.ByteRate)
	fmt.Fprintf(w, "# HELP sssonector_connections_active Currently active tunnel connections.\n")
	fmt.Fprintf(w, "# TYPE sssonector_connections_active gauge\n")
	fmt.Fprintf(w, "sssonector_connections_active %d\n", snap.Connections)
	fmt.Fprintf(w, "# HELP sssonector_connections_peak Peak concurrent tunnel connections.\n")
	fmt.Fprintf(w, "# TYPE sssonector_connections_peak gauge\n")
	fmt.Fprintf(w, "sssonector_connections_peak %d\n", snap.MaxConnections)
	fmt.Fprintf(w, "# HELP sssonector_throttle_hits_total Requests that had to wait for tokens.\n")
	fmt.Fprintf(w, "# TYPE sssonector_throttle_hits_total counter\n")
	fmt.Fprintf(w, "sssonector_throttle_hits_total{direction=\"in\"} %d\n", snap.ThrottleHitsIn)
	fmt.Fprintf(w, "sssonector_throttle_hits_total{direction=\"out\"} %d\n", snap.ThrottleHitsOut)
	fmt.Fprintf(w, "# HELP sssonector_throttle_effective_rate_bytes_per_second Effective paced rate including TCP overhead.\n")
	fmt.Fprintf(w, "# TYPE sssonector_throttle_effective_rate_bytes_per_second gauge\n")
	fmt.Fprintf(w, "sssonector_throttle_effective_rate_bytes_per_second %f\n", snap.ThrottleRate)
	fmt.Fprintf(w, "# HELP sssonector_throttle_burst_bytes Burst allowance in bytes.\n")
	fmt.Fprintf(w, "# TYPE sssonector_throttle_burst_bytes gauge\n")
	fmt.Fprintf(w, "sssonector_throttle_burst_bytes %f\n", snap.ThrottleBurst)
	fmt.Fprintf(w, "# HELP sssonector_cpu_usage_percent Process CPU usage percentage.\n")
	fmt.Fprintf(w, "# TYPE sssonector_cpu_usage_percent gauge\n")
	fmt.Fprintf(w, "sssonector_cpu_usage_percent %f\n", snap.CPUUsage)
	fmt.Fprintf(w, "# HELP sssonector_memory_alloc_bytes Bytes allocated by the process heap.\n")
	fmt.Fprintf(w, "# TYPE sssonector_memory_alloc_bytes gauge\n")
	fmt.Fprintf(w, "sssonector_memory_alloc_bytes %d\n", snap.MemoryUsage)
	fmt.Fprintf(w, "# HELP sssonector_goroutines Current goroutine count.\n")
	fmt.Fprintf(w, "# TYPE sssonector_goroutines gauge\n")
	fmt.Fprintf(w, "sssonector_goroutines %d\n", snap.GoroutineNum)
	fmt.Fprintf(w, "# HELP sssonector_uptime_seconds Process uptime in seconds.\n")
	fmt.Fprintf(w, "# TYPE sssonector_uptime_seconds gauge\n")
	fmt.Fprintf(w, "sssonector_uptime_seconds %d\n", time.Since(m.startTime)/time.Second)
}

// UpdateMetrics updates monitoring metrics
func (m *Monitor) UpdateMetrics(bytesIn, bytesOut, packetsIn, packetsOut, errors int64, connections int) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.metrics.UpdateNetworkMetrics(bytesIn, bytesOut, packetsIn, packetsOut)
	m.metrics.UpdateErrorMetrics(errors, "", 0, 0) // No retry/drop info yet
	m.metrics.UpdateConnectionMetrics(int32(connections), int32(connections), 0, 0)
}

// UpdateThrottleMetrics records rate-limiter counters and the effective
// pacing configuration sampled from the live data path.
func (m *Monitor) UpdateThrottleMetrics(hitsIn, hitsOut uint64, rate, burst float64) {
	m.metrics.SetThrottle(hitsIn, hitsOut, rate, burst)
}

// GetMetrics returns current metrics
func (m *Monitor) GetMetrics() *Metrics {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.metrics.Clone()
}

// collectSystemMetrics periodically collects system-wide metrics
func (m *Monitor) collectSystemMetrics() {
	defer m.shutdownWg.Done()

	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()

	var memStats runtime.MemStats
	for {
		select {
		case <-m.shutdownCh:
			return
		case <-ticker.C:
			// Update memory stats
			runtime.ReadMemStats(&memStats)

			// Get number of goroutines
			numGoroutines := runtime.NumGoroutine()

			m.mu.Lock()
			// Update resource metrics
			m.metrics.UpdateResourceMetrics(
				m.metrics.CPUUsage, // Preserved from system metrics collector
				int64(memStats.Alloc),
				int64(memStats.HeapAlloc),
				0, // Queue length from tunnel
				int64(numGoroutines),
			)

			// Collect system metrics
			if err := m.sysMetrics.CollectMetrics(m.metrics); err != nil {
				m.logger.Error("Failed to collect system metrics", zap.Error(err))
			}
			m.mu.Unlock()
		}
	}
}
