package nat

import "encoding/binary"

// checksumUpdate performs RFC 1624 incremental checksum update:
// HC' = ~(~HC + ~m + m') with one's-complement arithmetic.
// oldW/newW are the 16-bit big-endian words being replaced.
func checksumUpdate(oldChecksum, oldW, newW uint16) uint16 {
	sum := uint32(^oldChecksum) + uint32(^oldW) + uint32(newW)
	for sum>>16 != 0 {
		sum = (sum >> 16) + (sum & 0xFFFF)
	}
	return ^uint16(sum)
}

// tcpChecksum computes the TCP checksum over the pseudo-header (src, dst,
// zero, protocol, tcpLen) plus the TCP segment. checksumOffset is where
// the checksum field lives in the segment; the field is treated as zero.
func tcpChecksum(src, dst [4]byte, tcpSegment []byte, checksumOffset int) uint16 {
	const protocolTCP = 6
	sum := uint32(0)
	sum += uint32(src[0])<<8 | uint32(src[1])
	sum += uint32(src[2])<<8 | uint32(src[3])
	sum += uint32(dst[0])<<8 | uint32(dst[1])
	sum += uint32(dst[2])<<8 | uint32(dst[3])
	sum += protocolTCP
	tcpLen := len(tcpSegment)
	sum += uint32(tcpLen)

	for i := 0; i < tcpLen; i += 2 {
		if i == checksumOffset {
			continue // checksum field counts as zero
		}
		if i+1 < tcpLen {
			sum += uint32(tcpSegment[i])<<8 | uint32(tcpSegment[i+1])
		} else {
			// odd final byte padded with zero
			sum += uint32(tcpSegment[i]) << 8
		}
	}
	for sum>>16 != 0 {
		sum = (sum >> 16) + (sum & 0xFFFF)
	}
	return ^uint16(sum)
}

// rewriteUint16 updates a 16-bit big-endian field at buf[at] from oldW to
// newW, applying the incremental checksum at checksumOffset. Returns an
// error only if offsets are out of range (callers validate first).
func rewriteUint16(buf []byte, at int, oldW, newW uint16, checksumOffset int) {
	binary.BigEndian.PutUint16(buf[at:], newW)
	binary.BigEndian.PutUint16(buf[checksumOffset:],
		checksumUpdate(binary.BigEndian.Uint16(buf[checksumOffset:]), oldW, newW))
}
