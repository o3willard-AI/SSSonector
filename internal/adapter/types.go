package adapter

// Statistics represents interface statistics
type Statistics struct {
BytesReceived   uint64
BytesSent       uint64
PacketsReceived uint64
PacketsSent     uint64
Errors          uint64
}

// Status represents interface status
type Status struct {
IsUp       bool
MTU        int
Statistics *Statistics
}

// Options represents additional interface options
type Options struct {
EnableIPv6    bool
EnableRouting bool
QueueSize     int
}
