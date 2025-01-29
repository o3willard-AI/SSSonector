package monitor

import (
	"encoding/asn1"
	"fmt"
	"net"
	"strconv"
	"sync"
	"time"

	"go.uber.org/zap"
)

// SNMPAgent handles SNMP monitoring
type SNMPAgent struct {
	logger    *zap.Logger
	port      int
	community string
	conn      *net.UDPConn
	metrics   *MetricsCollector
	done      chan struct{}
	wg        sync.WaitGroup
}

// NewSNMPAgent creates a new SNMP agent
func NewSNMPAgent(logger *zap.Logger, port int, community string) *SNMPAgent {
	return &SNMPAgent{
		logger:    logger,
		port:      port,
		community: community,
		metrics:   NewMetricsCollector(),
		done:      make(chan struct{}),
	}
}

// Start starts the SNMP agent
func (a *SNMPAgent) Start() error {
	addr := &net.UDPAddr{
		IP:   net.ParseIP("0.0.0.0"),
		Port: a.port,
	}

	var err error
	a.conn, err = net.ListenUDP("udp", addr)
	if err != nil {
		return fmt.Errorf("failed to start SNMP server: %w", err)
	}

	a.wg.Add(1)
	go a.serve()

	a.logger.Info("SNMP agent started",
		zap.Int("port", a.port),
		zap.String("community", a.community),
	)

	return nil
}

// Stop stops the SNMP agent
func (a *SNMPAgent) Stop() error {
	close(a.done)
	if a.conn != nil {
		a.conn.Close()
	}
	a.wg.Wait()
	return nil
}

// UpdateStats updates the agent's metrics
func (a *SNMPAgent) UpdateStats(bytesReceived, bytesSent, packetsLost uint64, latency int64) {
	a.metrics.UpdateNetworkMetrics(bytesReceived, bytesSent, packetsLost, latency)
}

// serve handles incoming SNMP requests
func (a *SNMPAgent) serve() {
	defer a.wg.Done()

	buffer := make([]byte, 4096)
	for {
		select {
		case <-a.done:
			return
		default:
			a.conn.SetReadDeadline(time.Now().Add(1 * time.Second))
			n, remoteAddr, err := a.conn.ReadFromUDP(buffer)
			if err != nil {
				if netErr, ok := err.(net.Error); ok && netErr.Timeout() {
					continue
				}
				a.logger.Error("Failed to read UDP packet", zap.Error(err))
				continue
			}

			response, err := a.handleRequest(buffer[:n])
			if err != nil {
				a.logger.Error("Failed to handle SNMP request",
					zap.Error(err),
					zap.String("remote_addr", remoteAddr.String()),
				)
				continue
			}

			if response != nil {
				_, err = a.conn.WriteToUDP(response, remoteAddr)
				if err != nil {
					a.logger.Error("Failed to send SNMP response",
						zap.Error(err),
						zap.String("remote_addr", remoteAddr.String()),
					)
				}
			}
		}
	}
}

// handleRequest processes an SNMP request and returns a response
func (a *SNMPAgent) handleRequest(request []byte) ([]byte, error) {
	// Parse SNMP request
	var snmpPdu struct {
		Version   int
		Community string
		Data      asn1.RawValue
	}

	_, err := asn1.Unmarshal(request, &snmpPdu)
	if err != nil {
		return nil, fmt.Errorf("failed to parse SNMP request: %w", err)
	}

	// Verify community string
	if snmpPdu.Community != a.community {
		return nil, fmt.Errorf("invalid community string")
	}

	// Get current metrics
	metrics := a.metrics.GetSnapshot()

	// Build response based on request type
	var response []byte
	switch snmpPdu.Data.Tag {
	case 0xa0: // GetRequest
		response, err = a.handleGetRequest(request, metrics)
	case 0xa1: // GetNextRequest
		response, err = a.handleGetNextRequest(request, metrics)
	default:
		return nil, fmt.Errorf("unsupported SNMP request type: %d", snmpPdu.Data.Tag)
	}

	if err != nil {
		return nil, fmt.Errorf("failed to handle SNMP request: %w", err)
	}

	return response, nil
}

// handleGetRequest processes an SNMP GET request
func (a *SNMPAgent) handleGetRequest(request []byte, metrics *Metrics) ([]byte, error) {
	// Create response with same structure but updated values
	response := make([]byte, len(request))
	copy(response, request)

	// Update response type to GetResponse
	response[len(response)-len(request)+16] = 0xa2

	// Update values based on OIDs
	// Note: This is a simplified implementation
	// In a real implementation, we would parse the OIDs and update values accordingly

	return response, nil
}

// handleGetNextRequest processes an SNMP GETNEXT request
func (a *SNMPAgent) handleGetNextRequest(request []byte, metrics *Metrics) ([]byte, error) {
	// Similar to handleGetRequest but returns next OID in sequence
	response := make([]byte, len(request))
	copy(response, request)

	// Update response type to GetResponse
	response[len(response)-len(request)+16] = 0xa2

	return response, nil
}

// getLastOIDNumber extracts the last number from an OID
func getLastOIDNumber(oid string) (int, error) {
	// Find the last dot in the OID
	lastDot := -1
	for i := len(oid) - 1; i >= 0; i-- {
		if oid[i] == '.' {
			lastDot = i
			break
		}
	}
	if lastDot == -1 {
		return 0, fmt.Errorf("invalid OID format")
	}

	// Parse the number after the last dot
	num, err := strconv.Atoi(oid[lastDot+1:])
	if err != nil {
		return 0, fmt.Errorf("invalid OID number: %w", err)
	}

	return num, nil
}
