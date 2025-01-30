# QA Testing Guide for SSSonector

This guide outlines the testing procedures for verifying SSSonector functionality.

## Test Environment Setup

### Requirements

1. Two Ubuntu VMs (follow ubuntu_install.md)
   - Server VM: 2GB RAM, 20GB storage
   - Client VM: 2GB RAM, 20GB storage
2. VirtualBox with Host-only Network configured
3. Latest SSSonector release installed on both VMs

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

### 3. Network Interface Tests

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

### 4. Connectivity Tests

```bash
# From Client VM
ping -c 4 10.0.0.1
# Should succeed

# From Server VM
ping -c 4 10.0.0.2
# Should succeed

# Test bandwidth (optional)
# On Server VM:
iperf3 -s
# On Client VM:
iperf3 -c 10.0.0.1
```

### 5. SSL/TLS Tests

```bash
# On Server VM
sudo openssl s_client -connect localhost:8443 -tls1_3
# Verify:
# - TLS 1.3 connection successful
# - Correct certificate chain
# - EU-exportable cipher suite

# Check SNMP metrics
snmpwalk -v2c -c public localhost:161 .1.3.6.1.4.1.XXXXX
# Verify SSL/TLS metrics present
```

### 6. Performance Tests

```bash
# On Client VM
# Test with different payload sizes
for size in 64 256 1024 4096; do
  ping -s $size -c 100 10.0.0.1 | tee ping_${size}.log
done

# Check bandwidth throttling
# Set throttle in config:
# throttle:
#   upload_kbps: 1024
#   download_kbps: 1024

iperf3 -c 10.0.0.1 -t 30
# Verify bandwidth doesn't exceed limits
```

### 7. Reconnection Tests

```bash
# On Server VM
sudo systemctl stop sssonector
sleep 10
sudo systemctl start sssonector

# On Client VM
# Check logs
sudo journalctl -u sssonector -f
# Verify automatic reconnection
```

### 8. Stress Tests

```bash
# On Client VM
# Run continuous ping while doing file transfer
ping 10.0.0.1 > ping.log &
dd if=/dev/zero bs=1M count=100 | nc 10.0.0.1 5000

# Check for packet loss
grep -i "packet loss" ping.log
```

### 9. Cleanup Tests

```bash
# On both VMs
sudo systemctl stop sssonector
sudo apt remove sssonector
sudo rm -rf /etc/sssonector /var/log/sssonector

# Verify
ls -la /etc/sssonector
# Should not exist
ip addr show tun0
# Should not exist
```

## Test Results Documentation

For each test run, document:

1. Test environment details:
   - VM configurations
   - Network setup
   - SSSonector version

2. Test results:
   - Pass/Fail status
   - Any errors or warnings
   - Performance metrics
   - Log snippets for issues

3. Issues found:
   - Description
   - Steps to reproduce
   - Severity level
   - Screenshots/logs

## Common Issues

1. Connection Failures
   - Check firewall rules
   - Verify certificate permissions
   - Ensure correct IP addresses in configs

2. Performance Issues
   - Check host system resources
   - Verify MTU settings
   - Check for competing network traffic

3. SSL/TLS Issues
   - Verify certificate dates
   - Check cipher suite compatibility
   - Ensure key permissions are correct

## Reporting Bugs

When reporting issues:

1. Include environment details
2. Attach relevant logs
3. Provide exact steps to reproduce
4. Include config files (sanitized)
5. Add packet captures if relevant
