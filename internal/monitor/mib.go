package monitor

import (
	"fmt"
	"sort"
	"strings"
)

// OID constants for our enterprise MIB
const (
	// Base OID for our enterprise MIB (example: 1.3.6.1.4.1.XXXXX)
	// Replace XXXXX with an actual enterprise number in production
	baseOID = ".1.3.6.1.4.1.54321"

	// Metric OIDs
	bytesInOID     = baseOID + ".1.1"
	bytesOutOID    = baseOID + ".1.2"
	packetsInOID   = baseOID + ".1.3"
	packetsOutOID  = baseOID + ".1.4"
	errorsOID      = baseOID + ".1.5"
	uptimeOID      = baseOID + ".1.6"
	connectionsOID = baseOID + ".1.7"
)

// MIBEntry represents a single entry in our MIB
type MIBEntry struct {
	OID          string
	Name         string
	Description  string
	Type         string
	Value        interface{}
	ValueToInt64 func(interface{}) int64
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

	// Define MIB entries
	tree.entries[bytesInOID] = MIBEntry{
		OID:         bytesInOID,
		Name:        "bytesIn",
		Description: "Total bytes received",
		Type:        "Counter64",
		Value:       metrics.BytesIn,
		ValueToInt64: func(v interface{}) int64 {
			if val, ok := v.(int64); ok {
				return val
			}
			return 0
		},
	}

	tree.entries[bytesOutOID] = MIBEntry{
		OID:         bytesOutOID,
		Name:        "bytesOut",
		Description: "Total bytes sent",
		Type:        "Counter64",
		Value:       metrics.BytesOut,
		ValueToInt64: func(v interface{}) int64 {
			if val, ok := v.(int64); ok {
				return val
			}
			return 0
		},
	}

	tree.entries[packetsInOID] = MIBEntry{
		OID:         packetsInOID,
		Name:        "packetsIn",
		Description: "Total packets received",
		Type:        "Counter64",
		Value:       metrics.PacketsIn,
		ValueToInt64: func(v interface{}) int64 {
			if val, ok := v.(int64); ok {
				return val
			}
			return 0
		},
	}

	tree.entries[packetsOutOID] = MIBEntry{
		OID:         packetsOutOID,
		Name:        "packetsOut",
		Description: "Total packets sent",
		Type:        "Counter64",
		Value:       metrics.PacketsOut,
		ValueToInt64: func(v interface{}) int64 {
			if val, ok := v.(int64); ok {
				return val
			}
			return 0
		},
	}

	tree.entries[errorsOID] = MIBEntry{
		OID:         errorsOID,
		Name:        "errors",
		Description: "Total error count",
		Type:        "Counter64",
		Value:       metrics.Errors,
		ValueToInt64: func(v interface{}) int64 {
			if val, ok := v.(int64); ok {
				return val
			}
			return 0
		},
	}

	tree.entries[uptimeOID] = MIBEntry{
		OID:         uptimeOID,
		Name:        "uptime",
		Description: "System uptime in seconds",
		Type:        "Counter64",
		Value:       metrics.Uptime,
		ValueToInt64: func(v interface{}) int64 {
			if val, ok := v.(int64); ok {
				return val
			}
			return 0
		},
	}

	tree.entries[connectionsOID] = MIBEntry{
		OID:         connectionsOID,
		Name:        "connections",
		Description: "Current number of connections",
		Type:        "Gauge32",
		Value:       metrics.Connections,
		ValueToInt64: func(v interface{}) int64 {
			if val, ok := v.(int); ok {
				return int64(val)
			}
			return 0
		},
	}

	return tree
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
		case packetsInOID:
			newEntry.Value = metrics.PacketsIn
		case packetsOutOID:
			newEntry.Value = metrics.PacketsOut
		case errorsOID:
			newEntry.Value = metrics.Errors
		case uptimeOID:
			newEntry.Value = metrics.Uptime
		case connectionsOID:
			newEntry.Value = metrics.Connections
		}
		newEntries[oid] = newEntry
	}
	t.entries = newEntries
}

// GetEntry retrieves a MIB entry by its OID
func (t *MIBTree) GetEntry(oid string) (MIBEntry, bool) {
	entry, ok := t.entries[oid]
	return entry, ok
}

// GetNextEntry retrieves the next MIB entry after the given OID
func (t *MIBTree) GetNextEntry(oid string) (MIBEntry, bool) {
	// Get all OIDs and sort them
	var oids []string
	for k := range t.entries {
		oids = append(oids, k)
	}
	sort.Strings(oids)

	// Find the next OID
	for i, currentOID := range oids {
		if currentOID > oid && i < len(oids) {
			return t.entries[oids[i]], true
		}
	}
	return MIBEntry{}, false
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
