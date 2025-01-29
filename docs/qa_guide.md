# SSSonector QA Guide

This guide provides instructions for testing the SSSonector application in both server and client modes.

## Test Environment Setup

### Requirements

- Two VMs (one for server, one for client)
- Each VM needs:
  - Linux/macOS/Windows
  - Go 1.21 or later
  - Root/Administrator access
  - Network connectivity between VMs

### Recommended VM Configuration

#### Server VM
- 2 CPU cores
- 2GB RAM
- 20GB disk
- Two network interfaces:
  - NAT (for internet access)
  - Host-only network (for client connectivity)

#### Client VM
- 2 CPU cores
- 2GB RAM
- 20GB disk
- Two network interfaces:
  - NAT (for internet access)
  - Host-only network (for server connectivity)

## Test Setup

1. Clone the repository on both VMs:
   ```bash
   git clone https://github.com/yourusername/SSSonector.git
   cd SSSonector
   ```

2. Build the application on both VMs:
   ```bash
   go build -v cmd/tunnel/main.go
   ```

3. Create test directories:
   ```bash
   sudo mkdir -p /var/log/SSSonector
   sudo mkdir -p certs
   ```

## Test Cases

### 1. Basic Connectivity

#### Server Setup
1. Edit configs/server.yaml:
   ```yaml
   mode: "server"
   network:
     interface: "tun0"
     address: "10.0.0.1"
     listen_address: "0.0.0.0"
     listen_port: 5000
   ```

2. Start server:
   ```bash
   sudo ./main --config configs/server.yaml
   ```

3. Verify server is listening:
   ```bash
   netstat -tulpn | grep 5000
   ```

#### Client Setup
1. Edit configs/client.yaml:
   ```yaml
   mode: "client"
   network:
     interface: "tun0"
     address: "10.0.0.2"
     server_address: "<server-ip>"
     server_port: 5000
   ```

2. Start client:
   ```bash
   sudo ./main --config configs/client.yaml
   ```

#### Test Steps
1. Verify interface creation:
   ```bash
   ip addr show tun0
   ```

2. Test connectivity:
   ```bash
   # From server
   ping 10.0.0.2

   # From client
   ping 10.0.0.1
   ```

### 2. Certificate Management

#### Test Certificate Generation and Management
1. Run the certificate test script:
   ```bash
   # Basic test with default settings
   ./scripts/test-certs.sh

   # Custom test configuration
   ./scripts/test-certs.sh \
     --dir /path/to/certs \
     --days 365 \
     --cn tunnel.example.com \
     --ips "10.0.0.1,192.168.1.1" \
     --dns "tunnel.local,tunnel.internal"
   ```

2. Verify test results:
   - Certificate generation success
   - Certificate validity period
   - Subject and SAN fields
   - Private key match
   - Certificate chain verification
   - TLS connection test

#### Test Auto-generation
1. Delete existing certificates:
   ```bash
   rm -f certs/*
   ```

2. Start server with auto-generate enabled:
   ```bash
   sudo ./main --config configs/server.yaml
   ```

3. Verify certificates:
   ```bash
   # List generated files
   ls -l certs/

   # Inspect certificate details
   openssl x509 -in certs/server.crt -text -noout
   ```

#### Test Certificate Expiry Warning
1. Create test certificate with short validity:
   ```bash
   ./scripts/test-certs.sh --days 2
   ```

2. Monitor logs for expiry warnings:
   ```bash
   # Run server and watch logs
   sudo ./main --config configs/server.yaml 2>&1 | grep -i "certificate"
   ```

3. Verify auto-renewal:
   ```bash
   # Get initial certificate expiry
   EXPIRY1=$(openssl x509 -in certs/server.crt -enddate -noout)
   
   # Wait for auto-renewal (after warning)
   sleep 2h
   
   # Get new certificate expiry
   EXPIRY2=$(openssl x509 -in certs/server.crt -enddate -noout)
   
   # Compare expiry dates
   [ "$EXPIRY1" != "$EXPIRY2" ] && echo "Certificate was renewed"
   ```

#### Test Certificate Chain
1. Create CA and server certificates:
   ```bash
   # Generate CA
   openssl req -x509 -new -nodes -key ca.key -sha256 -days 1024 -out ca.crt

   # Generate server CSR
   openssl req -new -key server.key -out server.csr

   # Sign server certificate
   openssl x509 -req -in server.csr -CA ca.crt -CAkey ca.key -CAcreateserial \
     -out server.crt -days 365 -sha256
   ```

2. Verify certificate chain:
   ```bash
   openssl verify -CAfile ca.crt server.crt
   ```

3. Test with client:
   ```bash
   # Configure client to use CA certificate
   cp ca.crt configs/client/ca.crt
   
   # Start client and verify connection
   sudo ./main --config configs/client.yaml
   ```

#### Security Testing
1. Test with invalid certificates:
   ```bash
   # Try connecting with expired certificate
   ./scripts/test-certs.sh --days -1
   
   # Try connecting with mismatched key
   openssl genpkey -algorithm RSA -out wrong.key
   ```

2. Test certificate permissions:
   ```bash
   # Check file permissions
   stat -c "%a %n" certs/*
   
   # Try running with wrong permissions
   chmod 666 certs/server.key
   sudo ./main --config configs/server.yaml
   ```

3. Test certificate validation:
   ```bash
   # Test with self-signed certificate
   openssl req -x509 -newkey rsa:4096 -keyout test.key -out test.crt -days 365
   
   # Try connecting with untrusted certificate
   cp test.crt configs/client/ca.crt
   sudo ./main --config configs/client.yaml
   ```

### 3. Performance Testing

#### Bandwidth Test
1. Install iperf3 on both VMs
2. Start iperf3 server on server VM:
   ```bash
   iperf3 -s
   ```

3. Run test from client VM:
   ```bash
   iperf3 -c 10.0.0.1
   ```

#### Throttling Test
1. Enable throttling in config:
   ```yaml
   throttle:
     enabled: true
     upload_kbps: 1024
     down_kbps: 1024
   ```

2. Repeat iperf3 test and verify speeds are limited

### 4. High Availability

#### Connection Retry
1. Start client with retry config:
   ```yaml
   network:
     retry_attempts: 3
     retry_interval: 5
   ```

2. Stop server and observe client retry behavior
3. Verify retry count matches configuration
4. Start server and verify automatic reconnection

#### Multiple Clients
1. Configure server with max_clients:
   ```yaml
   network:
     max_clients: 2
   ```

2. Connect multiple clients
3. Verify connection limit is enforced

### 5. Monitoring

#### SNMP Testing
1. Install SNMP tools:
   ```bash
   # Ubuntu/Debian
   apt-get install snmp snmpd

   # RHEL/CentOS
   yum install net-snmp-utils

   # macOS
   brew install net-snmp
   ```

2. Load MIB definitions:
   ```bash
   # Copy MIB file to system MIB directory
   sudo cp mibs/SSL-TUNNEL-MIB.txt /usr/share/snmp/mibs/  # Linux
   sudo cp mibs/SSL-TUNNEL-MIB.txt /usr/local/share/snmp/mibs/  # macOS
   ```

3. Test SNMP metrics using the provided script:
   ```bash
   # Basic usage (defaults to localhost:161 with community string 'public')
   ./scripts/test-snmp.sh

   # Test remote server
   ./scripts/test-snmp.sh -h tunnel.example.com -p 161 -c mycommunity
   ```

4. Verify available metrics:
   - Bytes Received (Counter64)
   - Bytes Sent (Counter64)
   - Packets Lost (Counter64)
   - Latency (Integer32, microseconds)
   - Uptime (Integer32, seconds)
   - CPU Usage (DisplayString, percentage)
   - Memory Usage (DisplayString, percentage)

5. Manual SNMP testing:
   ```bash
   # Query specific metrics
   snmpget -v2c -c public localhost:161 .1.3.6.1.4.1.2021.10.1.3.1  # Bytes Received
   snmpget -v2c -c public localhost:161 .1.3.6.1.4.1.2021.10.1.3.2  # Bytes Sent
   snmpget -v2c -c public localhost:161 .1.3.6.1.4.1.2021.10.1.3.3  # Packets Lost
   snmpget -v2c -c public localhost:161 .1.3.6.1.4.1.2021.10.1.3.4  # Latency
   snmpget -v2c -c public localhost:161 .1.3.6.1.4.1.2021.10.1.3.5  # Uptime
   snmpget -v2c -c public localhost:161 .1.3.6.1.4.1.2021.10.1.3.6  # CPU Usage
   snmpget -v2c -c public localhost:161 .1.3.6.1.4.1.2021.10.1.3.7  # Memory Usage

   # Walk all metrics
   snmpwalk -v2c -c public localhost:161 .1.3.6.1.4.1.2021.10.1.3
   ```

6. Integration with monitoring systems:
   - The SNMP agent supports standard SNMP v2c queries
   - Compatible with monitoring systems like:
     * Nagios/Icinga
     * Zabbix
     * Cacti
     * PRTG
     * SolarWinds
   - Use the provided MIB file to configure monitoring templates

### 6. Service Integration

#### Systemd (Linux)
1. Install service:
   ```bash
   sudo cp scripts/service/systemd/SSSonector.service /etc/systemd/system/
   sudo systemctl daemon-reload
   sudo systemctl enable SSSonector
   ```

2. Test service operations:
   ```bash
   sudo systemctl start SSSonector
   sudo systemctl status SSSonector
   sudo systemctl stop SSSonector
   ```

3. Verify auto-start on boot

## Test Validation Checklist

- [ ] Virtual interfaces created successfully
- [ ] TLS connection established
- [ ] Bi-directional ping successful
- [ ] Bandwidth throttling effective
- [ ] Certificate auto-generation working
- [ ] Certificate expiry warnings visible
- [ ] Connection retry functioning
- [ ] Multiple client limits enforced
- [ ] SNMP metrics available
- [ ] Service integration working
- [ ] Logs properly rotated
- [ ] QoS tags applied correctly

## Common Issues and Solutions

1. Interface Creation Fails
   - Check kernel modules: `lsmod | grep tun`
   - Verify TUN/TAP support: `ls -l /dev/net/tun`

2. Connection Failures
   - Check firewall rules
   - Verify IP addresses and ports
   - Check TLS certificate paths

3. Permission Issues
   - Verify running as root/admin
   - Check file permissions
   - Verify service user permissions

## Test Report Template

```markdown
# SSSonector Test Report

Date: YYYY-MM-DD
Version Tested: X.Y.Z
Environment: [OS/Version]

## Test Results
1. Basic Connectivity: [PASS/FAIL]
2. Certificate Management: [PASS/FAIL]
3. Performance Testing: [PASS/FAIL]
4. High Availability: [PASS/FAIL]
5. Monitoring: [PASS/FAIL]
6. Service Integration: [PASS/FAIL]

## Issues Found
1. [Issue description]
   - Severity: [High/Medium/Low]
   - Steps to reproduce
   - Expected vs actual behavior

## Recommendations
- [List any recommendations for improvement]

## Notes
- [Additional observations]
