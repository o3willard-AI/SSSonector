# SSSonector Troubleshooting Guide

## Overview
This guide provides comprehensive troubleshooting information for common issues, error messages, and diagnostic procedures for SSSonector.

## Table of Contents
1. [Quick Diagnostics](#quick-diagnostics)
2. [Common Issues](#common-issues)
3. [Error Messages](#error-messages)
4. [System Checks](#system-checks)
5. [Advanced Diagnostics](#advanced-diagnostics)
6. [Recovery Procedures](#recovery-procedures)

## Quick Diagnostics

### Health Check Script
```bash
#!/bin/bash
# Quick health check for SSSonector

echo "=== SSSonector Health Check ==="

# Check service status
echo "Checking service status..."
systemctl status sssonector

# Check TUN interface
echo "Checking TUN interface..."
ip addr show tun0

# Check connectivity
echo "Checking connectivity..."
ping -c 3 10.0.0.1

# Check logs
echo "Checking recent logs..."
tail -n 20 /var/log/sssonector/sssonector.log

# Check resource usage
echo "Checking resource usage..."
ps aux | grep sssonector
```

### Status Commands
```bash
# Basic status
sssonector status

# Detailed status
sssonector status --verbose

# Connection metrics
sssonector metrics

# Configuration check
sssonector validate --config /etc/sssonector/config.yaml
```

## Common Issues

### Connection Problems

1. **Connection Timeout**
```yaml
symptoms:
  - Connection attempts fail
  - Timeout errors in logs
  - No response from server

checks:
  - Verify server is running
  - Check firewall rules
  - Validate network connectivity
  - Confirm port availability

solutions:
  # Check server status
  systemctl status sssonector-server
  
  # Verify firewall rules
  sudo iptables -L | grep 8080
  
  # Test network connectivity
  telnet server_ip 8080
  
  # Check port availability
  netstat -tuln | grep 8080
```

2. **TUN Interface Issues**
```yaml
symptoms:
  - TUN device creation fails
  - Network unreachable errors
  - Routing problems

checks:
  - Verify TUN module is loaded
  - Check interface permissions
  - Validate routing table
  - Confirm network configuration

solutions:
  # Load TUN module
  sudo modprobe tun
  
  # Check TUN status
  lsmod | grep tun
  
  # Fix permissions
  sudo chmod 666 /dev/net/tun
  
  # Verify routing
  ip route show
```

3. **Certificate Problems**
```yaml
symptoms:
  - TLS handshake failures
  - Certificate validation errors
  - Authentication failures

checks:
  - Verify certificate validity
  - Check certificate paths
  - Validate certificate chain
  - Confirm permissions

solutions:
  # Check certificate validity
  openssl verify -CAfile ca.crt server.crt
  
  # Verify certificate dates
  openssl x509 -in server.crt -noout -dates
  
  # Check permissions
  ls -l /etc/sssonector/certs/
  
  # Validate configuration
  sssonector validate --config config.yaml
```

### Performance Issues

1. **High Latency**
```yaml
symptoms:
  - Slow connection speed
  - High ping times
  - Packet delays

checks:
  - Monitor network latency
  - Check system resources
  - Verify buffer sizes
  - Analyze traffic patterns

solutions:
  # Monitor latency
  ping -c 10 10.0.0.1
  
  # Check system load
  top -n 1
  
  # Adjust buffer sizes
  sysctl -w net.core.rmem_max=4194304
  sysctl -w net.core.wmem_max=4194304
  
  # Analyze traffic
  tcpdump -i tun0
```

2. **Resource Exhaustion**
```yaml
symptoms:
  - High CPU usage
  - Memory depletion
  - File descriptor limits
  - System slowdown

checks:
  - Monitor resource usage
  - Check system limits
  - Verify configuration
  - Analyze processes

solutions:
  # Check resource usage
  ps aux | grep sssonector
  
  # Monitor system metrics
  vmstat 1
  
  # Adjust limits
  ulimit -n 65535
  
  # Review configuration
  cat /etc/sssonector/config.yaml
```

3. **Throughput Problems**
```yaml
symptoms:
  - Low bandwidth
  - Poor transfer rates
  - Network congestion
  - Buffer overflows

checks:
  - Test network speed
  - Monitor throughput
  - Check MTU settings
  - Verify rate limits

solutions:
  # Test throughput
  iperf3 -c 10.0.0.1
  
  # Check MTU
  ip link show tun0
  
  # Monitor traffic
  iftop -i tun0
  
  # Adjust rate limits
  vim /etc/sssonector/config.yaml
```

## Error Messages

### Common Error Codes

1. **ERR_CONN_REFUSED**
```yaml
description: Connection refused by remote host
causes:
  - Server not running
  - Firewall blocking
  - Wrong port/address
  - Service crashed

resolution:
  # Check server status
  systemctl status sssonector-server
  
  # Verify firewall
  sudo iptables -L
  
  # Check logs
  tail -f /var/log/sssonector/sssonector.log
```

2. **ERR_CERT_INVALID**
```yaml
description: Certificate validation failed
causes:
  - Expired certificate
  - Wrong CA chain
  - Invalid certificate
  - Permission issues

resolution:
  # Check certificate
  openssl verify -CAfile ca.crt server.crt
  
  # Verify dates
  openssl x509 -in server.crt -noout -dates
  
  # Check permissions
  ls -l /etc/sssonector/certs/
```

3. **ERR_TUN_FAILED**
```yaml
description: TUN device creation failed
causes:
  - Missing permissions
  - Module not loaded
  - Device busy
  - System limits

resolution:
  # Load module
  sudo modprobe tun
  
  # Fix permissions
  sudo chmod 666 /dev/net/tun
  
  # Check devices
  ls -l /dev/net/tun
```

## System Checks

### Network Diagnostics
```bash
# Interface status
ip addr show
ip link show

# Routing table
ip route show
ip route get 10.0.0.1

# Connection tracking
conntrack -L
ss -tuln

# DNS resolution
dig server.example.com
host server.example.com
```

### Resource Monitoring
```bash
# CPU usage
top -bn1
mpstat 1 5

# Memory usage
free -m
vmstat 1

# File descriptors
lsof -p $(pgrep sssonector)
ulimit -n
```

### Log Analysis
```bash
# Error patterns
grep ERROR /var/log/sssonector/sssonector.log

# Warning patterns
grep WARN /var/log/sssonector/sssonector.log

# Connection events
grep "connection" /var/log/sssonector/sssonector.log

# Full debug
tail -f /var/log/sssonector/debug.log
```

## Advanced Diagnostics

### Network Profiling
```bash
# Packet capture
tcpdump -i tun0 -w capture.pcap

# Traffic analysis
wireshark capture.pcap

# Bandwidth monitoring
iftop -i tun0

# Connection tracking
netstat -np | grep sssonector
```

### Performance Profiling
```bash
# CPU profiling
go tool pprof http://localhost:6060/debug/pprof/profile

# Memory profiling
go tool pprof http://localhost:6060/debug/pprof/heap

# Goroutine analysis
go tool pprof http://localhost:6060/debug/pprof/goroutine
```

### Security Auditing
```bash
# Certificate check
openssl s_client -connect server:8080

# TLS version check
nmap --script ssl-enum-ciphers -p 8080 server

# Port scanning
nmap -sS -sV -p- server

# Security logs
grep "security" /var/log/sssonector/sssonector.log
```

## Recovery Procedures

### Service Recovery
```bash
# Stop service
systemctl stop sssonector

# Clean up resources
rm -f /var/run/sssonector.pid
ip link delete tun0

# Reset state
rm -f /var/lib/sssonector/state.db

# Start service
systemctl start sssonector
```

### Configuration Recovery
```bash
# Backup current config
cp /etc/sssonector/config.yaml /etc/sssonector/config.yaml.bak

# Restore from backup
cp /etc/sssonector/config.yaml.bak /etc/sssonector/config.yaml

# Validate config
sssonector validate --config /etc/sssonector/config.yaml

# Apply changes
systemctl restart sssonector
```

### Network Recovery
```bash
# Reset TUN interface
ip link delete tun0
ip tuntap add name tun0 mode tun
ip link set tun0 up

# Reset routing
ip route flush table main
ip route add default via 192.168.1.1

# Reset firewall
iptables-restore < /etc/iptables/rules.v4

# Verify connectivity
ping -c 3 10.0.0.1
```

## Best Practices

### Preventive Measures
1. Regular health checks
2. Automated monitoring
3. Backup configurations
4. Update certificates
5. Resource monitoring

### Maintenance Schedule
1. Daily log review
2. Weekly backup
3. Monthly certificate check
4. Quarterly security audit

### Documentation
1. Keep error logs
2. Document changes
3. Track incidents
4. Update procedures

## Quick Reference

### Common Commands
```bash
# Service control
systemctl status sssonector
systemctl restart sssonector
systemctl stop sssonector

# Log viewing
tail -f /var/log/sssonector/sssonector.log
journalctl -u sssonector

# Configuration
sssonector validate --config config.yaml
sssonector reload --config config.yaml

# Diagnostics
sssonector status
sssonector metrics
sssonector debug
```

### Important Files
```bash
# Configuration
/etc/sssonector/config.yaml

# Certificates
/etc/sssonector/certs/

# Logs
/var/log/sssonector/

# State
/var/lib/sssonector/
```

### System Limits
```bash
# File descriptors
ulimit -n 65535

# Process limits
ulimit -u 65535

# Memory limits
sysctl -w vm.max_map_count=262144

# Network limits
sysctl -w net.core.rmem_max=4194304
