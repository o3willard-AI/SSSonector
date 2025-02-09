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
```

### 2. Service Management

#### Server VM
```bash
# Start service
sudo systemctl start sssonector
sudo systemctl status sssonector
# Should show: active (running)

# Check logs
sudo journalctl -u sssonector -n 50
# Verify: No errors, listening on configured port
```

#### Client VM
```bash
# Start service
sudo systemctl start sssonector
sudo systemctl status sssonector
# Should show: active (running)

# Check logs
sudo journalctl -u sssonector -n 50
# Verify: Connected to server successfully
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
```

#### Monitoring Verification
```bash
# Monitor throughput during tests
watch -n 1 '/usr/local/bin/sssonector-snmp throughput'

# Check rate limiting metrics
snmpwalk -v2c -c public localhost:10161 .1.3.6.1.4.1.54321.1.3

# Verify:
# - Metrics update in real-time
# - Values match iperf3 results
# - TCP overhead is accounted for
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
```

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

3. Issues found:
   - Description
   - Steps to reproduce
   - Performance impact
   - Metrics/logs showing the issue

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

## Reporting Bugs

When reporting issues:

1. Include environment details
2. Attach relevant logs
3. Provide exact steps to reproduce
4. Include config files (sanitized)
5. Add iperf3 test results
6. Include SNMP metrics data
