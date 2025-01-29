# SSSonector QA Testing Guide

This guide outlines the steps to test the SSSonector SSL tunnel application in both server and client modes.

## Prerequisites

1. Two machines (physical or virtual) for testing:
   - Server machine
   - Client machine
2. Root/Administrator access on both machines
3. Network connectivity between machines
4. OpenSSL installed for certificate generation

## Test Environment Setup

### Server Setup

1. Install SSSonector on the server machine:
```bash
sudo make install
```

2. Generate certificates:
```bash
make generate-certs
```

3. Copy server configuration:
```bash
sudo mkdir -p /etc/sssonector/certs
sudo cp certs/* /etc/sssonector/certs/
sudo cp configs/server.yaml /etc/sssonector/config.yaml
sudo chmod 600 /etc/sssonector/certs/*.key
sudo chmod 644 /etc/sssonector/certs/*.crt
```

### Client Setup

1. Install SSSonector on the client machine:
```bash
sudo make install
```

2. Copy certificates from server:
```bash
sudo mkdir -p /etc/sssonector/certs
sudo scp server:/etc/sssonector/certs/* /etc/sssonector/certs/
sudo chmod 600 /etc/sssonector/certs/*.key
sudo chmod 644 /etc/sssonector/certs/*.crt
```

3. Copy and modify client configuration:
```bash
sudo cp configs/client.yaml /etc/sssonector/config.yaml
```

Edit `/etc/sssonector/config.yaml` to set the correct server address.

## Test Cases

### 1. Basic Connectivity

1. Start server:
```bash
sudo sssonector -config /etc/sssonector/config.yaml
```

2. Start client:
```bash
sudo sssonector -config /etc/sssonector/config.yaml
```

Expected Results:
- Server logs show successful startup and listening
- Client logs show successful connection
- Virtual interfaces are created on both machines
- ICMP ping works between virtual interfaces

### 2. Certificate Authentication

1. Test with invalid certificates:
   - Replace client certificate with incorrect one
   - Verify connection is rejected
   - Restore correct certificate

2. Test with expired certificates:
   - Generate short-lived test certificates
   - Wait for expiration
   - Verify connection is rejected

### 3. Network Interface

1. Verify interface creation:
```bash
ip addr show tun0
```

2. Test MTU settings:
```bash
ip link show tun0
```

3. Verify routing:
```bash
ip route show
```

### 4. Bandwidth Throttling

1. Test upload throttling:
```bash
iperf3 -c <remote_ip> -p 5201
```

2. Test download throttling:
```bash
iperf3 -c <remote_ip> -p 5201 -R
```

Expected Results:
- Bandwidth should not exceed configured limits
- Both directions should be independently throttled

### 5. SNMP Monitoring

1. Test SNMP connectivity:
```bash
snmpwalk -v2c -c public localhost
```

2. Verify metrics:
- Connection status
- Bytes sent/received
- Uptime
- Current bandwidth usage

### 6. Reconnection Logic

1. Test network interruption:
```bash
sudo iptables -A INPUT -p tcp --dport 8443 -j DROP
# Wait 30 seconds
sudo iptables -D INPUT -p tcp --dport 8443 -j DROP
```

Expected Results:
- Client should attempt reconnection
- Connection should be re-established
- No data loss after reconnection

### 7. Performance Testing

1. Latency test:
```bash
ping -c 100 <remote_virtual_ip>
```

2. Throughput test:
```bash
iperf3 -c <remote_virtual_ip> -t 60
```

3. Concurrent connections (server mode):
```bash
# Start multiple clients
for i in {1..5}; do
    sudo sssonector -config client$i.yaml &
done
```

### 8. Resource Usage

Monitor resource usage during operation:
```bash
top -p $(pgrep sssonector)
```

Expected Results:
- CPU usage should be reasonable (<20% per tunnel)
- Memory usage should be stable
- No memory leaks over time

### 9. Log Verification

Check logs for proper operation:
```bash
tail -f /var/log/sssonector/server.log
tail -f /var/log/sssonector/client.log
```

Verify:
- Connection events
- Error handling
- Performance metrics
- Security events

## Troubleshooting

### Common Issues

1. Connection Failures
   - Check firewall rules
   - Verify certificate permissions
   - Confirm network connectivity
   - Check system logs

2. Performance Issues
   - Monitor system resources
   - Check network conditions
   - Verify throttling settings
   - Review MTU configuration

3. Certificate Problems
   - Verify file permissions
   - Check certificate dates
   - Confirm proper CA chain
   - Validate certificate formats

## Test Report Template

```markdown
# SSSonector Test Report

Date: YYYY-MM-DD
Version: X.Y.Z
Tester: Name

## Environment
- Server OS:
- Client OS:
- Network Configuration:
- Test Duration:

## Test Results

### Basic Connectivity
- [ ] Server startup
- [ ] Client connection
- [ ] Interface creation
- [ ] ICMP connectivity

### Security
- [ ] Certificate validation
- [ ] Authentication
- [ ] Encryption

### Performance
- [ ] Throughput
- [ ] Latency
- [ ] Resource usage
- [ ] Stability

### Monitoring
- [ ] SNMP metrics
- [ ] Logging
- [ ] Statistics

## Issues Found
1. 
2. 
3. 

## Recommendations
1. 
2. 
3. 

## Conclusion
Pass/Fail and summary
