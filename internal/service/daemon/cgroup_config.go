package daemon

// CgroupConfig represents cgroup configuration
type CgroupConfig struct {
	Memory MemoryConfig
	CPU    CPUConfig
	IO     IOConfig
	PIDs   PIDConfig
}

// MemoryConfig represents memory configuration
type MemoryConfig struct {
	Limit      uint64 // Memory limit in bytes
	SwapLimit  uint64 // Swap limit in bytes
	KernLimit  uint64 // Kernel memory limit in bytes
	OOMControl bool   // Enable OOM killer
}

// CPUConfig represents CPU configuration
type CPUConfig struct {
	Shares   uint64  // CPU shares (relative weight)
	Quota    int64   // CPU quota in microseconds
	Period   uint64  // CPU period in microseconds
	Cpuset   string  // CPUs and memory nodes
	Percent  float64 // CPU usage percentage limit
	Realtime bool    // Enable realtime scheduling
}

// IOConfig represents I/O configuration
type IOConfig struct {
	Weight       uint16 // I/O weight (relative weight)
	ReadBPS      uint64 // Read bytes per second
	WriteBPS     uint64 // Write bytes per second
	ReadIOPS     uint64 // Read I/O operations per second
	WriteIOPS    uint64 // Write I/O operations per second
	StrictLimits bool   // Enable strict I/O limits
}

// PIDConfig represents PID configuration
type PIDConfig struct {
	Limit uint64 // Maximum number of processes
}

// GetDefaultConfig returns default cgroup configuration
func GetDefaultConfig() *CgroupConfig {
	return &CgroupConfig{
		Memory: MemoryConfig{
			Limit:      1 << 30, // 1GB
			SwapLimit:  1 << 31, // 2GB
			KernLimit:  1 << 29, // 512MB
			OOMControl: true,
		},
		CPU: CPUConfig{
			Shares:   1024,   // Default shares
			Quota:    100000, // 100ms
			Period:   100000, // 100ms
			Cpuset:   "0-3",  // First 4 CPUs
			Percent:  80,     // 80% CPU limit
			Realtime: false,
		},
		IO: IOConfig{
			Weight:       500,     // Medium priority
			ReadBPS:      1 << 30, // 1GB/s
			WriteBPS:     1 << 30, // 1GB/s
			ReadIOPS:     10000,   // 10k IOPS
			WriteIOPS:    10000,   // 10k IOPS
			StrictLimits: true,
		},
		PIDs: PIDConfig{
			Limit: 1024, // Maximum 1024 processes
		},
	}
}
