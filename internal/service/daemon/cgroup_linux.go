//go:build linux

package daemon

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/containerd/cgroups"
	"github.com/opencontainers/runtime-spec/specs-go"
	"go.uber.org/zap"
)

// CgroupManager handles Linux control group management
type CgroupManager struct {
	logger     *zap.Logger
	cgroup     cgroups.Cgroup
	path       string
	resources  *specs.LinuxResources
	hierarchy  cgroups.Hierarchy
	subsystems []cgroups.Subsystem
}

// CgroupConfig represents cgroup configuration options
type CgroupConfig struct {
	// Memory limits
	MemoryLimit     int64  // Memory limit in bytes
	MemorySoftLimit int64  // Soft memory limit in bytes
	MemorySwapLimit int64  // Swap limit in bytes
	KernelMemLimit  int64  // Kernel memory limit in bytes
	OOMKillDisable  bool   // Disable OOM killer
	SwapPriority    uint64 // Swap priority

	// CPU limits
	CPUShares          uint64 // CPU shares (relative weight)
	CPUQuota           int64  // CPU quota in microseconds
	CPUPeriod          uint64 // CPU period in microseconds
	CPURealtimeRuntime int64  // CPU realtime runtime in microseconds
	CPURealtimePeriod  uint64 // CPU realtime period in microseconds
	CPUSetCPUs         string // CPUs in which to allow execution (0-3,...)
	CPUSetMems         string // Memory nodes in which to allow execution (0-3,...)

	// Block IO limits
	BlkioWeight       uint16 // Block IO weight (relative weight)
	BlkioLeafWeight   uint16 // Block IO leaf weight
	BlkioDeviceWeight []specs.LinuxWeightDevice
	BlkioReadBps      []specs.LinuxThrottleDevice
	BlkioWriteBps     []specs.LinuxThrottleDevice
	BlkioReadIOps     []specs.LinuxThrottleDevice
	BlkioWriteIOps    []specs.LinuxThrottleDevice

	// Process limits
	PidsLimit int64 // Maximum number of processes
}

// NewCgroupManager creates a new cgroup manager
func NewCgroupManager(logger *zap.Logger, name string) (*CgroupManager, error) {
	// Check if cgroups are available
	if _, err := os.Stat("/sys/fs/cgroup"); os.IsNotExist(err) {
		return nil, fmt.Errorf("cgroups not available: %w", err)
	}

	// Create cgroup path
	path := filepath.Join("/sys/fs/cgroup", "sssonector", name)

	// Initialize manager
	manager := &CgroupManager{
		logger: logger,
		path:   path,
		resources: &specs.LinuxResources{
			Memory:  &specs.LinuxMemory{},
			CPU:     &specs.LinuxCPU{},
			BlockIO: &specs.LinuxBlockIO{},
			Pids:    &specs.LinuxPids{},
		},
	}

	return manager, nil
}

// Start initializes the cgroup
func (m *CgroupManager) Start(config CgroupConfig) error {
	// Apply configuration
	m.applyCgroupConfig(config)

	// Create cgroup
	cg, err := cgroups.New(cgroups.Systemd, cgroups.StaticPath(m.path), m.resources)
	if err != nil {
		return fmt.Errorf("failed to create cgroup: %w", err)
	}
	m.cgroup = cg

	// Add current process
	if err := m.AddProcess(os.Getpid()); err != nil {
		return fmt.Errorf("failed to add process to cgroup: %w", err)
	}

	m.logger.Info("Initialized cgroup",
		zap.String("path", m.path))

	return nil
}

// Stop cleans up the cgroup
func (m *CgroupManager) Stop() error {
	if m.cgroup != nil {
		if err := m.cgroup.Delete(); err != nil {
			return fmt.Errorf("failed to delete cgroup: %w", err)
		}
	}
	return nil
}

// AddProcess adds a process to the cgroup
func (m *CgroupManager) AddProcess(pid int) error {
	if m.cgroup == nil {
		return fmt.Errorf("cgroup not initialized")
	}

	if err := m.cgroup.Add(cgroups.Process{Pid: pid}); err != nil {
		return fmt.Errorf("failed to add process %d: %w", pid, err)
	}

	return nil
}

// RemoveProcess removes a process from the cgroup
func (m *CgroupManager) RemoveProcess(pid int) error {
	if m.cgroup == nil {
		return fmt.Errorf("cgroup not initialized")
	}

	processes, err := m.cgroup.Processes(cgroups.Devices, false)
	if err != nil {
		return fmt.Errorf("failed to get processes: %w", err)
	}

	for _, p := range processes {
		if p.Pid == pid {
			if err := m.cgroup.Delete(); err != nil {
				return fmt.Errorf("failed to remove process %d: %w", pid, err)
			}
			return nil
		}
	}

	return fmt.Errorf("process %d not found in cgroup", pid)
}

// UpdateConfig updates the cgroup configuration
func (m *CgroupManager) UpdateConfig(config CgroupConfig) error {
	if m.cgroup == nil {
		return fmt.Errorf("cgroup not initialized")
	}

	m.applyCgroupConfig(config)

	if err := m.cgroup.Update(m.resources); err != nil {
		return fmt.Errorf("failed to update cgroup: %w", err)
	}

	return nil
}

// GetStats returns cgroup statistics
func (m *CgroupManager) GetStats() (interface{}, error) {
	if m.cgroup == nil {
		return nil, fmt.Errorf("cgroup not initialized")
	}

	metrics, err := m.cgroup.Stat()
	if err != nil {
		return nil, fmt.Errorf("failed to get cgroup stats: %w", err)
	}

	return metrics, nil
}

// applyCgroupConfig applies the configuration to resources
func (m *CgroupManager) applyCgroupConfig(config CgroupConfig) {
	// Memory limits
	m.resources.Memory.Limit = &config.MemoryLimit
	m.resources.Memory.Reservation = &config.MemorySoftLimit
	m.resources.Memory.Swap = &config.MemorySwapLimit
	m.resources.Memory.Kernel = &config.KernelMemLimit
	m.resources.Memory.DisableOOMKiller = &config.OOMKillDisable
	m.resources.Memory.Swappiness = &config.SwapPriority

	// CPU limits
	m.resources.CPU.Shares = &config.CPUShares
	m.resources.CPU.Quota = &config.CPUQuota
	m.resources.CPU.Period = &config.CPUPeriod
	m.resources.CPU.RealtimeRuntime = &config.CPURealtimeRuntime
	m.resources.CPU.RealtimePeriod = &config.CPURealtimePeriod
	m.resources.CPU.Cpus = config.CPUSetCPUs
	m.resources.CPU.Mems = config.CPUSetMems

	// Block IO limits
	m.resources.BlockIO.Weight = &config.BlkioWeight
	m.resources.BlockIO.LeafWeight = &config.BlkioLeafWeight
	m.resources.BlockIO.WeightDevice = config.BlkioDeviceWeight
	m.resources.BlockIO.ThrottleReadBpsDevice = config.BlkioReadBps
	m.resources.BlockIO.ThrottleWriteBpsDevice = config.BlkioWriteBps
	m.resources.BlockIO.ThrottleReadIOPSDevice = config.BlkioReadIOps
	m.resources.BlockIO.ThrottleWriteIOPSDevice = config.BlkioWriteIOps

	// Process limits
	m.resources.Pids.Limit = config.PidsLimit
}

// GetDefaultConfig returns default cgroup configuration
func GetDefaultConfig() CgroupConfig {
	return CgroupConfig{
		MemoryLimit:     1024 * 1024 * 1024, // 1GB
		MemorySoftLimit: 512 * 1024 * 1024,  // 512MB
		CPUShares:       1024,               // Default CPU shares
		CPUQuota:        100000,             // 100ms
		CPUPeriod:       100000,             // 100ms
		BlkioWeight:     500,                // Mid-range IO weight
		PidsLimit:       1024,               // Max 1024 processes
	}
}
