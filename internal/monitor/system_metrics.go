package monitor

import (
	"bufio"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
)

type cpuStats struct {
	user    uint64
	nice    uint64
	system  uint64
	idle    uint64
	iowait  uint64
	irq     uint64
	softirq uint64
	steal   uint64
	guest   uint64
}

type diskStats struct {
	readOps      uint64
	readSectors  uint64
	writeOps     uint64
	writeSectors uint64
}

type netStats struct {
	bytesReceived   uint64
	bytesSent       uint64
	packetsReceived uint64
	packetsSent     uint64
}

// SystemMetricsCollector collects system-wide metrics
type SystemMetricsCollector struct {
	lastCPU     cpuStats
	lastDisk    diskStats
	lastNet     netStats
	lastCollect time.Time
}

// NewSystemMetricsCollector creates a new system metrics collector
func NewSystemMetricsCollector() *SystemMetricsCollector {
	return &SystemMetricsCollector{
		lastCollect: time.Now(),
	}
}

// CollectMetrics gathers system metrics and updates the provided Metrics struct
func (c *SystemMetricsCollector) CollectMetrics(m *Metrics) error {
	if err := c.collectCPUMetrics(m); err != nil {
		return fmt.Errorf("failed to collect CPU metrics: %w", err)
	}

	if err := c.collectDiskMetrics(m); err != nil {
		return fmt.Errorf("failed to collect disk metrics: %w", err)
	}

	if err := c.collectNetworkMetrics(m); err != nil {
		return fmt.Errorf("failed to collect network metrics: %w", err)
	}

	return nil
}

// collectCPUMetrics reads /proc/stat and calculates CPU usage
func (c *SystemMetricsCollector) collectCPUMetrics(m *Metrics) error {
	file, err := os.Open("/proc/stat")
	if err != nil {
		return err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		fields := strings.Fields(scanner.Text())
		if len(fields) < 8 || fields[0] != "cpu" {
			continue
		}

		current := cpuStats{}
		current.user, _ = strconv.ParseUint(fields[1], 10, 64)
		current.nice, _ = strconv.ParseUint(fields[2], 10, 64)
		current.system, _ = strconv.ParseUint(fields[3], 10, 64)
		current.idle, _ = strconv.ParseUint(fields[4], 10, 64)
		current.iowait, _ = strconv.ParseUint(fields[5], 10, 64)
		current.irq, _ = strconv.ParseUint(fields[6], 10, 64)
		current.softirq, _ = strconv.ParseUint(fields[7], 10, 64)

		if len(fields) > 8 {
			current.steal, _ = strconv.ParseUint(fields[8], 10, 64)
		}
		if len(fields) > 9 {
			current.guest, _ = strconv.ParseUint(fields[9], 10, 64)
		}

		// Calculate CPU usage percentage
		if c.lastCollect.Unix() > 0 {
			prevIdle := c.lastCPU.idle + c.lastCPU.iowait
			idle := current.idle + current.iowait

			prevNonIdle := c.lastCPU.user + c.lastCPU.nice + c.lastCPU.system +
				c.lastCPU.irq + c.lastCPU.softirq + c.lastCPU.steal + c.lastCPU.guest
			nonIdle := current.user + current.nice + current.system +
				current.irq + current.softirq + current.steal + current.guest

			prevTotal := prevIdle + prevNonIdle
			total := idle + nonIdle

			totalDiff := total - prevTotal
			idleDiff := idle - prevIdle

			if totalDiff > 0 {
				m.CPUUsage = (1 - float64(idleDiff)/float64(totalDiff)) * 100
			}
		}

		c.lastCPU = current
		break
	}

	return scanner.Err()
}

// collectDiskMetrics reads /proc/diskstats and calculates disk I/O
func (c *SystemMetricsCollector) collectDiskMetrics(m *Metrics) error {
	file, err := os.Open("/proc/diskstats")
	if err != nil {
		return err
	}
	defer file.Close()

	var current diskStats
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		fields := strings.Fields(scanner.Text())
		if len(fields) < 14 {
			continue
		}

		// Skip non-disk devices
		if strings.HasPrefix(fields[2], "loop") || strings.HasPrefix(fields[2], "ram") {
			continue
		}

		readOps, _ := strconv.ParseUint(fields[3], 10, 64)
		readSectors, _ := strconv.ParseUint(fields[5], 10, 64)
		writeOps, _ := strconv.ParseUint(fields[7], 10, 64)
		writeSectors, _ := strconv.ParseUint(fields[9], 10, 64)

		current.readOps += readOps
		current.readSectors += readSectors
		current.writeOps += writeOps
		current.writeSectors += writeSectors
	}

	// Calculate disk I/O rate (sectors are 512 bytes)
	if c.lastCollect.Unix() > 0 {
		elapsed := time.Since(c.lastCollect).Seconds()
		if elapsed > 0 {
			readBytes := (current.readSectors - c.lastDisk.readSectors) * 512
			writeBytes := (current.writeSectors - c.lastDisk.writeSectors) * 512
			m.DiskIO = int64((float64(readBytes+writeBytes) / elapsed))
		}
	}

	c.lastDisk = current
	return scanner.Err()
}

// collectNetworkMetrics reads /proc/net/dev and calculates network I/O
func (c *SystemMetricsCollector) collectNetworkMetrics(m *Metrics) error {
	file, err := os.Open("/proc/net/dev")
	if err != nil {
		return err
	}
	defer file.Close()

	var current netStats
	scanner := bufio.NewScanner(file)
	// Skip header lines
	scanner.Scan()
	scanner.Scan()

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		fields := strings.Fields(line)
		if len(fields) < 17 || strings.HasPrefix(fields[0], "lo:") {
			continue
		}

		bytesReceived, _ := strconv.ParseUint(fields[1], 10, 64)
		packetsReceived, _ := strconv.ParseUint(fields[2], 10, 64)
		bytesSent, _ := strconv.ParseUint(fields[9], 10, 64)
		packetsSent, _ := strconv.ParseUint(fields[10], 10, 64)

		current.bytesReceived += bytesReceived
		current.packetsReceived += packetsReceived
		current.bytesSent += bytesSent
		current.packetsSent += packetsSent
	}

	// Calculate network I/O rate
	if c.lastCollect.Unix() > 0 {
		elapsed := time.Since(c.lastCollect).Seconds()
		if elapsed > 0 {
			bytesTotal := (current.bytesReceived - c.lastNet.bytesReceived) +
				(current.bytesSent - c.lastNet.bytesSent)
			m.NetworkIO = int64(float64(bytesTotal) / elapsed)
		}
	}

	c.lastNet = current
	c.lastCollect = time.Now()
	return scanner.Err()
}
