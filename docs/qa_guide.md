# QA Testing Guide for SSSonector

This guide outlines the testing procedures for verifying SSSonector functionality.

## Test Environment Setup

### Requirements

1. Two Ubuntu VMs (follow ubuntu_install.md)
   - Server VM: 2GB RAM, 20GB storage
   - Client VM: 2GB RAM, 20GB storage
2. VirtualBox with Host-only Network configured
3. Latest SSSonector release installed on both VMs
4. iperf3 installed on both VMs for rate limiting tests
5. net-snmp-utils for SNMP monitoring

### Network Configuration

1. Verify Host-only Network:
   ```bash
   # On both VMs
   ip addr show
   # Should see two interfaces:
   # - NAT (usually enp0s3)
   # - Host-only (usually enp0s8)
   ```

2. Test VM Connectivity:
   ```bash
   # From Client VM to Server VM
   ping <server-host-only-ip>
   ```

## Test Cases

### 1. Basic Installation

#### Server VM
```bash
# Check package installation
dpkg -l | grep sssonector

# Verify file permissions
ls -l /etc/sssonector/certs/
# Should show:
# -rw------- server.key
# -rw-r--r-- server.crt
# -rw------- client.key
# -rw-r--r-- client.crt

# Check config file
cat /etc/sssonector/config.yaml
# Verify mode: "server" and correct IP/port

# Verify state directory permissions
ls -l /var/lib/sssonector/
# Should show:
# drwx------ state/
# drwxr-x--- stats/
```

#### Client VM
```bash
# Check package installation
dpkg -l | grep sssonector

# Verify certificate copy
ls -l /etc/sssonector/certs/
# Should show same permissions as server

# Check config file
cat /etc/sssonector/config.yaml
# Verify mode: "client" and correct server IP/port

# Verify state directory
ls -l /var/lib/sssonector/
# Should show same permissions as server
```

### 2. Service Management

#### Server VM
```bash
# Start service
sudo systemctl start sssonector
sudo systemctl status sssonector
# Should show: active (running)

# Check logs for state transitions
sudo journalctl -u sssonector -n 50
# Verify sequence:
# 1. StateUninitialized -> StateInitializing
# 2. StateInitializing -> StateReady
# 3. No error states or unexpected transitions

# Verify resource allocation
sudo lsof -p $(pidof sssonector)
# Check for:
# - TUN device
# - Config files
# - State files
# - Network sockets
```

#### Client VM
```bash
# Start service
sudo systemctl start sssonector
sudo systemctl status sssonector
# Should show: active (running)

# Check logs for state transitions
sudo journalctl -u sssonector -n 50
# Verify same state transition sequence as server

# Verify connection establishment
sudo journalctl -u sssonector | grep "Connection established"
# Should show successful connection with retry count
```

### 3. Rate Limiting Tests

#### Test Setup
```bash
# Install iperf3
sudo apt-get update && sudo apt-get install -y iperf3

# Start iperf3 server on Client VM
iperf3 -s -D

# Verify SNMP monitoring on Monitor VM
/usr/local/bin/sssonector-snmp throughput
```

#### Basic Rate Limiting Tests
```bash
# Test 5 Mbps limit
sudo tc qdisc del dev enp0s3 root 2>/dev/null
sudo tc qdisc add dev enp0s3 root tbf rate 5mbit burst 32kbit latency 50ms
iperf3 -c <client-ip> -t 30 -J

# Verify:
# - Actual throughput ~5.25 Mbps (includes 5% TCP overhead)
# - Stable latency
# - No packet loss
# - Connection state remains stable
# - No resource leaks
```

#### Dynamic Rate Tests
```bash
# Test multiple rates
for rate in 5 10 25 50; do
    echo "Testing ${rate}Mbps..."
    sudo tc qdisc del dev enp0s3 root 2>/dev/null
    sudo tc qdisc add dev enp0s3 root tbf rate ${rate}mbit burst 32kbit latency 50ms
    iperf3 -c <client-ip> -t 30 -J
    sleep 5
done

# Verify for each rate:
# - Actual throughput ~105% of configured rate
# - Stable performance
# - Proper burst control
# - Clean state transitions
# - Resource cleanup between tests
```

### 4. Network Interface Tests

#### Server VM
```bash
# Check TUN interface
ip addr show tun0
# Should show:
# - UP flag
# - Configured IP (10.0.0.1)
# - Correct MTU

# Verify routing
ip route show
# Should include route for 10.0.0.0/24 via tun0

# Check interface statistics
ip -s link show tun0
# Verify:
# - No packet drops
# - No errors
# - Consistent RX/TX counts
```

#### Client VM
```bash
# Check TUN interface
ip addr show tun0
# Should show:
# - UP flag
# - Configured IP (10.0.0.2)
# - Correct MTU

# Verify routing
ip route show
# Should include route for 10.0.0.0/24 via tun0

# Check interface statistics
ip -s link show tun0
# Verify same metrics as server
```

### 5. Performance Tests

```bash
# Test with different payload sizes
for size in 64 256 1024 4096; do
  ping -s $size -c 100 10.0.0.1 | tee ping_${size}.log
done

# Verify latency impact of rate limiting
# With 5 Mbps limit:
iperf3 -c 10.0.0.1 -t 30 --json > iperf_5mbps.json
jq '.end.streams[0].sender.mean_rtt' iperf_5mbps.json
# Should be < 50ms

# Monitor resource usage
while true; do
  ps -p $(pidof sssonector) -o %cpu,%mem,rss,vsz
  sleep 1
done > resource_usage.log
```

### 6. Stress Tests

```bash
# Test rate limiting under load
# Start continuous ping
ping 10.0.0.1 > ping.log &

# Run multiple iperf3 streams
for i in {1..5}; do
    iperf3 -c 10.0.0.1 -t 300 -P 4 &
done

# Monitor:
# - Rate limiting effectiveness
# - System resource usage
# - Latency impact
# - State stability
# - Connection tracking
```

### 7. Startup Logging Tests

#### Configuration Verification
```bash
# Check startup logging configuration
grep -A 5 "logging:" /etc/sssonector/config.yaml
# Should show:
#   startup_logs: true
#   level: info
#   format: json

# Verify log file permissions
ls -l /var/log/sssonector/sssonector.log
# Should be readable by sssonector user
```

#### Startup Phase Logging
```bash
# Start service with clean log
sudo rm /var/log/sssonector/sssonector.log
sudo systemctl restart sssonector

# Check startup phase sequence
grep "Entering.*phase" /var/log/sssonector/sssonector.log
# Should show in order:
# 1. PreStartup phase
# 2. Initialization phase
# 3. Connection phase
# 4. Listen phase (server only)

# Verify phase timing
grep "duration" /var/log/sssonector/sssonector.log
# Check reasonable durations for each phase
```

#### Operation Logging
```bash
# Check operation details
grep "startup_log" /var/log/sssonector/sssonector.log | jq .
# Verify for each operation:
# - Correct phase
# - Component identification
# - Operation description
# - Timing information
# - Success/failure status
# - Error details if any

# Verify critical operations
grep -A 2 "Create TUN adapter" /var/log/sssonector/sssonector.log
grep -A 2 "Create TCP listener" /var/log/sssonector/sssonector.log
grep -A 2 "Connect to server" /var/log/sssonector/sssonector.log
```

#### Resource State Tracking
```bash
# Check adapter state tracking
grep "Resource state: adapter" /var/log/sssonector/sssonector.log
# Verify:
# - State transitions
# - Interface details
# - Address configuration
# - Error conditions

# Monitor state changes
sudo systemctl restart sssonector
watch -n 1 'tail -n 50 /var/log/sssonector/sssonector.log | grep "state"'
# Verify clean state progression
```

#### Error Handling
```bash
# Test invalid configuration
sudo sed -i 's/startup_logs: true/startup_logs: invalid/' /etc/sssonector/config.yaml
sudo systemctl restart sssonector
grep "error" /var/log/sssonector/sssonector.log
# Should show validation error

# Test missing permissions
sudo chmod 000 /dev/net/tun
sudo systemctl restart sssonector
grep "error" /var/log/sssonector/sssonector.log
# Should show detailed error with context
sudo chmod 666 /dev/net/tun

# Test network failure
sudo ip link set enp0s3 down
sudo systemctl restart sssonector
grep "error" /var/log/sssonector/sssonector.log
# Should show network error details
sudo ip link set enp0s3 up
```

#### Performance Impact
```bash
# Measure startup time with logging enabled
time sudo systemctl restart sssonector
grep "duration" /var/log/sssonector/sssonector.log | jq -s 'add'

# Measure startup time with logging disabled
sudo sed -i 's/startup_logs: true/startup_logs: false/' /etc/sssonector/config.yaml
time sudo systemctl restart sssonector

# Compare resource usage
ps -p $(pidof sssonector) -o %cpu,%mem,rss,vsz --no-headers
```

#### Log Format Validation
```bash
# Verify JSON formatting
cat /var/log/sssonector/sssonector.log | while read line; do
    echo "$line" | jq . >/dev/null 2>&1 || echo "Invalid JSON: $line"
done

# Check required fields
jq -r 'select(.startup_log != null) | .startup_log | [.phase, .component, .operation, .timestamp] | @csv' /var/log/sssonector/sssonector.log
# Verify all fields present and valid

# Validate timestamps
jq -r 'select(.startup_log != null) | .startup_log.timestamp' /var/log/sssonector/sssonector.log | while read ts; do
    date -d "$ts" >/dev/null 2>&1 || echo "Invalid timestamp: $ts"
done
```

### 8. Reliability Testing

#### State Transition Verification
```bash
# Monitor state transitions
journalctl -u sssonector -f | grep "State transition:"

# Verify proper sequence:
# 1. Uninitialized -> Initializing
# 2. Initializing -> Ready
# 3. Ready -> Running
# 4. Running -> Stopping (on shutdown)
# 5. Stopping -> Stopped

# Check for error states:
journalctl -u sssonector | grep -i "error state"
# Should show no unexpected error states
```

#### Retry Mechanism Testing
```bash
# Test network interruption recovery
sudo ip link set tun0 down
sleep 5
sudo ip link set tun0 up

# Verify in logs:
# - Connection retry attempts
# - Backoff timing
# - Successful recovery
# - State consistency

# Test configuration reload
sudo systemctl reload sssonector
# Verify:
# - Clean reload
# - State preservation
# - Connection maintenance
```

#### Resource Cleanup Verification
```bash
# Check for resource leaks
watch -n 1 'sudo lsof -p $(pidof sssonector)'

# Monitor file descriptors
watch -n 1 'ls -l /proc/$(pidof sssonector)/fd'

# Check memory usage over time
ps -p $(pidof sssonector) -o pid,ppid,%cpu,%mem,cmd --forest

# Verify cleanup after restart
systemctl restart sssonector
# Check:
# - TUN interface removed
# - Sockets closed
# - File handles released
# - Memory freed
```

#### Connection Tracking
```bash
# Monitor active connections
ss -tnp | grep sssonector

# Check connection states
netstat -anp | grep sssonector

# Verify connection cleanup
systemctl stop sssonector
# Check:
# - All connections closed
# - No lingering sockets
# - Clean state file
```

#### Statistics Monitoring
```bash
# Check SNMP metrics
snmpwalk -v2c -c public localhost .1.3.6.1.4.1.54321

# Monitor throughput
iftop -i tun0

# Check interface statistics
ip -s link show tun0

# Verify metrics accuracy
# Compare reported vs actual:
# - Bandwidth usage
# - Packet counts
# - Error rates
# - Retry counts
```

## Troubleshooting Guide

### State Transition Issues

1. Stuck in Initializing State
   - Check TUN device permissions
   - Verify network interface availability
   - Check system capabilities
   - Review initialization logs

2. Unexpected Error States
   - Check system resources
   - Verify configuration
   - Review connection logs
   - Check for network issues

3. Failed State Transitions
   - Review transition sequence
   - Check for blocked operations
   - Verify resource availability
   - Check for permission issues

### Resource Cleanup Problems

1. Lingering TUN Interfaces
   ```bash
   # List interfaces
   ip link show
   # Clean up manually
   sudo ip link delete tun0
   ```

2. Stuck File Handles
   ```bash
   # List open files
   sudo lsof -p $(pidof sssonector)
   # Force close if needed
   kill -9 $(pidof sssonector)
   ```

3. Memory Leaks
   ```bash
   # Monitor memory
   ps -o pid,ppid,rss,vsize,pcpu,pmem,cmd -p $(pidof sssonector)
   # Check for growth
   ```

### Connection Tracking Errors

1. Stale Connections
   ```bash
   # List connections
   ss -tnp | grep sssonector
   # Clean up
   sudo ss -K dst <ip:port>
   ```

2. Connection State Mismatch
   ```bash
   # Compare states
   ss -tnp | grep sssonector
   netstat -anp | grep sssonector
   # Restart if inconsistent
   ```

### Statistics Monitoring Issues

1. Missing Metrics
   - Check SNMP configuration
   - Verify monitoring permissions
   - Restart statistics collection

2. Inaccurate Counts
   - Compare with system tools
   - Reset counters if needed
   - Verify collection intervals

## Test Results Documentation

For each test run, document:

1. Test environment details:
   - VM configurations
   - Network setup
   - SSSonector version
   - Rate limiting settings

2. Test results:
   - Pass/Fail status
   - Actual throughput vs configured rate
   - Latency measurements
   - TCP overhead impact
   - SNMP metrics accuracy
   - State transition sequence
   - Resource usage patterns
   - Connection tracking status

3. Issues found:
   - Description
   - Steps to reproduce
   - Performance impact
   - Metrics/logs showing the issue
   - State at time of failure
   - Resource status
   - Connection status

## Common Issues

1. Rate Limiting Issues
   - Check TCP overhead compensation
   - Verify burst size configuration
   - Monitor actual vs configured rates
   - Check for system resource constraints

2. Performance Issues
   - Check host system resources
   - Verify MTU settings
   - Monitor rate limiting impact
   - Check for competing traffic

3. Monitoring Issues
   - Verify SNMP service status
   - Check monitoring script permissions
   - Validate metric collection
   - Review update frequency

4. State Management Issues
   - Check state file permissions
   - Verify transition sequences
   - Monitor resource availability
   - Check cleanup procedures

## Reporting Bugs

When reporting issues:

1. Include environment details
2. Attach relevant logs
3. Provide exact steps to reproduce
4. Include config files (sanitized)
5. Add iperf3 test results
6. Include SNMP metrics data
7. Attach state transition logs
8. Include resource usage data
9. Provide connection tracking info
