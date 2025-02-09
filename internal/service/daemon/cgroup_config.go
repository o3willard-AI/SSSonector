package daemon

// CgroupConfig represents cgroup configuration
type CgroupConfig struct {
	Memory MemoryConfig
	CPU    CPUConfig
	IO     IOConfig
	PIDs   PIDConfig
}

// MemoryConfig represents memory cgroup configuration
type MemoryConfig struct {
	Limit     uint64 // Memory limit in bytes
	SoftLimit uint64 // Soft memory limit in bytes
	SwapLimit uint64 // Swap limit in bytes
}

// CPUConfig represents CPU cgroup configuration
type CPUConfig struct {
	Shares uint64  // CPU shares (relative weight)
	Quota  int64   // CPU quota in microseconds
	Period uint64  // CPU period in microseconds
	Cpus   string  // CPUs to use (e.g., "0-3")
	Mems   string  // Memory nodes to use
	Weight float64 // CPU weight (0-100)
}

// IOConfig represents I/O cgroup configuration
type IOConfig struct {
	Weight       uint16           // I/O weight (10-1000)
	DeviceWeight []DeviceWeight   // Per-device weights
	ReadBPS      []ThrottleDevice // Read bytes per second
	WriteBPS     []ThrottleDevice // Write bytes per second
	ReadIOPS     []ThrottleDevice // Read I/O operations per second
	WriteIOPS    []ThrottleDevice // Write I/O operations per second
}

// PIDConfig represents PID cgroup configuration
type PIDConfig struct {
	Max uint64 // Maximum number of processes
}

// DeviceWeight represents per-device I/O weight
type DeviceWeight struct {
	Major  int64  // Device major number
	Minor  int64  // Device minor number
	Weight uint16 // Device weight
}

// ThrottleDevice represents I/O throttling settings
type ThrottleDevice struct {
	Major int64  // Device major number
	Minor int64  // Device minor number
	Rate  uint64 // Rate limit
}

// GetDefaultConfig returns default cgroup configuration
func GetDefaultConfig() *CgroupConfig {
	return &CgroupConfig{
		Memory: MemoryConfig{
			Limit:     1024 * 1024 * 1024, // 1GB
			SoftLimit: 512 * 1024 * 1024,  // 512MB
			SwapLimit: 1024 * 1024 * 1024, // 1GB
		},
		CPU: CPUConfig{
			Shares: 1024,
			Quota:  100000,
			Period: 100000,
			Weight: 100,
			Cpus:   "",
			Mems:   "",
		},
		IO: IOConfig{
			Weight:       500,
			DeviceWeight: []DeviceWeight{},
			ReadBPS:      []ThrottleDevice{},
			WriteBPS:     []ThrottleDevice{},
			ReadIOPS:     []ThrottleDevice{},
			WriteIOPS:    []ThrottleDevice{},
		},
		PIDs: PIDConfig{
			Max: 1024,
		},
	}
}
