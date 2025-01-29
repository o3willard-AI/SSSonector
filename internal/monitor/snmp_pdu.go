package monitor

import (
	"encoding/asn1"
	"fmt"
	"strconv"
	"strings"
)

// SNMP PDU types
const (
	snmpGetRequest     = 0xa0
	snmpGetNextRequest = 0xa1
	snmpGetResponse    = 0xa2
	snmpSetRequest     = 0xa3
)

// SNMP data types
const (
	snmpInteger     = 0x02
	snmpOctetString = 0x04
	snmpNull        = 0x05
	snmpObjectID    = 0x06
	snmpSequence    = 0x30
	snmpCounter64   = 0x46
	snmpGauge32     = 0x42
	snmpTimeTicks   = 0x43
)

// SNMP error codes
const (
	snmpNoError            = 0
	snmpTooBig             = 1
	snmpNoSuchName         = 2
	snmpBadValue           = 3
	snmpReadOnly           = 4
	snmpGenErr             = 5
	snmpNoAccess           = 6
	snmpWrongType          = 7
	snmpWrongLength        = 8
	snmpWrongEncoding      = 9
	snmpWrongValue         = 10
	snmpNoCreation         = 11
	snmpInconsistentValue  = 12
	snmpResourceUnavail    = 13
	snmpCommitFailed       = 14
	snmpUndoFailed         = 15
	snmpAuthorizationError = 16
	snmpNotWritable        = 17
	snmpInconsistentName   = 18
)

// SnmpPDU represents an SNMP Protocol Data Unit
type SnmpPDU struct {
	Type  byte
	Value interface{}
}

// SnmpPacket represents a complete SNMP packet
type SnmpPacket struct {
	Version    int
	Community  string
	PDUType    byte
	RequestID  int
	Error      int
	ErrorIndex int
	Variables  []SnmpPDU
}

// marshalPDU encodes a PDU into ASN.1 BER format
func marshalPDU(pdu SnmpPDU) ([]byte, error) {
	// ASN.1 class types
	const (
		classUniversal = 0x00
	)

	var value asn1.RawValue

	switch v := pdu.Value.(type) {
	case int64:
		value = asn1.RawValue{
			Class:      classUniversal,
			Tag:        int(snmpInteger),
			IsCompound: false,
			Bytes:      encodeInt64(v),
		}
	case uint64:
		value = asn1.RawValue{
			Class:      classUniversal,
			Tag:        int(snmpCounter64),
			IsCompound: false,
			Bytes:      encodeUint64(v),
		}
	case string:
		value = asn1.RawValue{
			Class:      classUniversal,
			Tag:        int(snmpOctetString),
			IsCompound: false,
			Bytes:      []byte(v),
		}
	case []byte:
		value = asn1.RawValue{
			Class:      classUniversal,
			Tag:        int(snmpOctetString),
			IsCompound: false,
			Bytes:      v,
		}
	default:
		return nil, fmt.Errorf("unsupported PDU value type: %T", v)
	}

	return asn1.Marshal(value)
}

// unmarshalPDU decodes a PDU from ASN.1 BER format
func unmarshalPDU(data []byte) (SnmpPDU, error) {
	var raw asn1.RawValue
	_, err := asn1.Unmarshal(data, &raw)
	if err != nil {
		return SnmpPDU{}, fmt.Errorf("failed to unmarshal PDU: %w", err)
	}

	pdu := SnmpPDU{Type: byte(raw.Tag)}

	switch raw.Tag {
	case snmpInteger:
		pdu.Value = decodeInt64(raw.Bytes)
	case snmpCounter64:
		pdu.Value = decodeUint64(raw.Bytes)
	case snmpOctetString:
		pdu.Value = string(raw.Bytes)
	case snmpObjectID:
		pdu.Value = decodeOID(raw.Bytes)
	case snmpNull:
		pdu.Value = nil
	default:
		return SnmpPDU{}, fmt.Errorf("unsupported PDU type: %d", raw.Tag)
	}

	return pdu, nil
}

// encodeInt64 encodes an int64 in big-endian format
func encodeInt64(value int64) []byte {
	bytes := make([]byte, 8)
	for i := 7; i >= 0; i-- {
		bytes[i] = byte(value & 0xff)
		value >>= 8
	}
	return bytes
}

// decodeInt64 decodes an int64 from big-endian format
func decodeInt64(bytes []byte) int64 {
	var value int64
	for _, b := range bytes {
		value = (value << 8) | int64(b)
	}
	return value
}

// encodeUint64 encodes a uint64 in big-endian format
func encodeUint64(value uint64) []byte {
	bytes := make([]byte, 8)
	for i := 7; i >= 0; i-- {
		bytes[i] = byte(value & 0xff)
		value >>= 8
	}
	return bytes
}

// decodeUint64 decodes a uint64 from big-endian format
func decodeUint64(bytes []byte) uint64 {
	var value uint64
	for _, b := range bytes {
		value = (value << 8) | uint64(b)
	}
	return value
}

// decodeOID decodes an OID from ASN.1 format
func decodeOID(bytes []byte) string {
	if len(bytes) == 0 {
		return ""
	}

	// First byte encodes first two numbers: X.Y where X = byte/40 and Y = byte%40
	first := bytes[0] / 40
	second := bytes[0] % 40
	result := fmt.Sprintf("%d.%d", first, second)

	// Remaining bytes encode subsequent numbers
	var value uint
	for i := 1; i < len(bytes); i++ {
		value = (value << 7) | uint(bytes[i]&0x7f)
		if bytes[i]&0x80 == 0 {
			result += fmt.Sprintf(".%d", value)
			value = 0
		}
	}

	return result
}

// encodeOID encodes an OID into ASN.1 format
func encodeOID(oid string) ([]byte, error) {
	// Split OID into components
	parts := strings.Split(oid, ".")
	if len(parts) < 2 {
		return nil, fmt.Errorf("invalid OID format: %s", oid)
	}

	// Parse first two numbers
	first, err := strconv.Atoi(parts[0])
	if err != nil {
		return nil, fmt.Errorf("invalid first OID component: %w", err)
	}
	second, err := strconv.Atoi(parts[1])
	if err != nil {
		return nil, fmt.Errorf("invalid second OID component: %w", err)
	}

	// First byte encodes first two numbers: first*40 + second
	result := []byte{byte(first*40 + second)}

	// Encode remaining numbers
	for _, part := range parts[2:] {
		num, err := strconv.ParseUint(part, 10, 32)
		if err != nil {
			return nil, fmt.Errorf("invalid OID component: %w", err)
		}

		// Encode number in base 128 with continuation bits
		if num < 128 {
			result = append(result, byte(num))
		} else {
			// Calculate how many bytes we need
			bytes := make([]byte, 0)
			for num > 0 {
				bytes = append([]byte{byte(num & 0x7f)}, bytes...)
				num >>= 7
			}
			// Set continuation bits on all but the last byte
			for i := 0; i < len(bytes)-1; i++ {
				bytes[i] |= 0x80
			}
			result = append(result, bytes...)
		}
	}

	return result, nil
}
