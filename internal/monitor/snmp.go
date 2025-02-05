package monitor

import (
	"fmt"
	"net"
	"sync"
	"time"

	"github.com/gosnmp/gosnmp"
)

// SNMPAgent handles SNMP monitoring
type SNMPAgent struct {
	config    *Config
	metrics   *Metrics
	mibTree   *MIBTree
	conn      *net.UDPConn
	startTime time.Time
	mu        sync.RWMutex
}

// NewSNMPAgent creates a new SNMP agent
func NewSNMPAgent(cfg *Config, metrics *Metrics) (*SNMPAgent, error) {
	agent := &SNMPAgent{
		config:    cfg,
		metrics:   metrics,
		startTime: time.Now(),
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

	// Start handling requests
	go a.handleRequests()

	return nil
}

// Stop shuts down the SNMP agent
func (a *SNMPAgent) Stop() {
	if a.conn != nil {
		a.conn.Close()
	}
}

func (a *SNMPAgent) handleRequests() {
	buffer := make([]byte, 2048)
	for {
		n, remoteAddr, err := a.conn.ReadFromUDP(buffer)
		if err != nil {
			if opErr, ok := err.(*net.OpError); ok && opErr.Err.Error() == "use of closed network connection" {
				return
			}
			fmt.Printf("Error reading UDP: %v\n", err)
			continue
		}

		// Parse incoming SNMP packet
		request, err := DecodeMessage(buffer[:n])
		if err != nil {
			fmt.Printf("Error decoding SNMP packet: %v\n", err)
			continue
		}

		// Verify community string
		if request.Community != a.config.SNMPCommunity {
			fmt.Printf("Invalid community string from %v\n", remoteAddr)
			continue
		}

		// Handle request in a goroutine
		go a.processRequest(request, remoteAddr)
	}
}

func (a *SNMPAgent) processRequest(request *SNMPMessage, remoteAddr *net.UDPAddr) {
	response := &SNMPMessage{
		Version:   request.Version,
		Community: request.Community,
		PDUType:   gosnmp.GetResponse,
		RequestID: request.RequestID,
		Variables: make([]gosnmp.SnmpPDU, 0, len(request.Variables)),
		Error:     gosnmp.NoError,
		Index:     0,
	}

	// Lock metrics while processing request
	a.mu.RLock()
	a.mibTree.UpdateMetrics(a.metrics)

	// Process each variable in the request
	for _, varBind := range request.Variables {
		oid := varBind.Name
		var result gosnmp.SnmpPDU

		switch request.PDUType {
		case gosnmp.GetRequest:
			if entry, ok := a.mibTree.GetEntry(oid); ok {
				result = gosnmp.SnmpPDU{
					Name:  oid,
					Type:  gosnmp.Counter64,
					Value: entry.ValueToInt64(entry.Value),
				}
			} else {
				response.Error = gosnmp.NoSuchName
				response.Index = 0
			}

		case gosnmp.GetNextRequest:
			if entry, ok := a.mibTree.GetNextEntry(oid); ok {
				result = gosnmp.SnmpPDU{
					Name:  entry.OID,
					Type:  gosnmp.Counter64,
					Value: entry.ValueToInt64(entry.Value),
				}
			} else {
				response.Error = gosnmp.NoSuchName
				response.Index = 0
			}

		default:
			response.Error = gosnmp.GenErr
			response.Index = 0
		}

		if response.Error == gosnmp.NoError {
			response.Variables = append(response.Variables, result)
		}
	}

	a.mu.RUnlock()

	// Encode and send response
	responseBytes, err := EncodeMessage(response)
	if err != nil {
		fmt.Printf("Error encoding SNMP response: %v\n", err)
		return
	}

	if _, err := a.conn.WriteToUDP(responseBytes, remoteAddr); err != nil {
		fmt.Printf("Error sending SNMP response: %v\n", err)
	}
}
