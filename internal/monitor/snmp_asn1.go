package monitor

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

// SNMPError represents SNMP error status
type SNMPError int

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
