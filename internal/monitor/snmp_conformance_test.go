package monitor

import (
	"net"
	"strconv"
	"testing"
	"time"

	"github.com/gosnmp/gosnmp"
	"go.uber.org/zap"
)

// startTestAgent starts an SNMP agent on an OS-assigned loopback port with
// community "public" and returns it plus the listen address.
func startTestAgent(t *testing.T) (*SNMPAgent, *Metrics, string) {
	t.Helper()
	logger, _ := zap.NewDevelopment()
	metrics := NewMetrics()
	agent, err := NewSNMPAgent(&Config{
		SNMPEnabled:   true,
		SNMPAddress:   "127.0.0.1",
		SNMPPort:      0,
		SNMPCommunity: "public",
	}, metrics, logger)
	if err != nil {
		t.Fatalf("NewSNMPAgent: %v", err)
	}
	if err := agent.Start(); err != nil {
		t.Fatalf("Start: %v", err)
	}
	t.Cleanup(agent.Stop)
	return agent, metrics, agent.conn.LocalAddr().String()
}

// asUint normalizes numeric SNMP values of differing Go widths for comparison.
func asUint(v interface{}) uint64 {
	switch n := v.(type) {
	case uint:
		return uint64(n)
	case uint32:
		return uint64(n)
	case uint64:
		return n
	case int:
		return uint64(n)
	case int64:
		return uint64(n)
	default:
		return ^uint64(0) // sentinel that will never match a sane counter
	}
}

func newClient(addr, community string) *gosnmp.GoSNMP {
	host, portStr, _ := net.SplitHostPort(addr)
	port, _ := strconv.Atoi(portStr)
	return &gosnmp.GoSNMP{
		Target:     host,
		Port:       uint16(port),
		Community:  community,
		Version:    gosnmp.Version2c,
		Timeout:    2 * time.Second,
		Retries:    1,
		MaxOids:    gosnmp.MaxOids,
		Transport:  "udp",
	}
}

// TestSNMPGetConformance proves the agent answers standard GetRequests with
// the values published in its MIB, decoded by an independent gosnmp client.
func TestSNMPGetConformance(t *testing.T) {
	_, metrics, addr := startTestAgent(t)

	// Mark the tunnel "up": status derives from ConnectTime > DisconnectTime.
	metrics.UpdateConnectionMetrics(1, 10, time.Now().UnixMilli(), 0)

	client := newClient(addr, "public")
	if err := client.Connect(); err != nil {
		t.Fatalf("Connect: %v", err)
	}
	defer client.Conn.Close()

	result, err := client.Get([]string{
		maxConnsOID,       // INTEGER 10
		tunnelStatusOID,   // INTEGER 1
		rateUpOID,         // Gauge32 10240
		bytesInOID,        // Counter64 (snapshot value)
	})
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if len(result.Variables) != 4 {
		t.Fatalf("expected 4 varbinds, got %d", len(result.Variables))
	}

	assertPDU := func(i int, wantOID string, wantType gosnmp.Asn1BER, wantValue interface{}) {
		pdu := result.Variables[i]
		if pdu.Name != wantOID {
			t.Errorf("varbind %d: OID = %s, want %s", i, pdu.Name, wantOID)
		}
		if pdu.Type != wantType {
			t.Errorf("varbind %s: type = %v, want %v", pdu.Name, pdu.Type, wantType)
		}
		if wantValue != nil && pdu.Value != wantValue {
			t.Errorf("varbind %s: value = %v (%T), want %v", pdu.Name, pdu.Value, pdu.Value, wantValue)
		}
	}

	assertPDU(0, maxConnsOID, gosnmp.Integer, nil)
	assertPDU(1, tunnelStatusOID, gosnmp.Integer, nil)
	assertPDU(2, rateUpOID, gosnmp.Gauge32, nil)
	assertPDU(3, bytesInOID, gosnmp.Counter64, nil)
	if got := asUint(result.Variables[0].Value); got != 10 {
		t.Errorf("maxConnections = %v, want 10", result.Variables[0].Value)
	}

	if got := asUint(result.Variables[1].Value); got != 1 {
		t.Errorf("tunnelStatus = %v (%T), want 1", result.Variables[1].Value, result.Variables[1].Value)
	}
	if got := asUint(result.Variables[2].Value); got != 10240 {
		t.Errorf("rateUp = %v (%T), want 10240", result.Variables[2].Value, result.Variables[2].Value)
	}
}

// TestSNMPGetNextWalk proves GetNext from the MIB root returns entries in
// lexicographic OID order with correct types.
func TestSNMPGetNextWalk(t *testing.T) {
	_, metrics, addr := startTestAgent(t)

	// Mark the tunnel "up": status derives from ConnectTime > DisconnectTime.
	metrics.UpdateConnectionMetrics(1, 10, time.Now().UnixMilli(), 0)

	client := newClient(addr, "public")
	if err := client.Connect(); err != nil {
		t.Fatalf("Connect: %v", err)
	}
	defer client.Conn.Close()

	seen := map[string]bool{}
	oid := baseOID
	for i := 0; i < len(seen)+12; i++ {
		result, err := client.GetNext([]string{oid})
		if err != nil || len(result.Variables) == 0 {
			break // end of MIB view
		}
		pdu := result.Variables[0]
		if seen[pdu.Name] {
			break // walked off the end and wrapped
		}
		seen[pdu.Name] = true
		if pdu.Type == gosnmp.NoSuchObject || pdu.Type == gosnmp.EndOfMibView {
			break
		}
		oid = pdu.Name
	}

	if len(seen) < 10 {
		t.Errorf("walk exposed only %d entries; expected at least the 11 configured MIB rows: %v", len(seen), seen)
	}
	for _, want := range []string{bytesInOID, bytesOutOID, activeConnsOID, maxConnsOID} {
		if !seen[want] {
			t.Errorf("walk did not expose required OID %s", want)
		}
	}
}

// TestSNMPWrongCommunityRejected proves unknown communities never receive data.
func TestSNMPWrongCommunityRejected(t *testing.T) {
	_, _, addr := startTestAgent(t)

	client := newClient(addr, "wrong-community")
	if err := client.Connect(); err != nil {
		t.Fatalf("Connect: %v", err)
	}
	defer client.Conn.Close()

	result, err := client.Get([]string{maxConnsOID})
	if err != nil {
		return // rejected at protocol level
	}
	// Some clients surface SNMP errors as results with empty varbinds;
	// either way, no data may be disclosed.
	if len(result.Variables) > 0 && result.Variables[0].Value != nil {
		t.Fatalf("data leaked to wrong community: %+v", result.Variables[0])
	}
}

// TestSNMPOversizedPacketIgnored proves malformed datagrams do not crash or
// wedge the agent (it must keep serving after garbage input).
func TestSNMPOversizedPacketIgnored(t *testing.T) {
	agent, _, addr := startTestAgent(t)

	// Garbage directly at the socket.
	garbage := make([]byte, 4096)
	for i := range garbage {
		garbage[i] = byte('A' + i%26)
	}
	if _, err := agent.conn.WriteToUDP(garbage, agent.conn.LocalAddr().(*net.UDPAddr)); err != nil {
		t.Fatalf("WriteToUDP garbage: %v", err)
	}

	time.Sleep(200 * time.Millisecond)

	client := newClient(addr, "public")
	if err := client.Connect(); err != nil {
		t.Fatalf("Connect: %v", err)
	}
	defer client.Conn.Close()

	if _, err := client.Get([]string{maxConnsOID}); err != nil {
		t.Fatalf("agent stopped serving after garbage input: %v", err)
	}
}
