package monitor

import (
	"errors"
	"fmt"

	"github.com/gosnmp/gosnmp"
)

// This file implements the minimal SNMP v1/v2c wire decoding the agent needs.
// Response encoding is delegated to gosnmp's exported (*SnmpPacket).MarshalMsg,
// so only the inbound direction is implemented here.
//
// Wire layout (RFC 2570 style BER):
//
//	SEQUENCE {
//	  INTEGER     version        -- 0 = v1, 1 = v2c
//	  OCTET STRING community
//	  PDU [APPLICATION n] IMPLICIT SEQUENCE {
//	    INTEGER request-id
//	    INTEGER error-status
//	    INTEGER error-index
//	    SEQUENCE OF VarBind
//	    VarBind ::= SEQUENCE { OBJECT IDENTIFIER, value }
//	  }
//	}

const (
	berTagInteger     = 0x02
	berTagOctetString = 0x04
	berTagOID         = 0x06
	berTagSequence    = 0x30
	berTagContext     = 0xA0 // PDU choice: A0 get, A1 next, A2 response, A5 set-ish variants
)

var errMalformed = errors.New("malformed SNMP packet")

type berTLV struct {
	tag    byte
	value  []byte
	header int // length of header bytes
}

func berRead(data []byte, offset int) (berTLV, error) {
	if offset >= len(data) {
		return berTLV{}, errMalformed
	}
	tag := data[offset]
	pos := offset + 1
	if pos >= len(data) {
		return berTLV{}, errMalformed
	}
	length := int(data[pos])
	pos++
	if length&0x80 != 0 {
		n := length & 0x7F
		if n == 0 || n > 4 || pos+n > len(data) {
			return berTLV{}, errMalformed
		}
		length = 0
		for i := 0; i < n; i++ {
			length = length<<8 | int(data[pos+i])
		}
		pos += n
	}
	if pos+length > len(data) {
		return berTLV{}, errMalformed
	}
	return berTLV{tag: tag, value: data[pos : pos+length], header: (pos - offset) + length}, nil
}

func berReadInt(data []byte, offset int) (int64, int, error) {
	tlv, err := berRead(data, offset)
	consumed := tlv.header
	if err != nil {
		return 0, 0, err
	}
	if tlv.tag != berTagInteger || len(tlv.value) == 0 || len(tlv.value) > 8 {
		return 0, 0, errMalformed
	}
	var v int64
	if tlv.value[0]&0x80 != 0 {
		v = -1
	}
	for _, b := range tlv.value {
		v = v<<8 | int64(b)
	}
	return v, consumed, nil
}

func berReadBytes(data []byte, offset int, wantTag byte) ([]byte, int, error) {
	tlv, err := berRead(data, offset)
	consumed := tlv.header
	if err != nil {
		return nil, 0, err
	}
	if tlv.tag != wantTag {
		return nil, 0, errMalformed
	}
	return tlv.value, consumed, nil
}

// decodeOID converts BER OID content octets to dotted notation with a
// leading dot (".1.3.6.1..."), the canonical form used by the MIB tree.
func decodeOID(content []byte) (string, error) {
	if len(content) == 0 {
		return "", errMalformed
	}
	out := fmt.Sprintf(".%d.%d", content[0]/40, content[0]%40)
	for i := 1; i < len(content); {
		var sub uint64
		for i < len(content) {
			b := content[i]
			i++
			sub = sub<<7 | uint64(b&0x7F)
			if b&0x80 == 0 {
				break
			}
		}
		out += fmt.Sprintf(".%d", sub)
	}
	return out, nil
}

// DecodeSNMP decodes an SNMP v1/v2c request into a gosnmp packet struct.
// OIDs are normalized to leading-dot form for MIB lookup.
func DecodeSNMP(data []byte) (*gosnmp.SnmpPacket, error) {
	msg, err := berRead(data, 0)
	if err != nil || msg.tag != berTagSequence {
		return nil, errMalformed
	}

	version, consumed, err := berReadInt(msg.value, 0)
	if err != nil {
		return nil, errMalformed
	}
	var ver gosnmp.SnmpVersion
	switch version {
	case 0:
		ver = gosnmp.Version1
	case 1:
		ver = gosnmp.Version2c
	default:
		return nil, fmt.Errorf("%w: unsupported version %d", errMalformed, version)
	}

	community, cUsed, err := berReadBytes(msg.value, consumed, berTagOctetString)
	if err != nil {
		return nil, errMalformed
	}

	pduTLV, err := berRead(msg.value, consumed+cUsed)
	if err != nil {
		return nil, errMalformed
	}
	if pduTLV.tag < berTagContext || pduTLV.tag > berTagContext+5 {
		return nil, errMalformed
	}

	reqID, pUsed, err := berReadInt(pduTLV.value, 0)
	if err != nil {
		return nil, errMalformed
	}
	errStatus, eUsed, err := berReadInt(pduTLV.value, pUsed)
	if err != nil {
		return nil, errMalformed
	}
	errIndex, iUsed, err := berReadInt(pduTLV.value, pUsed+eUsed)
	if err != nil {
		return nil, errMalformed
	}

	vbSeq, err := berRead(pduTLV.value, pUsed+eUsed+iUsed)
	if err != nil || vbSeq.tag != berTagSequence {
		return nil, errMalformed
	}

	packet := &gosnmp.SnmpPacket{
		Version:       ver,
		Community:     string(community),
		PDUType:       gosnmp.PDUType(pduTLV.tag),
		RequestID:     uint32(reqID),
		Error:         gosnmp.SNMPError(errStatus),
		ErrorIndex:    uint8(errIndex),
		Variables:     make([]gosnmp.SnmpPDU, 0, 8),
		SecurityModel: gosnmp.UserSecurityModel,
	}

	offset := 0
	for offset < len(vbSeq.value) {
		vb, err := berRead(vbSeq.value, offset)
		vUsed := vb.header
		if err != nil || vb.tag != berTagSequence {
			return nil, errMalformed
		}
		oidContent, oUsed, err := berReadBytes(vb.value, 0, berTagOID)
		if err != nil {
			return nil, errMalformed
		}
		oid, err := decodeOID(oidContent)
		if err != nil {
			return nil, errMalformed
		}
		valTLV, err := berRead(vb.value, oUsed)
		if err != nil {
			return nil, errMalformed
		}
		packet.Variables = append(packet.Variables, gosnmp.SnmpPDU{
			Name:  oid,
			Type:  gosnmp.Asn1BER(valTLV.tag),
			Value: decodeVarbindValue(valTLV),
		})
		offset += vUsed
	}

	return packet, nil
}

// decodeVarbindValue converts a varbind value TLV into a Go value mirroring
// what a gosnmp client would produce for the same encoding.
func decodeVarbindValue(tlv berTLV) interface{} {
	switch gosnmp.Asn1BER(tlv.tag) {
	case gosnmp.Integer:
		var v int64
		if len(tlv.value) > 0 && len(tlv.value) <= 8 {
			if tlv.value[0]&0x80 != 0 {
				v = -1
			}
			for _, b := range tlv.value {
				v = v<<8 | int64(b)
			}
		}
		return v
	case gosnmp.OctetString:
		return string(tlv.value)
	case gosnmp.Counter32, gosnmp.Gauge32, gosnmp.TimeTicks:
		var v uint32
		if len(tlv.value) > 0 && len(tlv.value) <= 5 {
			for _, b := range tlv.value {
				v = v<<8 | uint32(b)
			}
		}
		return v
	case gosnmp.Counter64:
		var v uint64
		if len(tlv.value) > 0 && len(tlv.value) <= 9 {
			for _, b := range tlv.value {
				v = v<<8 | uint64(b)
			}
		}
		return v
	default:
		return tlv.value
	}
}
