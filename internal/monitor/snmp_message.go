package monitor

import (
	"fmt"
	"strconv"

	"github.com/gosnmp/gosnmp"
)

// SNMPMessage represents an SNMP message
type SNMPMessage struct {
	Version   gosnmp.SnmpVersion
	Community string
	PDUType   gosnmp.PDUType
	RequestID int32
	Variables []gosnmp.SnmpPDU
	Error     gosnmp.SNMPError
	Index     int
}

// DecodeMessage decodes an SNMP message from BER format
func DecodeMessage(data []byte) (*SNMPMessage, error) {
	if err := validateLength(len(data), MaxSNMPPacketSize); err != nil {
		return nil, fmt.Errorf("invalid message length: %w", err)
	}

	if len(data) < 2 {
		return nil, fmt.Errorf("message too short")
	}

	// Check sequence tag
	if data[0] != TagSequence {
		return nil, fmt.Errorf("invalid SNMP message format")
	}

	msg := &SNMPMessage{}
	offset := 1

	// Get sequence length
	seqLen := int(data[offset])
	offset++
	if offset+seqLen > len(data) {
		return nil, fmt.Errorf("sequence length exceeds message size")
	}

	// Decode version
	if data[offset] != TagInteger {
		return nil, fmt.Errorf("invalid version tag")
	}
	offset++
	versionLen := int(data[offset])
	offset++
	if offset+versionLen > len(data) {
		return nil, fmt.Errorf("message too short for version value")
	}
	version := int(data[offset])
	msg.Version = gosnmp.SnmpVersion(version)
	offset += versionLen

	// Decode community string
	if data[offset] != TagOctetString {
		return nil, fmt.Errorf("invalid community tag")
	}
	offset++
	communityLen := int(data[offset])
	offset++
	if offset+communityLen > len(data) {
		return nil, fmt.Errorf("message too short for community string")
	}
	community := string(data[offset : offset+communityLen])
	if err := validateCommunity(community); err != nil {
		return nil, fmt.Errorf("invalid community string: %w", err)
	}
	msg.Community = community
	offset += communityLen

	// Decode PDU
	pduType := data[offset]
	msg.PDUType = gosnmp.PDUType(pduType)
	offset++

	// Skip PDU length
	pduLen := int(data[offset])
	offset++
	if offset+pduLen > len(data) {
		return nil, fmt.Errorf("PDU length exceeds message size")
	}

	// Decode request ID
	if data[offset] != TagInteger {
		return nil, fmt.Errorf("invalid request ID tag")
	}
	offset++
	reqIDLen := int(data[offset])
	offset++
	if reqIDLen != 4 {
		return nil, fmt.Errorf("invalid request ID length")
	}
	msg.RequestID = int32(data[offset])<<24 | int32(data[offset+1])<<16 |
		int32(data[offset+2])<<8 | int32(data[offset+3])
	offset += reqIDLen

	// Decode error status
	if data[offset] != TagInteger {
		return nil, fmt.Errorf("invalid error status tag")
	}
	offset++
	if data[offset] != 1 {
		return nil, fmt.Errorf("invalid error status length")
	}
	offset++
	msg.Error = gosnmp.SNMPError(data[offset])
	offset++

	// Decode error index
	if data[offset] != TagInteger {
		return nil, fmt.Errorf("invalid error index tag")
	}
	offset++
	if data[offset] != 1 {
		return nil, fmt.Errorf("invalid error index length")
	}
	offset++
	msg.Index = int(data[offset])
	offset++

	// Decode variable bindings sequence
	if data[offset] != TagSequence {
		return nil, fmt.Errorf("invalid variable bindings tag")
	}
	offset++
	varbindLen := int(data[offset])
	offset++
	if offset+varbindLen > len(data) {
		return nil, fmt.Errorf("variable bindings length exceeds message size")
	}

	// Pre-allocate variables slice with reasonable capacity
	msg.Variables = make([]gosnmp.SnmpPDU, 0, 16)
	endOffset := offset + varbindLen

	for offset < endOffset {
		// Decode varbind sequence
		if data[offset] != TagSequence {
			return nil, fmt.Errorf("invalid varbind sequence tag")
		}
		offset++
		varbindSeqLen := int(data[offset])
		offset++
		if offset+varbindSeqLen > len(data) {
			return nil, fmt.Errorf("varbind sequence length exceeds message size")
		}

		// Decode OID
		if data[offset] != TagObjectID {
			return nil, fmt.Errorf("invalid OID tag")
		}
		offset++
		oidLen := int(data[offset])
		offset++
		if offset+oidLen > len(data) {
			return nil, fmt.Errorf("message too short for OID")
		}
		oid := string(data[offset : offset+oidLen])
		if err := validateOID(oid); err != nil {
			return nil, fmt.Errorf("invalid OID: %w", err)
		}
		offset += oidLen

		// Decode value
		if offset >= len(data) {
			return nil, fmt.Errorf("message too short for value")
		}
		valueType := data[offset]
		offset++
		valueLen := int(data[offset])
		offset++
		if offset+valueLen > len(data) {
			return nil, fmt.Errorf("message too short for value data")
		}
		value := data[offset : offset+valueLen]
		offset += valueLen

		// Convert value based on type
		var finalValue interface{}
		switch valueType {
		case TagInteger, TagCounter32, TagGauge32:
			if valueLen == 4 {
				finalValue = int64(value[0])<<24 | int64(value[1])<<16 |
					int64(value[2])<<8 | int64(value[3])
			} else {
				finalValue = int64(value[0])
			}
		case TagCounter64:
			if valueLen == 8 {
				finalValue = int64(value[0])<<56 | int64(value[1])<<48 |
					int64(value[2])<<40 | int64(value[3])<<32 |
					int64(value[4])<<24 | int64(value[5])<<16 |
					int64(value[6])<<8 | int64(value[7])
			} else {
				finalValue = int64(0)
			}
		default:
			finalValue = string(value)
		}

		pdu := gosnmp.SnmpPDU{
			Name:  oid,
			Type:  gosnmp.Asn1BER(valueType),
			Value: finalValue,
		}
		msg.Variables = append(msg.Variables, pdu)
	}

	return msg, nil
}

// EncodeMessage encodes an SNMP message to BER format
func EncodeMessage(msg *SNMPMessage) ([]byte, error) {
	// Validate message fields
	if err := validateCommunity(msg.Community); err != nil {
		return nil, fmt.Errorf("invalid community string: %w", err)
	}

	if err := validateVarBinds(len(msg.Variables)); err != nil {
		return nil, fmt.Errorf("invalid variable bindings: %w", err)
	}

	for _, v := range msg.Variables {
		if err := validateOID(v.Name); err != nil {
			return nil, fmt.Errorf("invalid OID in variable binding: %w", err)
		}
	}
	// Pre-calculate buffer size
	bufSize := 0

	// Version (tag + len + value)
	bufSize += 3

	// Community string (tag + len + value)
	bufSize += 2 + len(msg.Community)

	// PDU header (tag + len)
	bufSize += 2

	// Request ID (tag + len + value)
	bufSize += 6

	// Error status (tag + len + value)
	bufSize += 3

	// Error index (tag + len + value)
	bufSize += 3

	// Variable bindings sequence header (tag + len)
	bufSize += 2

	// Variable bindings
	for _, v := range msg.Variables {
		// Varbind sequence (tag + len)
		bufSize += 2

		// OID (tag + len + value)
		bufSize += 2 + len(v.Name)

		// Value (tag + len + value)
		var valueStr string
		switch v.Type {
		case gosnmp.Counter64, gosnmp.Gauge32, gosnmp.Integer:
			valueStr = strconv.FormatInt(v.Value.(int64), 10)
		default:
			valueStr = fmt.Sprintf("%v", v.Value)
		}
		bufSize += 2 + len(valueStr)
	}

	// Allocate buffer with pre-calculated size
	buf := make([]byte, bufSize+2) // +2 for outer sequence header
	offset := 0

	// Sequence header
	buf[offset] = TagSequence
	offset++
	buf[offset] = byte(bufSize) // Total length
	offset++

	// Version
	buf[offset] = TagInteger
	offset++
	buf[offset] = 1 // Length
	offset++
	buf[offset] = byte(msg.Version)
	offset++

	// Community string
	buf[offset] = TagOctetString
	offset++
	buf[offset] = byte(len(msg.Community))
	offset++
	copy(buf[offset:], []byte(msg.Community))
	offset += len(msg.Community)

	// PDU header
	buf[offset] = byte(msg.PDUType)
	offset++
	pduStartPos := offset // Remember position to calculate PDU length
	buf[offset] = 0       // Temporary length
	offset++

	// Request ID
	buf[offset] = TagInteger
	offset++
	buf[offset] = 4 // Length
	offset++
	buf[offset] = byte(msg.RequestID >> 24)
	buf[offset+1] = byte(msg.RequestID >> 16)
	buf[offset+2] = byte(msg.RequestID >> 8)
	buf[offset+3] = byte(msg.RequestID)
	offset += 4

	// Error status
	buf[offset] = TagInteger
	offset++
	buf[offset] = 1 // Length
	offset++
	buf[offset] = byte(msg.Error)
	offset++

	// Error index
	buf[offset] = TagInteger
	offset++
	buf[offset] = 1 // Length
	offset++
	buf[offset] = byte(msg.Index)
	offset++

	// Variable bindings sequence
	varbindStartPos := offset
	buf[offset] = TagSequence
	offset++
	buf[offset] = 0 // Temporary length
	offset++

	for _, v := range msg.Variables {
		// Start varbind sequence
		buf[offset] = TagSequence
		offset++
		varbindSeqPos := offset
		buf[offset] = 0 // Temporary length
		offset++

		// OID
		buf[offset] = TagObjectID
		offset++
		oidLen := len(v.Name)
		buf[offset] = byte(oidLen)
		offset++
		copy(buf[offset:], []byte(v.Name))
		offset += oidLen

		// Value
		buf[offset] = byte(v.Type)
		offset++
		var valueStr string
		switch v.Type {
		case gosnmp.Counter64, gosnmp.Gauge32, gosnmp.Integer:
			valueStr = strconv.FormatInt(v.Value.(int64), 10)
		default:
			valueStr = fmt.Sprintf("%v", v.Value)
		}
		valueLen := len(valueStr)
		buf[offset] = byte(valueLen)
		offset++
		copy(buf[offset:], []byte(valueStr))
		offset += valueLen

		// Update varbind sequence length
		varbindLen := offset - varbindSeqPos - 1
		buf[varbindSeqPos] = byte(varbindLen)
	}

	// Update varbindings sequence length
	varbindLen := offset - varbindStartPos - 2
	buf[varbindStartPos+1] = byte(varbindLen)

	// Update PDU length
	pduLen := offset - pduStartPos - 1
	buf[pduStartPos] = byte(pduLen)

	return buf[:offset], nil
}
