package nat

import (
	"encoding/binary"
	"errors"
	"net"
)

const (
	ipv4HeaderMinLen = 20
	ipv4ProtoTCP     = 6
	tcpHeaderMinLen  = 20
)

// parseIPv4TCP validates a raw IPv4 TCP packet and extracts addresses,
// ports, and the TCP header offset within the packet.
func parseIPv4TCP(pkt []byte) (srcIP net.IP, dstIP net.IP, srcPort, dstPort, tcpOff int, err error) {
	if len(pkt) < ipv4HeaderMinLen {
		return nil, nil, 0, 0, 0, ErrPacketTooShort
	}
	if pkt[0]>>4 != 4 {
		return nil, nil, 0, 0, 0, ErrNotIPv4
	}
	ihl := int(pkt[0]&0x0F) * 4
	if ihl < ipv4HeaderMinLen || len(pkt) < ihl {
		return nil, nil, 0, 0, 0, ErrPacketTooShort
	}
	if pkt[9] != ipv4ProtoTCP {
		return nil, nil, 0, 0, 0, ErrNotTCP
	}
	totalLen := int(binary.BigEndian.Uint16(pkt[2:4]))
	if totalLen > len(pkt) {
		return nil, nil, 0, 0, 0, ErrPacketTooShort
	}
	tcpOff = ihl
	if len(pkt) < tcpOff+tcpHeaderMinLen {
		return nil, nil, 0, 0, 0, ErrPacketTooShort
	}

	srcIP = net.IP(append([]byte(nil), pkt[12:16]...))
	dstIP = net.IP(append([]byte(nil), pkt[16:20]...))
	srcPort = int(binary.BigEndian.Uint16(pkt[tcpOff : tcpOff+2]))
	dstPort = int(binary.BigEndian.Uint16(pkt[tcpOff+2 : tcpOff+4]))
	return srcIP, dstIP, srcPort, dstPort, tcpOff, nil
}

// tcpFlags extracts SYN/ACK/FIN/RST from the TCP header at tcpOff.
func tcpFlags(pkt []byte, tcpOff int) (syn, ack, fin, rst bool) {
	f := pkt[tcpOff+13]
	return f&0x02 != 0, f&0x10 != 0, f&0x01 != 0, f&0x04 != 0
}

// rewriteIPv4Src replaces the IPv4 source address, applying the
// incremental IP header checksum update (RFC 1624).
func rewriteIPv4Src(pkt []byte, oldIP, newIP [4]byte) {
	pkt[12], pkt[13], pkt[14], pkt[15] = newIP[0], newIP[1], newIP[2], newIP[3]
	// Incremental checksum: treat each 16-bit pair as one word.
	for i := 0; i < 4; i += 2 {
		oldW := uint16(oldIP[i])<<8 | uint16(oldIP[i+1])
		newW := uint16(newIP[i])<<8 | uint16(newIP[i+1])
		oldCk := binary.BigEndian.Uint16(pkt[10:12])
		binary.BigEndian.PutUint16(pkt[10:], checksumUpdate(oldCk, oldW, newW))
	}
}

// rewriteIPv4Dst replaces the IPv4 destination address, applying the
// incremental IP header checksum update.
func rewriteIPv4Dst(pkt []byte, oldIP [4]byte, newIP net.IP) {
	for i := 0; i < 4; i += 2 {
		oldW := uint16(oldIP[i])<<8 | uint16(oldIP[i+1])
		newW := uint16(newIP[i])<<8 | uint16(newIP[i+1])
		oldCk := binary.BigEndian.Uint16(pkt[10:12])
		binary.BigEndian.PutUint16(pkt[10:], checksumUpdate(oldCk, oldW, newW))
	}
	copy(pkt[16:20], newIP.To4())
}

// rewriteTCPPort replaces the TCP source port (portOffset 0) or
// destination port (portOffset 2), applying the incremental TCP checksum
// update. Full recomputation is performed afterwards by the caller, so
// this only rewrites bytes and updates the IP checksum-independent copy.
func rewriteTCPPort(pkt []byte, tcpOff, portOffset int, oldPort, newPort uint16) {
	at := tcpOff + portOffset
	binary.BigEndian.PutUint16(pkt[at:], newPort)
	// Incremental TCP checksum update over the port word.
	ckAt := tcpOff + 16
	oldCk := binary.BigEndian.Uint16(pkt[ckAt : ckAt+2])
	binary.BigEndian.PutUint16(pkt[ckAt:], checksumUpdate(oldCk, oldPort, newPort))
}

// recomputeChecksums recomputes both IP and TCP checksums from scratch
// after rewrites. This is belt-and-braces over incremental updates: the
// full recompute is O(packet length) but happens per packet either way,
// and correctness beats micro-optimization on the NAT path (per the
// repo's data-path-correctness invariant).
func recomputeIPv4Checksum(pkt []byte) {
	ihl := int(pkt[0]&0x0F) * 4
	pkt[10], pkt[11] = 0, 0
	sum := uint32(0)
	for i := 0; i < ihl; i += 2 {
		sum += uint32(pkt[i])<<8 | uint32(pkt[i+1])
	}
	for sum>>16 != 0 {
		sum = (sum >> 16) + (sum & 0xFFFF)
	}
	ck := ^uint16(sum)
	pkt[10] = byte(ck >> 8)
	pkt[11] = byte(ck)
}

// recomputeTCPChecksum recomputes the TCP checksum from scratch over the
// pseudo-header and segment, storing it at the header's checksum field.
func recomputeTCPChecksum(pkt []byte, tcpOff int) {
	// The TCP checksum field must be zero during computation.
	ckAt := tcpOff + 16
	pkt[ckAt], pkt[ckAt+1] = 0, 0
	ck := tcpChecksum(
		[4]byte{pkt[12], pkt[13], pkt[14], pkt[15]},
		[4]byte{pkt[16], pkt[17], pkt[18], pkt[19]},
		pkt[tcpOff:],
		16,
	)
	pkt[ckAt] = byte(ck >> 8)
	pkt[ckAt+1] = byte(ck)
}

// errPacketTruncated indicates a packet shorter than its headers claim.
var errPacketTruncated = errors.New("packet truncated")
