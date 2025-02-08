package monitor

import (
	"fmt"
	"sort"
	"strings"
)

// OID constants for our enterprise MIB
const (
	// Base OID for our enterprise MIB
	baseOID = ".1.3.6.1.4.1.54321"

	// Performance Metrics (1.x)
	bytesInOID     = baseOID + ".1.1" // Counter64: Total bytes received
	bytesOutOID    = baseOID + ".1.2" // Counter64: Total bytes sent
	activeConnsOID = baseOID + ".1.7" // Gauge32: Current active connections
	cpuUsageOID    = baseOID + ".1.8" // Gauge32: CPU usage percentage
	memoryUsageOID = baseOID + ".1.9" // Gauge32: Memory usage in MB

	// Status Metrics (2.x)
	tunnelStatusOID = baseOID + ".2.1" // INTEGER: 0=down, 1=up
	lastErrorOID    = baseOID + ".2.2" // OCTET STRING: Last error message
	startTimeOID    = baseOID + ".2.3" // Counter64: Start time (unix timestamp)

	// Configuration (3.x)
	maxConnsOID = baseOID + ".3.1" // INTEGER: Maximum allowed connections
	rateUpOID   = baseOID + ".3.2" // Gauge32: Upload rate limit (kbps)
	rateDownOID = baseOID + ".3.3" // Gauge32: Download rate limit (kbps)
)

// SNMP community strings
const (
	defaultCommunity = "public"
	readCommunity    = defaultCommunity
	writeCommunity   = "private"
)

// MIBError represents MIB-specific errors
type MIBError struct {
	Code    int
	Message string
}

func (e *MIBError) Error() string {
	return fmt.Sprintf("MIB Error %d: %s", e.Code, e.Message)
}

// Common MIB errors
var (
	ErrInvalidCommunity = &MIBError{Code: 1, Message: "Invalid community string"}
	ErrNoAccess         = &MIBError{Code: 2, Message: "No access to this OID"}
	ErrWrongType        = &MIBError{Code: 3, Message: "Wrong value type"}
)

// MIBEntry represents a single entry in our MIB
type MIBEntry struct {
	OID          string
	Name         string
	Description  string
	Type         string
	Value        interface{}
	ValueToInt64 func(interface{}) int64
	Access       string       // "read-only", "read-write", or "write-only"
	Validate     func() error // Custom validation function
}

// MIBTree represents our complete MIB structure
type MIBTree struct {
	entries map[string]MIBEntry
}

// NewMIBTree creates a new MIB tree with our metric entries
func NewMIBTree(metrics *Metrics) *MIBTree {
	tree := &MIBTree{
		entries: make(map[string]MIBEntry),
	}

	// Performance Metrics
	tree.addCounter64(bytesInOID, "bytesIn", "Total bytes received", metrics.BytesIn, "read-only")
	tree.addCounter64(bytesOutOID, "bytesOut", "Total bytes sent", metrics.BytesOut, "read-only")
	tree.addGauge32(activeConnsOID, "activeConnections", "Current active connections", metrics.Connections, "read-only")
	tree.addGauge32(cpuUsageOID, "cpuUsage", "CPU usage percentage", int32(metrics.CPUUsage), "read-only")
	tree.addGauge32(memoryUsageOID, "memoryUsage", "Memory usage in MB", int32(metrics.MemoryUsage/1024/1024), "read-only")

	// Status Metrics
	tree.addInteger(tunnelStatusOID, "tunnelStatus", "Tunnel operational status", 1, "read-only")
	tree.addString(lastErrorOID, "lastError", "Last error message", "", "read-only")
	tree.addCounter64(startTimeOID, "startTime", "Service start time", metrics.StartTime.Unix(), "read-only")

	// Configuration
	tree.addInteger(maxConnsOID, "maxConnections", "Maximum allowed connections", 10, "read-write")
	tree.addGauge32(rateUpOID, "uploadRateLimit", "Upload rate limit in kbps", 10240, "read-write")
	tree.addGauge32(rateDownOID, "downloadRateLimit", "Download rate limit in kbps", 10240, "read-write")

	return tree
}

// Helper methods for adding metrics
func (t *MIBTree) addCounter64(oid, name, desc string, value int64, access string) {
	t.entries[oid] = MIBEntry{
		OID:         oid,
		Name:        name,
		Description: desc,
		Type:        "Counter64",
		Value:       value,
		Access:      access,
		ValueToInt64: func(v interface{}) int64 {
			if val, ok := v.(int64); ok {
				return val
			}
			return 0
		},
		Validate: func() error {
			if value < 0 {
				return &MIBError{Code: 4, Message: "Counter64 cannot be negative"}
			}
			return nil
		},
	}
}

func (t *MIBTree) addGauge32(oid, name, desc string, value int32, access string) {
	t.entries[oid] = MIBEntry{
		OID:         oid,
		Name:        name,
		Description: desc,
		Type:        "Gauge32",
		Value:       value,
		Access:      access,
		ValueToInt64: func(v interface{}) int64 {
			if val, ok := v.(int32); ok {
				return int64(val)
			}
			return 0
		},
		Validate: func() error {
			if value < 0 {
				return &MIBError{Code: 5, Message: "Gauge32 cannot be negative"}
			}
			return nil
		},
	}
}

func (t *MIBTree) addInteger(oid, name, desc string, value int, access string) {
	t.entries[oid] = MIBEntry{
		OID:         oid,
		Name:        name,
		Description: desc,
		Type:        "INTEGER",
		Value:       value,
		Access:      access,
		ValueToInt64: func(v interface{}) int64 {
			if val, ok := v.(int); ok {
				return int64(val)
			}
			return 0
		},
		Validate: func() error {
			return nil // No specific validation for INTEGER
		},
	}
}

func (t *MIBTree) addString(oid, name, desc string, value string, access string) {
	t.entries[oid] = MIBEntry{
		OID:         oid,
		Name:        name,
		Description: desc,
		Type:        "OCTET STRING",
		Value:       value,
		Access:      access,
		ValueToInt64: func(v interface{}) int64 {
			return 0 // Strings don't convert to int64
		},
		Validate: func() error {
			return nil // No specific validation for strings
		},
	}
}

// UpdateMetrics updates all metric values in the MIB tree
func (t *MIBTree) UpdateMetrics(metrics *Metrics) {
	// Create new entries with updated values
	newEntries := make(map[string]MIBEntry)
	for oid, entry := range t.entries {
		newEntry := entry
		switch oid {
		case bytesInOID:
			newEntry.Value = metrics.BytesIn
		case bytesOutOID:
			newEntry.Value = metrics.BytesOut
		case activeConnsOID:
			newEntry.Value = metrics.Connections
		case cpuUsageOID:
			newEntry.Value = int32(metrics.CPUUsage)
		case memoryUsageOID:
			newEntry.Value = int32(metrics.MemoryUsage / 1024 / 1024) // Convert to MB
		case tunnelStatusOID:
			// Connected if last connect time is after last disconnect time
			if metrics.ConnectTime > metrics.DisconnectTime {
				newEntry.Value = 1
			} else {
				newEntry.Value = 0
			}
		case lastErrorOID:
			newEntry.Value = metrics.LastError
		case startTimeOID:
			newEntry.Value = metrics.StartTime.Unix()
		}
		newEntries[oid] = newEntry
	}
	t.entries = newEntries
}

// GetEntry retrieves a MIB entry by its OID and community string
func (t *MIBTree) GetEntry(oid string, community string) (MIBEntry, error) {
	// Validate community string
	if community == "" || (community != readCommunity && community != writeCommunity) {
		return MIBEntry{}, ErrInvalidCommunity
	}

	entry, ok := t.entries[oid]
	if !ok {
		return MIBEntry{}, &MIBError{Code: 6, Message: "OID not found"}
	}

	// Check access rights
	if community == readCommunity && entry.Access == "write-only" {
		return MIBEntry{}, ErrNoAccess
	}

	return entry, nil
}

// GetNextEntry retrieves the next MIB entry after the given OID with community string validation
func (t *MIBTree) GetNextEntry(oid string, community string) (MIBEntry, error) {
	// Validate community string
	if community == "" || (community != readCommunity && community != writeCommunity) {
		return MIBEntry{}, ErrInvalidCommunity
	}

	// Get all OIDs and sort them
	var oids []string
	for k, entry := range t.entries {
		// Skip write-only entries for read community
		if community == readCommunity && entry.Access == "write-only" {
			continue
		}
		oids = append(oids, k)
	}
	sort.Strings(oids)

	// Find the next OID
	for i, currentOID := range oids {
		if currentOID > oid && i < len(oids) {
			return t.entries[oids[i]], nil
		}
	}
	return MIBEntry{}, &MIBError{Code: 7, Message: "No next OID found"}
}

// String returns a string representation of the MIB tree
func (t *MIBTree) String() string {
	var sb strings.Builder
	sb.WriteString("MIB Tree:\n")
	for _, entry := range t.entries {
		sb.WriteString(fmt.Sprintf("OID: %s\n", entry.OID))
		sb.WriteString(fmt.Sprintf("  Name: %s\n", entry.Name))
		sb.WriteString(fmt.Sprintf("  Description: %s\n", entry.Description))
		sb.WriteString(fmt.Sprintf("  Type: %s\n", entry.Type))
		sb.WriteString(fmt.Sprintf("  Value: %v\n", entry.Value))
		sb.WriteString("\n")
	}
	return sb.String()
}
