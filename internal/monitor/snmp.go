package monitor

import (
	"fmt"
	"net"
	"sync"
	"time"

	"github.com/o3willard-AI/SSSonector/internal/config"
	"github.com/shirou/gopsutil/v3/cpu"
	"github.com/shirou/gopsutil/v3/mem"
	"go.uber.org/zap"
)

// SNMPAgent handles SNMP monitoring
type SNMPAgent struct {
	logger *zap.Logger
	config *config.MonitorConfig
	conn   *net.UDPConn
	done   chan struct{}
	mu     sync.RWMutex
	stats  interface{}
}

// NewSNMPAgent creates a new SNMP agent
func NewSNMPAgent(logger *zap.Logger, cfg *config.MonitorConfig) (*SNMPAgent, error) {
	addr := fmt.Sprintf("%s:%d", cfg.SNMPAddress, cfg.SNMPPort)
	udpAddr, err := net.ResolveUDPAddr("udp", addr)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve UDP address: %w", err)
	}

	conn, err := net.ListenUDP("udp", udpAddr)
	if err != nil {
		return nil, fmt.Errorf("failed to listen on UDP: %w", err)
	}

	return &SNMPAgent{
		logger: logger,
		config: cfg,
		conn:   conn,
		done:   make(chan struct{}),
	}, nil
}

// Start starts the SNMP agent
func (a *SNMPAgent) Start() error {
	go a.collectMetrics()
	go a.handleRequests()
	return nil
}

// Stop stops the SNMP agent
func (a *SNMPAgent) Stop() error {
	close(a.done)
	return a.conn.Close()
}

// UpdateStats updates the agent's statistics
func (a *SNMPAgent) UpdateStats(stats interface{}) {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.stats = stats
}

// collectMetrics periodically collects system metrics
func (a *SNMPAgent) collectMetrics() {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-a.done:
			return
		case <-ticker.C:
			metrics := make(map[string]interface{})

			// CPU usage
			if cpuPercent, err := cpu.Percent(time.Second, false); err == nil {
				metrics["cpu_usage"] = cpuPercent[0]
			}

			// Memory usage
			if vmStat, err := mem.VirtualMemory(); err == nil {
				metrics["memory_used"] = vmStat.Used
				metrics["memory_total"] = vmStat.Total
			}

			// Add tunnel stats
			a.mu.RLock()
			if a.stats != nil {
				metrics["tunnel_stats"] = a.stats
			}
			a.mu.RUnlock()

			a.logger.Debug("Collected metrics",
				zap.Any("metrics", metrics),
			)
		}
	}
}

// handleRequests handles incoming SNMP requests
func (a *SNMPAgent) handleRequests() {
	buffer := make([]byte, 1024)

	for {
		select {
		case <-a.done:
			return
		default:
			n, remoteAddr, err := a.conn.ReadFromUDP(buffer)
			if err != nil {
				if !isClosedError(err) {
					a.logger.Error("Failed to read UDP packet",
						zap.Error(err),
					)
				}
				continue
			}

			go a.handleRequest(buffer[:n], remoteAddr)
		}
	}
}

// handleRequest processes a single SNMP request
func (a *SNMPAgent) handleRequest(data []byte, addr *net.UDPAddr) {
	// Basic SNMP request handling
	response := []byte{0x30, 0x03, 0x02, 0x01, 0x00} // Simple SNMP response
	_, err := a.conn.WriteToUDP(response, addr)
	if err != nil {
		a.logger.Error("Failed to send SNMP response",
			zap.Error(err),
			zap.String("remote_addr", addr.String()),
		)
	}
}

// isClosedError checks if the error is due to closed connection
func isClosedError(err error) bool {
	if err == nil {
		return false
	}
	return err.Error() == "use of closed network connection"
}
