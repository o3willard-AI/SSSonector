package monitor

import (
	"fmt"
	"net"
	"runtime"
	"strings"
	"sync"
	"time"
	"unicode"

	"github.com/gosnmp/gosnmp"
	"go.uber.org/zap"
)

// SNMPAgent handles SNMP monitoring
type SNMPAgent struct {
	config      *Config
	metrics     *Metrics
	mibTree     *MIBTree
	conn        *net.UDPConn
	startTime   time.Time
	mu          sync.RWMutex
	logger      *zap.Logger
	requestPool sync.Pool
	stats       *SNMPStats
}

// SNMPStats tracks SNMP agent statistics
type SNMPStats struct {
	totalRequests      uint64
	invalidRequests    uint64
	authErrors         uint64
	successfulRequests uint64
	lastError          string
	lastErrorTime      time.Time
	mu                 sync.RWMutex
}

// NewSNMPAgent creates a new SNMP agent
func NewSNMPAgent(cfg *Config, metrics *Metrics, logger *zap.Logger) (*SNMPAgent, error) {
	agent := &SNMPAgent{
		config:    cfg,
		metrics:   metrics,
		startTime: time.Now(),
		logger:    logger,
		stats:     &SNMPStats{},
		requestPool: sync.Pool{
			New: func() interface{} {
				return make([]byte, 4096) // Increased buffer size for large packets
			},
		},
	}
	agent.mibTree = NewMIBTree(metrics)
	return agent, nil
}

// Start initializes the SNMP agent
func (a *SNMPAgent) Start() error {
	addr := &net.UDPAddr{
		IP:   net.ParseIP(a.config.SNMPAddress),
		Port: a.config.SNMPPort,
	}

	var err error
	a.conn, err = net.ListenUDP("udp", addr)
	if err != nil {
		return fmt.Errorf("failed to start SNMP listener: %w", err)
	}

	// Set socket buffer sizes
	if err := a.conn.SetReadBuffer(262144); err != nil { // 256KB read buffer
		a.logger.Error("Failed to set UDP read buffer size", zap.Error(err))
	}
	if err := a.conn.SetWriteBuffer(262144); err != nil { // 256KB write buffer
		a.logger.Error("Failed to set UDP write buffer size", zap.Error(err))
	}

	a.logger.Info("SNMP agent started",
		zap.String("address", a.config.SNMPAddress),
		zap.Int("port", a.config.SNMPPort),
		zap.String("community", a.config.SNMPCommunity))

	// Start request handlers
	for i := 0; i < 4; i++ { // Multiple handlers for concurrent processing
		go a.handleRequests()
	}

	// Start metrics reporting
	go a.reportMetrics()

	return nil
}

// reportMetrics periodically logs SNMP agent statistics
func (a *SNMPAgent) reportMetrics() {
	ticker := time.NewTicker(60 * time.Second)
	defer ticker.Stop()

	for range ticker.C {
		a.stats.mu.RLock()
		a.logger.Info("SNMP Stats",
			zap.Uint64("total_requests", a.stats.totalRequests),
			zap.Uint64("successful_requests", a.stats.successfulRequests),
			zap.Uint64("invalid_requests", a.stats.invalidRequests),
			zap.Uint64("auth_errors", a.stats.authErrors))
		if a.stats.lastError != "" {
			a.logger.Info("Last Error",
				zap.Time("time", a.stats.lastErrorTime),
				zap.String("error", a.stats.lastError))
		}
		a.stats.mu.RUnlock()
	}
}

// Stop shuts down the SNMP agent
func (a *SNMPAgent) Stop() {
	if a.conn != nil {
		a.conn.Close()
	}
}

// validateCommunity checks if the provided community string matches the configured one
func (a *SNMPAgent) validateCommunity(received string) bool {
	// Maximum length for community string (RFC 3584 recommends max 32 chars)
	const maxCommunityLength = 32

	// Clean and validate received community string
	receivedCommunity := cleanCommunityString(received)
	if receivedCommunity == "" || len(receivedCommunity) > maxCommunityLength {
		return false
	}

	// Clean configured community string
	configCommunity := cleanCommunityString(a.config.SNMPCommunity)
	if configCommunity == "" {
		return false
	}

	return configCommunity == receivedCommunity
}

// cleanCommunityString sanitizes a community string by:
// 1. Removing null bytes
// 2. Trimming whitespace from both ends
// 3. Ensuring only printable ASCII characters
func cleanCommunityString(community string) string {
	// Remove null bytes
	community = strings.ReplaceAll(community, "\x00", "")

	// Trim whitespace from both ends
	community = strings.TrimSpace(community)

	// Check for printable ASCII characters only
	for _, c := range community {
		if !unicode.IsPrint(c) {
			return ""
		}
	}

	return community
}

func (a *SNMPAgent) handleRequests() {
	for {
		buffer := a.requestPool.Get().([]byte)
		n, remoteAddr, err := a.conn.ReadFromUDP(buffer)

		if err != nil {
			if opErr, ok := err.(*net.OpError); ok && opErr.Err.Error() == "use of closed network connection" {
				a.requestPool.Put(buffer)
				return
			}
			a.logger.Error("Error reading UDP", zap.Error(err))
			a.requestPool.Put(buffer)
			continue
		}

		// Update request stats
		a.stats.mu.Lock()
		a.stats.totalRequests++
		a.stats.mu.Unlock()

		// Debug log raw packet
		a.logger.Debug("Received SNMP packet",
			zap.Int("bytes", n),
			zap.String("remote_addr", remoteAddr.String()),
			zap.Binary("data", buffer[:n]))

		// Parse incoming SNMP packet
		request, err := DecodeMessage(buffer[:n])
		a.requestPool.Put(buffer) // Return buffer to pool

		if err != nil {
			a.stats.mu.Lock()
			a.stats.invalidRequests++
			a.stats.lastError = fmt.Sprintf("Decode error: %v", err)
			a.stats.lastErrorTime = time.Now()
			a.stats.mu.Unlock()
			a.logger.Error("Error decoding SNMP packet",
				zap.String("remote_addr", remoteAddr.String()),
				zap.Error(err))
			continue
		}

		// Debug log decoded message
		a.logger.Debug("Decoded SNMP message",
			zap.String("remote_addr", remoteAddr.String()),
			zap.Int("version", int(request.Version)),
			zap.String("community", request.Community),
			zap.Int("type", int(request.PDUType)))

		// Verify community string
		if !a.validateCommunity(request.Community) {
			a.stats.mu.Lock()
			a.stats.authErrors++
			a.stats.mu.Unlock()

			a.logger.Warn("Invalid community string",
				zap.String("remote_addr", remoteAddr.String()),
				zap.String("received", request.Community),
				zap.String("expected", a.config.SNMPCommunity))

			// Send back authentication failure
			response := &SNMPMessage{
				Version:   request.Version,
				Community: request.Community,
				PDUType:   gosnmp.GetResponse,
				RequestID: request.RequestID,
				Variables: make([]gosnmp.SnmpPDU, 0),
				Error:     gosnmp.AuthorizationError,
				Index:     0,
			}

			responseBytes, err := EncodeMessage(response)
			if err == nil {
				a.logger.Debug("Sending auth failure response",
					zap.String("remote_addr", remoteAddr.String()),
					zap.Binary("data", responseBytes))
				if _, err := a.conn.WriteToUDP(responseBytes, remoteAddr); err != nil {
					a.logger.Error("Error sending auth failure response", zap.Error(err))
				}
			} else {
				a.logger.Error("Error encoding auth failure response", zap.Error(err))
			}
			continue
		}

		// Handle request with timeout
		go func() {
			done := make(chan struct{})
			go func() {
				a.processRequest(request, remoteAddr)
				close(done)
			}()

			select {
			case <-done:
				// Request completed normally
			case <-time.After(5 * time.Second):
				a.logger.Error("Request timeout",
					zap.String("remote_addr", remoteAddr.String()))
				a.stats.mu.Lock()
				a.stats.lastError = "Request timeout"
				a.stats.lastErrorTime = time.Now()
				a.stats.mu.Unlock()
			}
		}()
	}
}

func (a *SNMPAgent) processRequest(request *SNMPMessage, remoteAddr *net.UDPAddr) {
	start := time.Now()
	defer func() {
		if r := recover(); r != nil {
			a.logger.Error("Panic in processRequest",
				zap.String("remote_addr", remoteAddr.String()),
				zap.Any("panic", r))
			a.stats.mu.Lock()
			a.stats.lastError = fmt.Sprintf("Panic: %v", r)
			a.stats.lastErrorTime = time.Now()
			a.stats.mu.Unlock()
		}
		duration := time.Since(start)
		a.logger.Debug("Request processing completed",
			zap.Duration("duration", duration))

		// Update metrics
		a.mu.Lock()
		a.metrics.UpdateResourceMetrics(
			a.metrics.CPUUsage,
			a.metrics.MemoryUsage,
			4096,                          // Fixed buffer size from pool
			int64(len(request.Variables)), // Queue length is number of variables
			int64(runtime.NumGoroutine()), // Current goroutines
		)
		a.mu.Unlock()

		// Update performance metrics
		a.metrics.UpdatePerformanceMetrics(
			duration.Microseconds(), // Latency
			0,                       // Jitter (not tracked yet)
			0,                       // RTT (not tracked yet)
			0,                       // Packet loss (not tracked yet)
			0,                       // Reordering rate (not tracked yet)
		)
	}()

	a.logger.Debug("Processing SNMP request",
		zap.String("remote_addr", remoteAddr.String()),
		zap.Int("type", int(request.PDUType)),
		zap.Int("variables", len(request.Variables)))

	response := &SNMPMessage{
		Version:   request.Version,
		Community: request.Community,
		PDUType:   gosnmp.GetResponse,
		RequestID: request.RequestID,
		Variables: make([]gosnmp.SnmpPDU, 0, len(request.Variables)),
		Error:     gosnmp.NoError,
		Index:     0,
	}

	// Track successful requests
	defer func() {
		if response.Error == gosnmp.NoError {
			a.stats.mu.Lock()
			a.stats.successfulRequests++
			a.stats.mu.Unlock()
		}
	}()

	// Lock metrics while processing request
	a.mu.RLock()
	a.mibTree.UpdateMetrics(a.metrics)

	// Process each variable in the request
	for i, varBind := range request.Variables {
		oid := varBind.Name
		var result gosnmp.SnmpPDU

		a.logger.Debug("Processing OID",
			zap.String("oid", oid),
			zap.String("remote_addr", remoteAddr.String()))

		switch request.PDUType {
		case gosnmp.GetRequest:
			entry, err := a.mibTree.GetEntry(oid, request.Community)
			if err != nil {
				if mibErr, ok := err.(*MIBError); ok {
					switch mibErr.Code {
					case 1: // Invalid community
						response.Error = gosnmp.AuthorizationError
					case 2: // No access
						response.Error = gosnmp.NoAccess
					case 3: // Wrong type
						response.Error = gosnmp.WrongType
					case 6: // OID not found
						response.Error = gosnmp.NoSuchName
					default:
						response.Error = gosnmp.GenErr
					}
				} else {
					response.Error = gosnmp.GenErr
				}
				response.Index = i
				a.logger.Error("Failed to get OID",
					zap.String("oid", oid),
					zap.Error(err))
				break
			}

			var snmpType gosnmp.Asn1BER
			switch entry.Type {
			case "Counter64":
				snmpType = gosnmp.Counter64
			case "Gauge32":
				snmpType = gosnmp.Gauge32
			case "OCTET STRING":
				snmpType = gosnmp.OctetString
			default:
				snmpType = gosnmp.Integer
			}

			var value interface{}
			if entry.Type == "OCTET STRING" {
				value = entry.Value
			} else {
				value = entry.ValueToInt64(entry.Value)
			}

			result = gosnmp.SnmpPDU{
				Name:  oid,
				Type:  snmpType,
				Value: value,
			}
			a.logger.Debug("Found value for OID",
				zap.String("oid", oid),
				zap.Any("value", result.Value),
				zap.String("type", entry.Type))

		case gosnmp.GetNextRequest:
			entry, err := a.mibTree.GetNextEntry(oid, request.Community)
			if err != nil {
				if mibErr, ok := err.(*MIBError); ok {
					switch mibErr.Code {
					case 1: // Invalid community
						response.Error = gosnmp.AuthorizationError
					case 2: // No access
						response.Error = gosnmp.NoAccess
					case 3: // Wrong type
						response.Error = gosnmp.WrongType
					case 7: // No next OID
						response.Error = gosnmp.NoSuchName
					default:
						response.Error = gosnmp.GenErr
					}
				} else {
					response.Error = gosnmp.GenErr
				}
				response.Index = i
				a.logger.Error("Failed to get next OID",
					zap.String("oid", oid),
					zap.Error(err))
				break
			}

			var snmpType gosnmp.Asn1BER
			switch entry.Type {
			case "Counter64":
				snmpType = gosnmp.Counter64
			case "Gauge32":
				snmpType = gosnmp.Gauge32
			case "OCTET STRING":
				snmpType = gosnmp.OctetString
			default:
				snmpType = gosnmp.Integer
			}

			var value interface{}
			if entry.Type == "OCTET STRING" {
				value = entry.Value
			} else {
				value = entry.ValueToInt64(entry.Value)
			}

			result = gosnmp.SnmpPDU{
				Name:  entry.OID,
				Type:  snmpType,
				Value: value,
			}
			a.logger.Debug("Found next OID",
				zap.String("after", oid),
				zap.String("next_oid", entry.OID),
				zap.Any("value", result.Value),
				zap.String("type", entry.Type))

		default:
			response.Error = gosnmp.GenErr
			response.Index = i
			a.logger.Error("Unsupported PDU type",
				zap.Int("type", int(request.PDUType)))
			break
		}

		if response.Error == gosnmp.NoError {
			response.Variables = append(response.Variables, result)
		}
	}

	a.mu.RUnlock()

	// Encode and send response
	responseBytes, err := EncodeMessage(response)
	if err != nil {
		a.logger.Error("Error encoding SNMP response", zap.Error(err))
		return
	}

	a.logger.Debug("Sending response",
		zap.String("remote_addr", remoteAddr.String()),
		zap.Binary("data", responseBytes))

	if _, err := a.conn.WriteToUDP(responseBytes, remoteAddr); err != nil {
		a.logger.Error("Failed to send SNMP response", zap.Error(err))
		return
	}

	// Log response summary
	a.logger.Info("Sent SNMP response",
		zap.String("remote_addr", remoteAddr.String()),
		zap.Int("error", int(response.Error)),
		zap.Int("index", response.Index),
		zap.Int("variables", len(response.Variables)))
}
