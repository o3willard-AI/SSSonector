package daemon

// CgroupStats represents cgroup statistics
type CgroupStats struct {
	CPU     CPUStats
	Memory  MemoryStats
	IO      IOStats
	Network NetworkStats
}

// CPUStats represents CPU statistics
type CPUStats struct {
	Usage     float64 // CPU usage percentage
	User      uint64  // User mode CPU time
	System    uint64  // System mode CPU time
	Throttled uint64  // Number of throttled periods
}

// MemoryStats represents memory statistics
type MemoryStats struct {
	Usage       uint64 // Current memory usage in bytes
	MaxUsage    uint64 // Maximum memory usage in bytes
	Cache       uint64 // Page cache memory in bytes
	RSS         uint64 // Anonymous and swap cache memory in bytes
	Swap        uint64 // Swap usage in bytes
	Failcnt     uint64 // Number of memory usage hits limits
	OOMKills    uint64 // Number of OOM killer invocations
	OOMKillsR   uint64 // Number of OOM killer invocations in recursive mode
	UnderOOM    bool   // Whether the cgroup is under OOM
	OOMPriority int    // OOM priority
}

// IOStats represents I/O statistics
type IOStats struct {
	ReadBytes  uint64 // Number of bytes read
	WriteBytes uint64 // Number of bytes written
	ReadOps    uint64 // Number of read operations
	WriteOps   uint64 // Number of write operations
	ReadTime   uint64 // Total read time in nanoseconds
	WriteTime  uint64 // Total write time in nanoseconds
	QueueTime  uint64 // Total queue time in nanoseconds
	Throttled  uint64 // Number of throttled operations
}

// NetworkStats represents network statistics
type NetworkStats struct {
	RxBytes   uint64 // Number of bytes received
	TxBytes   uint64 // Number of bytes transmitted
	RxPackets uint64 // Number of packets received
	TxPackets uint64 // Number of packets transmitted
	RxDropped uint64 // Number of received packets dropped
	TxDropped uint64 // Number of transmitted packets dropped
	RxErrors  uint64 // Number of received packet errors
	TxErrors  uint64 // Number of transmitted packet errors
}
