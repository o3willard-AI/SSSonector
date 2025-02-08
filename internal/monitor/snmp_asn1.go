package monitor

import (
	"fmt"
)

// Maximum sizes defined by RFCs
const (
	MaxSNMPPacketSize  = 65507 // Maximum UDP packet size
	MaxOIDLength       = 128   // Maximum OID length in bytes
	MaxCommunityLength = 32    // RFC 3584 max community string length
	MaxVarBinds        = 2048  // Maximum number of variable bindings
	MaxStringLength    = 65535 // Maximum octet string length
)

// ASN.1 BER type constants
const (
	// Tag classes
	ClassUniversal       = 0x00
	ClassApplication     = 0x40
	ClassContextSpecific = 0x80
	ClassPrivate         = 0xC0

	// Universal class tags
	TagInteger        = 0x02
	TagOctetString    = 0x04
	TagNull           = 0x05
	TagObjectID       = 0x06
	TagSequence       = 0x30
	TagIPAddress      = 0x40
	TagCounter32      = 0x41
	TagGauge32        = 0x42
	TagTimeTicks      = 0x43
	TagOpaque         = 0x44
	TagCounter64      = 0x46
	TagNoSuchObject   = 0x80
	TagNoSuchInstance = 0x81
	TagEndOfMibView   = 0x82

	// SNMP PDU tags (application class)
	TagGetRequest     = 0xA0
	TagGetNextRequest = 0xA1
	TagGetResponse    = 0xA2
	TagSetRequest     = 0xA3
	TagTrap           = 0xA4
	TagGetBulkRequest = 0xA5
	TagInformRequest  = 0xA6
	TagTrapV2         = 0xA7
	TagReport         = 0xA8
)

// SNMPVersion represents SNMP protocol version
type SNMPVersion int

const (
	Version1  SNMPVersion = 0
	Version2c SNMPVersion = 1
	Version3  SNMPVersion = 3
)

// SNMPError represents SNMP error status with description
type SNMPError int

// Error returns a human-readable error description
func (e SNMPError) Error() string {
	switch e {
	case NoError:
		return "No error"
	case TooBig:
		return "Response too big"
	case NoSuchName:
		return "Object not found"
	case BadValue:
		return "Bad value"
	case ReadOnly:
		return "Object is read-only"
	case GenErr:
		return "General error"
	case NoAccess:
		return "Access denied"
	case WrongType:
		return "Wrong type"
	case WrongLength:
		return "Wrong length"
	case WrongEncoding:
		return "Wrong encoding"
	case WrongValue:
		return "Wrong value"
	case NoCreation:
		return "Object creation not allowed"
	case InconsistentValue:
		return "Inconsistent value"
	case ResourceUnavailable:
		return "Resource unavailable"
	case CommitFailed:
		return "Commit failed"
	case UndoFailed:
		return "Undo failed"
	case AuthorizationError:
		return "Authorization error"
	case NotWritable:
		return "Object not writable"
	case InconsistentName:
		return "Inconsistent name"
	default:
		return fmt.Sprintf("Unknown error (%d)", e)
	}
}

// Validation functions
func validateLength(length, max int) error {
	if length < 0 {
		return fmt.Errorf("negative length: %d", length)
	}
	if length > max {
		return fmt.Errorf("length %d exceeds maximum %d", length, max)
	}
	return nil
}

func validateOID(oid string) error {
	if len(oid) == 0 {
		return fmt.Errorf("empty OID")
	}
	if len(oid) > MaxOIDLength {
		return fmt.Errorf("OID length %d exceeds maximum %d", len(oid), MaxOIDLength)
	}
	if oid[0] != '.' {
		return fmt.Errorf("OID must start with '.'")
	}
	return nil
}

func validateCommunity(community string) error {
	if len(community) == 0 {
		return fmt.Errorf("empty community string")
	}
	if len(community) > MaxCommunityLength {
		return fmt.Errorf("community string length %d exceeds maximum %d", len(community), MaxCommunityLength)
	}
	for _, c := range community {
		if c == 0 {
			return fmt.Errorf("community string contains null byte")
		}
		if !isPrintableASCII(c) {
			return fmt.Errorf("community string contains non-printable character: %x", c)
		}
	}
	return nil
}

func isPrintableASCII(c rune) bool {
	return c >= 0x20 && c <= 0x7E
}

func validateVarBinds(count int) error {
	if count < 0 {
		return fmt.Errorf("negative variable binding count: %d", count)
	}
	if count > MaxVarBinds {
		return fmt.Errorf("variable binding count %d exceeds maximum %d", count, MaxVarBinds)
	}
	return nil
}

const (
	NoError             SNMPError = 0
	TooBig              SNMPError = 1
	NoSuchName          SNMPError = 2
	BadValue            SNMPError = 3
	ReadOnly            SNMPError = 4
	GenErr              SNMPError = 5
	NoAccess            SNMPError = 6
	WrongType           SNMPError = 7
	WrongLength         SNMPError = 8
	WrongEncoding       SNMPError = 9
	WrongValue          SNMPError = 10
	NoCreation          SNMPError = 11
	InconsistentValue   SNMPError = 12
	ResourceUnavailable SNMPError = 13
	CommitFailed        SNMPError = 14
	UndoFailed          SNMPError = 15
	AuthorizationError  SNMPError = 16
	NotWritable         SNMPError = 17
	InconsistentName    SNMPError = 18
)
