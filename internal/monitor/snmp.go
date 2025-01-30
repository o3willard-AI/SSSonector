package monitor

import (
	"time"

	"github.com/gosnmp/gosnmp"
	"github.com/o3willard-AI/SSSonector/internal/config"
	"go.uber.org/zap"
)

// SNMPAgent handles SNMP monitoring
type SNMPAgent struct {
	logger    *zap.Logger
	config    *config.MonitorConfig
	snmp      *gosnmp.GoSNMP
	metrics   *Metrics
	startTime time.Time
}

// NewSNMPAgent creates a new SNMP agent
func NewSNMPAgent(logger *zap.Logger, cfg *config.MonitorConfig) (*SNMPAgent, error) {
	agent := &SNMPAgent{
		logger:    logger,
		config:    cfg,
		startTime: time.Now(),
		metrics:   NewMetrics(),
	}

	if err := agent.initSNMP(); err != nil {
		return nil, err
	}

	return agent, nil
}

// initSNMP initializes the SNMP server
func (a *SNMPAgent) initSNMP() error {
	a.snmp = &gosnmp.GoSNMP{
		Target:             a.config.SNMPAddress,
		Port:               uint16(a.config.SNMPPort),
		Community:          a.config.SNMPCommunity,
		Version:            gosnmp.Version2c,
		Timeout:            time.Duration(2) * time.Second,
		SecurityModel:      gosnmp.UserSecurityModel,
		MsgFlags:           gosnmp.NoAuthNoPriv,
		SecurityParameters: &gosnmp.UsmSecurityParameters{},
	}

	return a.snmp.Connect()
}

// Start starts the SNMP agent
func (a *SNMPAgent) Start() error {
	if !a.config.SNMPEnabled {
		return nil
	}

	a.logger.Info("Starting SNMP agent",
		zap.String("address", a.config.SNMPAddress),
		zap.Int("port", a.config.SNMPPort),
	)

	return nil
}

// Stop stops the SNMP agent
func (a *SNMPAgent) Stop() error {
	if a.snmp != nil {
		return a.snmp.Conn.Close()
	}
	return nil
}

// UpdateMetrics updates the SNMP metrics
func (a *SNMPAgent) UpdateMetrics(bytesReceived, bytesSent uint64, connections int) {
	if !a.config.SNMPEnabled {
		return
	}

	a.metrics.BytesReceived = bytesReceived
	a.metrics.BytesSent = bytesSent
	a.metrics.Connections = connections
	a.metrics.Uptime = time.Since(a.startTime).Seconds()
}

// Metrics holds monitoring metrics
type Metrics struct {
	BytesReceived uint64
	BytesSent     uint64
	Connections   int
	Uptime        float64
}

// NewMetrics creates a new metrics instance
func NewMetrics() *Metrics {
	return &Metrics{}
}
