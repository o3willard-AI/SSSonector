package connection

// TrackerState represents the state of a tracked connection
type TrackerState uint8

const (
	// TrackerStateNew represents a new connection
	TrackerStateNew TrackerState = iota
	// TrackerStateActive represents an active connection
	TrackerStateActive
	// TrackerStateClosing represents a closing connection
	TrackerStateClosing
	// TrackerStateClosed represents a closed connection
	TrackerStateClosed
)

// String returns the string representation of TrackerState
func (s TrackerState) String() string {
	switch s {
	case TrackerStateNew:
		return "new"
	case TrackerStateActive:
		return "active"
	case TrackerStateClosing:
		return "closing"
	case TrackerStateClosed:
		return "closed"
	default:
		return "unknown"
	}
}

// ManagerState represents the state of a managed connection
type ManagerState string

const (
	// ManagerStateConnected represents a connected state
	ManagerStateConnected ManagerState = "Connected"
	// ManagerStateDisconnected represents a disconnected state
	ManagerStateDisconnected ManagerState = "Disconnected"
)
