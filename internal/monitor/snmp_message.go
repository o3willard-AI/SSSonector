package monitor

import (
	"fmt"

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
	if len(data) < 2 {
		return nil, fmt.Errorf("message too short")
	}

	// Check sequence tag
	if data[0] != TagSequence {
		return nil, fmt.Errorf("invalid SNMP message format")
	}

	msg := &SNMPMessage{}

	// Skip sequence header
	offset := 2

	// Decode version
	if offset >= len(data) {
		return nil, fmt.Errorf("message too short for version")
	}
	version := int(data[offset])
	msg.Version = gosnmp.SnmpVersion(version)
	offset++

	// Decode community string
	if offset >= len(data) {
		return nil, fmt.Errorf("message too short for community")
	}
	communityLen := int(data[offset])
	offset++
	if offset+communityLen > len(data) {
		return nil, fmt.Errorf("message too short for community string")
	}
	msg.Community = string(data[offset : offset+communityLen])
	offset += communityLen

	// Decode PDU
	if offset >= len(data) {
		return nil, fmt.Errorf("message too short for PDU")
	}
	pduType := int(data[offset])
	msg.PDUType = gosnmp.PDUType(pduType)
	offset++

	// Decode request ID
	if offset+4 > len(data) {
		return nil, fmt.Errorf("message too short for request ID")
	}
	msg.RequestID = int32(data[offset])<<24 | int32(data[offset+1])<<16 |
		int32(data[offset+2])<<8 | int32(data[offset+3])
	offset += 4

	// Decode error status and index
	if offset+2 > len(data) {
		return nil, fmt.Errorf("message too short for error status/index")
	}
	msg.Error = gosnmp.SNMPError(data[offset])
	msg.Index = int(data[offset+1])
	offset += 2

	// Decode variable bindings
	msg.Variables = make([]gosnmp.SnmpPDU, 0)
	for offset < len(data) {
		if offset+2 > len(data) {
			break
		}
		oidLen := int(data[offset+1])
		offset += 2
		if offset+oidLen > len(data) {
			break
		}
		oid := string(data[offset : offset+oidLen])
		offset += oidLen

		if offset+2 > len(data) {
			break
		}
		valueType := data[offset]
		valueLen := int(data[offset+1])
		offset += 2
		if offset+valueLen > len(data) {
			break
		}
		value := data[offset : offset+valueLen]
		offset += valueLen

		pdu := gosnmp.SnmpPDU{
			Name:  oid,
			Type:  gosnmp.Asn1BER(valueType),
			Value: value,
		}
		msg.Variables = append(msg.Variables, pdu)
	}

	return msg, nil
}

// EncodeMessage encodes an SNMP message to BER format
func EncodeMessage(msg *SNMPMessage) ([]byte, error) {
	// Calculate total length
	totalLen := 1 + // Version
		2 + len(msg.Community) + // Community string
		1 + // PDU type
		4 + // Request ID
		2 + // Error status and index
		2 // Initial length of variable bindings sequence

	// Add length of variable bindings
	for _, v := range msg.Variables {
		totalLen += 2 + len(v.Name) + // OID
			2 + len(fmt.Sprintf("%v", v.Value)) // Value
	}

	// Allocate buffer
	buf := make([]byte, totalLen+2) // +2 for sequence header
	offset := 0

	// Sequence header
	buf[offset] = TagSequence
	buf[offset+1] = byte(totalLen)
	offset += 2

	// Version
	buf[offset] = byte(msg.Version)
	offset++

	// Community string
	buf[offset] = byte(len(msg.Community))
	offset++
	copy(buf[offset:], []byte(msg.Community))
	offset += len(msg.Community)

	// PDU type
	buf[offset] = byte(msg.PDUType)
	offset++

	// Request ID
	buf[offset] = byte(msg.RequestID >> 24)
	buf[offset+1] = byte(msg.RequestID >> 16)
	buf[offset+2] = byte(msg.RequestID >> 8)
	buf[offset+3] = byte(msg.RequestID)
	offset += 4

	// Error status and index
	buf[offset] = byte(msg.Error)
	buf[offset+1] = byte(msg.Index)
	offset += 2

	// Variable bindings sequence header
	varbindLen := totalLen - offset - 2
	buf[offset] = TagSequence
	buf[offset+1] = byte(varbindLen)
	offset += 2

	// Variable bindings
	for _, v := range msg.Variables {
		// OID
		buf[offset] = TagObjectID
		buf[offset+1] = byte(len(v.Name))
		offset += 2
		copy(buf[offset:], []byte(v.Name))
		offset += len(v.Name)

		// Value
		valueStr := fmt.Sprintf("%v", v.Value)
		buf[offset] = byte(v.Type)
		buf[offset+1] = byte(len(valueStr))
		offset += 2
		copy(buf[offset:], []byte(valueStr))
		offset += len(valueStr)
	}

	return buf, nil
}
