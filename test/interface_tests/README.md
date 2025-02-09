# TUN Interface Test Suite

This document outlines the test procedures for validating TUN interface management in SSSonector.

## Test Categories

### 1. State Transition Tests

#### Test Cases

1. Normal State Flow
```go
func TestNormalStateTransitions(t *testing.T) {
    // Test progression through all normal states
    // Uninitialized -> Initializing -> Ready -> Stopping -> Stopped
}
```

2. Error State Transitions
```go
func TestErrorStateTransitions(t *testing.T) {
    // Test error handling and recovery
    // Test invalid state transitions
    // Verify error state cleanup
}
```

3. Concurrent Operations
```go
func TestConcurrentOperations(t *testing.T) {
    // Test state transitions under concurrent load
    // Verify thread safety
    // Test race conditions
}
```

### 2. Cleanup Tests

#### Test Cases

1. Normal Cleanup
```bash
#!/bin/bash
# test_normal_cleanup.sh

# Start service
systemctl start sssonector

# Create test connections
./create_test_connections.sh 10

# Stop service gracefully
systemctl stop sssonector

# Verify cleanup
./verify_interfaces.sh
```

2. Force Kill Recovery
```bash
#!/bin/bash
# test_force_kill.sh

# Start service
systemctl start sssonector

# Create connections
./create_test_connections.sh 5

# Force kill
kill -9 $(pidof sssonector)

# Start service again
systemctl start sssonector

# Verify cleanup of old interfaces
./verify_interfaces.sh
```

3. Network Interruption
```bash
#!/bin/bash
# test_network_interruption.sh

# Start service
systemctl start sssonector

# Create connections
./create_test_connections.sh 3

# Simulate network failure
tc qdisc add dev eth0 root netem loss 100%

# Wait for timeout
sleep 30

# Restore network
tc qdisc del dev eth0 root

# Verify interface cleanup
./verify_interfaces.sh
```

### 3. Performance Tests

#### Test Cases

1. Multiple Interface Creation
```go
func TestMultipleInterfaces(t *testing.T) {
    // Create multiple interfaces concurrently
    // Verify resource usage
    // Check for memory leaks
}
```

2. Rapid Creation/Deletion
```go
func TestRapidLifecycle(t *testing.T) {
    // Rapidly create and delete interfaces
    // Monitor system resources
    // Check for resource leaks
}
```

3. Load Testing
```go
func TestUnderLoad(t *testing.T) {
    // Create interfaces under system load
    // Monitor performance metrics
    // Verify cleanup under load
}
```

## Test Utilities

### Interface Verification

```go
// verify/interfaces.go

func VerifyInterfaceCleanup(interfaceName string) error {
    // Check interface doesn't exist
    if _, err := os.Stat(fmt.Sprintf("/sys/class/net/%s", interfaceName)); !os.IsNotExist(err) {
        return fmt.Errorf("interface %s still exists", interfaceName)
    }

    // Check routes are cleaned up
    routes, err := netlink.RouteList(nil, netlink.FAMILY_ALL)
    if err != nil {
        return fmt.Errorf("failed to list routes: %w", err)
    }
    for _, route := range routes {
        if route.LinkIndex == interfaceIndex {
            return fmt.Errorf("route for interface %s still exists", interfaceName)
        }
    }

    return nil
}
```

### Resource Monitoring

```go
// monitor/resources.go

type ResourceStats struct {
    Interfaces int
    Memory     uint64
    CPUUsage   float64
    FileDescs  int
}

func CollectResourceStats() (*ResourceStats, error) {
    // Collect current resource usage
    // Monitor system metrics
    // Track interface count
}
```

## Running Tests

### 1. Unit Tests
```bash
cd test/interface_tests
go test -v ./...
```

### 2. Integration Tests
```bash
cd test/scenarios
./run_all_tests.sh
```

### 3. Performance Tests
```bash
cd test/performance
./run_perf_tests.sh
```

## Test Environment Setup

### Requirements

1. Test VM Configuration:
   - 4GB RAM minimum
   - 2 CPU cores
   - 20GB storage
   - Ubuntu 20.04 or later

2. Network Setup:
   - Isolated test network
   - Controlled bandwidth
   - Simulated latency capability

3. Monitoring Tools:
   - sar
   - iptraf-ng
   - nethogs
   - tcpdump

### Environment Preparation

```bash
#!/bin/bash
# setup_test_env.sh

# Install dependencies
apt-get update
apt-get install -y \
    iptraf-ng \
    nethogs \
    tcpdump \
    sysstat

# Configure system limits
cat >> /etc/sysctl.conf << EOF
net.ipv4.ip_forward = 1
net.ipv4.conf.all.forwarding = 1
net.ipv4.conf.all.rp_filter = 0
net.ipv4.conf.default.rp_filter = 0
EOF

sysctl -p

# Setup monitoring
systemctl enable sysstat
systemctl start sysstat
```

## Test Result Analysis

### Success Criteria

1. Interface Management:
   - All interfaces properly cleaned up
   - No orphaned routes or addresses
   - Correct state transitions logged

2. Resource Usage:
   - No memory leaks
   - CPU usage within limits
   - File descriptor count stable

3. Performance:
   - Interface creation < 100ms
   - Cleanup completion < 1s
   - No failed cleanups

### Reporting

Generate test reports using:
```bash
./generate_test_report.sh
```

Report includes:
- Test case results
- Resource usage graphs
- Error logs
- Performance metrics

## Troubleshooting Failed Tests

1. Interface Cleanup Failures:
   ```bash
   # List interfaces
   ip link show
   
   # Check routes
   ip route show
   
   # Examine logs
   journalctl -u sssonector
   ```

2. Resource Leaks:
   ```bash
   # Monitor file descriptors
   lsof -p $(pidof sssonector)
   
   # Check memory usage
   pmap $(pidof sssonector)
   ```

3. Performance Issues:
   ```bash
   # CPU profiling
   perf record -p $(pidof sssonector)
   
   # Memory profiling
   heaptrack $(which sssonector)
