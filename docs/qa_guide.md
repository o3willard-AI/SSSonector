# SSSonector QA Guide

This guide outlines the testing procedures for SSSonector in both server and client modes.

## Test Environment Setup

### Requirements
- Two machines (physical or virtual) running supported OS (Linux, macOS, or Windows)
- Network connectivity between machines
- Administrative/root access on both machines
- OpenSSL for certificate verification

### Initial Setup

1. Install SSSonector on both machines:
   ```bash
   # On Debian/Ubuntu
   sudo dpkg -i sssonector_1.0.0_amd64.deb
   
   # On macOS
   sudo installer -pkg SSSonector-1.0.0-macos.pkg -target /
   
   # On Windows
   # Run SSSonector-1.0.0-windows-amd64.exe as administrator
   ```

2. Generate test certificates:
   ```bash
   ./scripts/generate-certs.sh
   ```

3. Copy certificates to both machines:
   ```bash
   # On Server
   sudo mkdir -p /etc/sssonector/certs
   sudo cp certs/ca.crt certs/server.crt certs/server.key /etc/sssonector/certs/
   sudo chmod 600 /etc/sssonector/certs/*.key
   sudo chmod 644 /etc/sssonector/certs/*.crt

   # On Client
   sudo mkdir -p /etc/sssonector/certs
   sudo cp certs/ca.crt certs/client.crt certs/client.key /etc/sssonector/certs/
   sudo chmod 600 /etc/sssonector/certs/*.key
   sudo chmod 644 /etc/sssonector/certs/*.crt
   ```

## Test Cases

### 1. Basic Connectivity

#### Server Setup
1. Configure server:
   ```bash
   sudo vi /etc/sssonector/config.yaml
   # Set mode: "server"
   # Set network.address: "10.0.0.1"
   # Set tunnel.listen_address: "0.0.0.0"
   # Set tunnel.listen_port: "8443"
   ```

2. Start server:
   ```bash
   sudo systemctl start sssonector
   ```

3. Verify server is listening:
   ```bash
   sudo netstat -tulpn | grep sssonector
   ```

#### Client Setup
1. Configure client:
   ```bash
   sudo vi /etc/sssonector/config.yaml
   # Set mode: "client"
   # Set network.address: "10.0.0.2"
   # Set tunnel.server_address: "<server_ip>"
   # Set tunnel.server_port: "8443"
   ```

2. Start client:
   ```bash
   sudo systemctl start sssonector
   ```

#### Test Steps
1. Check interface creation:
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

3. Verify TLS connection:
   ```bash
   # On server
   sudo tcpdump -i any port 8443 -vv
   ```

### 2. Bandwidth Throttling

1. Configure bandwidth limit:
   ```yaml
   tunnel:
     bandwidth_limit: 1048576  # 1 MB/s
   ```

2. Test file transfer:
   ```bash
   # On server
   dd if=/dev/zero bs=1M count=100 | nc -l 5000

   # On client
   nc 10.0.0.1 5000 > /dev/null
   ```

3. Verify transfer rate stays within limit:
   ```bash
   iftop -i tun0
   ```

### 3. SNMP Monitoring

1. Install SNMP utilities:
   ```bash
   sudo apt-get install snmp snmpd
   ```

2. Test SNMP metrics:
   ```bash
   snmpwalk -v2c -c public localhost:161
   ```

3. Verify metrics:
   - Bytes in/out
   - Active connections
   - System stats

### 4. Reconnection Logic

1. Test automatic reconnection:
   ```bash
   # Stop server
   sudo systemctl stop sssonector

   # Wait 30 seconds
   sleep 30

   # Start server
   sudo systemctl start sssonector
   ```

2. Verify client reconnects:
   ```bash
   sudo journalctl -u sssonector -f
   ```

### 5. Certificate Validation

1. Test invalid certificate:
   ```bash
   # Backup original cert
   sudo cp /etc/sssonector/certs/client.crt{,.bak}

   # Replace with invalid cert
   sudo cp invalid.crt /etc/sssonector/certs/client.crt

   # Restart client
   sudo systemctl restart sssonector
   ```

2. Verify connection failure:
   ```bash
   sudo journalctl -u sssonector -f
   ```

### 6. Load Testing

1. Setup multiple clients:
   ```bash
   # Clone VM or setup additional machines
   # Configure each with unique IP (10.0.0.3, 10.0.0.4, etc.)
   ```

2. Monitor server performance:
   ```bash
   # CPU/Memory usage
   top -p $(pgrep sssonector)

   # Network stats
   iftop -i tun0

   # Connection count
   netstat -an | grep 8443 | wc -l
   ```

## Test Results Documentation

For each test case, document:
1. Test environment details
2. Steps performed
3. Expected results
4. Actual results
5. Any errors or unexpected behavior
6. System logs
7. Network captures (if relevant)

## Common Issues

### TUN/TAP Device Creation
- Verify kernel module: `lsmod | grep tun`
- Check permissions: `ls -l /dev/net/tun`
- Verify user/group: `id sssonector`

### Certificate Problems
- Check permissions: `ls -l /etc/sssonector/certs/`
- Verify certificate chain: `openssl verify -CAfile ca.crt server.crt`
- Check expiration: `openssl x509 -in server.crt -noout -dates`

### Network Issues
- Check firewall rules: `sudo iptables -L`
- Verify routing: `ip route show`
- Test raw connectivity: `nc -zv server_ip 8443`
