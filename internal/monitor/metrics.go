package monitor

// Metrics holds monitoring metrics
type Metrics struct {
	BytesIn     int64
	BytesOut    int64
	PacketsIn   int64
	PacketsOut  int64
	Errors      int64
	Uptime      int64
	Connections int
}

// NewMetrics creates a new metrics instance
func NewMetrics() *Metrics {
	return &Metrics{}
}

// Reset resets all metrics to zero
func (m *Metrics) Reset() {
	m.BytesIn = 0
	m.BytesOut = 0
	m.PacketsIn = 0
	m.PacketsOut = 0
	m.Errors = 0
	m.Uptime = 0
	m.Connections = 0
}

// Clone creates a copy of the metrics
func (m *Metrics) Clone() *Metrics {
	return &Metrics{
		BytesIn:     m.BytesIn,
		BytesOut:    m.BytesOut,
		PacketsIn:   m.PacketsIn,
		PacketsOut:  m.PacketsOut,
		Errors:      m.Errors,
		Uptime:      m.Uptime,
		Connections: m.Connections,
	}
}
