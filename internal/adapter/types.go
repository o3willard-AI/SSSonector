package adapter

// Statistics holds interface statistics
type Statistics struct {
BytesIn    int64
BytesOut   int64
PacketsIn  int64
PacketsOut int64
Errors     int64
}

// Status holds interface status
type Status struct {
IsUp      bool
MTU       int
Address   string
Connected bool
}

// Options holds interface options
type Options struct {
MTU     int
Address string
}
