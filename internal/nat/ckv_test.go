
package nat

import (
	"net"
	"testing"
)

func TestChecksumVerifierOnSyn(t *testing.T) {
	pkt := buildSynPacket(net.ParseIP("10.77.0.2"), net.ParseIP("10.77.0.1"), 12345, 8080)
	if !tcpChecksumValidBytes(pkt) {
		t.Fatal("verifier rejects our own SYN — verifier bug")
	}
	t.Log("verifier accepts buildSynPacket output")
}
