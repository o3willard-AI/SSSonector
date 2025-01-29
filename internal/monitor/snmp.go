package monitor

import (
	"fmt"
	"net"
	"sync/atomic"

	"github.com/o3willard-AI/SSSonector/internal/config"
	"github.com/shirou/gopsutil/v3/cpu"
	"github.com/shirou/gopsutil/v3/mem"
	"go.uber.org/zap"
)

// SNMPAgent handles SNMP monitoring
type SNMPAgent struct {
	logger *zap.Logger
	conn   *net.UDPConn

	// Statistics
	bytesIn            atomic.Int64
	bytesOut           atomic.Int64
	activeConnections  atomic.Int32
	totalConnections   atomic.Int64
	connectionFailures atomic.Int64
	lastError          string
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

	agent := &SNMPAgent{
		logger: logger,
		conn:   conn,
	}

	// Start SNMP request handler
	go agent.handleRequests()

	return agent, nil
}

// Close closes the SNMP agent
func (a *SNMPAgent) Close() error {
	if a.conn != nil {
		return a.conn.Close()
	}
	return nil
}

// UpdateStats updates the SNMP statistics
func (a *SNMPAgent) UpdateStats(bytesIn, bytesOut int64, activeConnections int) {
	a.bytesIn.Store(bytesIn)
	a.bytesOut.Store(bytesOut)
	a.activeConnections.Store(int32(activeConnections))
}

// handleRequests handles incoming SNMP requests
func (a *SNMPAgent) handleRequests() {
	buffer := make([]byte, 1024)
	for {
		n, remoteAddr, err := a.conn.ReadFromUDP(buffer)
		if err != nil {
			if !isClosedError(err) {
				a.logger.Error("Failed to read SNMP request",
					zap.Error(err),
				)
			}
			return
		}

		// Process SNMP request
		response, err := a.processRequest(buffer[:n])
		if err != nil {
			a.logger.Error("Failed to process SNMP request",
				zap.Error(err),
			)
			continue
		}

		// Send response
		_, err = a.conn.WriteToUDP(response, remoteAddr)
		if err != nil {
			a.logger.Error("Failed to send SNMP response",
				zap.Error(err),
			)
		}
	}
}

// processRequest processes an SNMP request and returns a response
func (a *SNMPAgent) processRequest(request []byte) ([]byte, error) {
	// Get system stats
	v, err := mem.VirtualMemory()
	if err != nil {
		return nil, fmt.Errorf("failed to get memory stats: %w", err)
	}

	c, err := cpu.Percent(0, false)
	if err != nil {
		return nil, fmt.Errorf("failed to get CPU stats: %w", err)
	}

	// Build response with current stats
	stats := map[string]interface{}{
		"bytes_in":            a.bytesIn.Load(),
		"bytes_out":           a.bytesOut.Load(),
		"active_connections":  a.activeConnections.Load(),
		"total_connections":   a.totalConnections.Load(),
		"connection_failures": a.connectionFailures.Load(),
		"last_error":          a.lastError,
		"system_memory_total": v.Total,
		"system_memory_used":  v.Used,
		"system_cpu_percent":  c[0],
	}

	// Encode response as SNMP PDU
	// This is a simplified implementation - in a real system,
	// you would properly encode according to SNMP protocol specs
	response := encodeStats(stats)
	return response, nil
}

// isClosedError checks if an error indicates the connection is closed
func isClosedError(err error) bool {
	if err == nil {
		return false
	}
	return err.Error() == "use of closed network connection"
}

// encodeStats encodes stats as a simple byte slice
// In a real implementation, this would properly encode as SNMP PDUs
func encodeStats(stats map[string]interface{}) []byte {
	// Simplified implementation - in reality, you would properly
	// encode according to SNMP protocol specifications
	return []byte(fmt.Sprintf("%v", stats))
}
