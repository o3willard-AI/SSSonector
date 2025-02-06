# SSSonector Installation Guide

## Prerequisites

### System Requirements
- Linux (Ubuntu 24.04 or later recommended)
- Go 1.21 or later
- iproute2 package
- TUN/TAP kernel module support

### User Setup
1. Add your user to the `tun` group:
   ```bash
   sudo groupadd -f tun
   sudo usermod -aG tun $USER
   ```
   Note: You'll need to log out and back in for group changes to take effect.

2. Load the TUN kernel module:
   ```bash
   sudo modprobe tun
   ```

3. Set up TUN device:
   ```bash
   sudo mkdir -p /dev/net
   sudo mknod /dev/net/tun c 10 200
   sudo chmod 666 /dev/net/tun
   ```

## Installation

### From Binary
1. Download the latest release from the releases page
2. Extract the archive:
   ```bash
   tar xzf sssonector-v1.0.0.tar.gz
   ```
3. Install the binary:
   ```bash
   sudo cp sssonector /usr/local/bin/
   sudo chmod +x /usr/local/bin/sssonector
   ```

### From Source
1. Clone the repository:
   ```bash
   git clone https://github.com/o3willard-AI/SSSonector.git
   cd SSSonector
   ```

2. Build the binary:
   ```bash
   make build
   ```

3. Install:
   ```bash
   sudo cp build/sssonector /usr/local/bin/
   sudo chmod +x /usr/local/bin/sssonector
   ```

## Configuration

### Certificate Setup
1. For production use, generate permanent certificates:
   ```bash
   mkdir -p /etc/sssonector/certs
   sssonector -mode server -generate-certs-only -keyfile /etc/sssonector/certs
   ```

2. Set proper permissions:
   ```bash
   chmod 600 /etc/sssonector/certs/*.key
   chmod 644 /etc/sssonector/certs/*.crt
   ```

### Configuration File
Create a configuration file at `/etc/sssonector/config.yaml`:

```yaml
mode: "server"  # or "client"

network:
  interface: "tun0"
  address: "10.0.0.1/24"  # Use "10.0.0.2/24" for clients
  mtu: 1500

tunnel:
  cert_file: "/etc/sssonector/certs/server.crt"  # Use client.crt for clients
  key_file: "/etc/sssonector/certs/server.key"   # Use client.key for clients
  ca_file: "/etc/sssonector/certs/ca.crt"
  listen_address: "0.0.0.0"
  listen_port: 8443
  server_address: "your.server.address"  # Only needed for clients
  server_port: 8443                      # Only needed for clients
  max_clients: 10
  upload_kbps: 10240    # 10 Mbps
  download_kbps: 10240  # 10 Mbps

monitor:
  log_file: "/var/log/sssonector.log"
  snmp_enabled: false
  snmp_port: 161
  snmp_community: "public"
```

## Running

### Server Mode
```bash
sudo sssonector -mode server -config /etc/sssonector/config.yaml
```

### Client Mode
```bash
sudo sssonector -mode client -config /etc/sssonector/config.yaml
```

## Troubleshooting

### TUN Interface Issues
1. Verify TUN module is loaded:
   ```bash
   lsmod | grep tun
   ```

2. Check TUN device permissions:
   ```bash
   ls -l /dev/net/tun
   ```

3. Verify user is in tun group:
   ```bash
   groups $USER | grep tun
   ```

### Process Cleanup
If the process doesn't exit cleanly:
```bash
sudo pkill -9 -f sssonector
```

### Certificate Issues
1. Verify certificate permissions:
   ```bash
   ls -l /etc/sssonector/certs/
   ```

2. Check certificate expiration:
   ```bash
   openssl x509 -in /etc/sssonector/certs/server.crt -text -noout | grep "Not After"
   ```

### Network Issues
1. Check interface status:
   ```bash
   ip addr show tun0
   ```

2. Verify routing:
   ```bash
   ip route show
   ```

3. Test connectivity:
   ```bash
   ping 10.0.0.1  # From client to server
   ```

## Monitoring

### Logs
Monitor the log file for issues:
```bash
tail -f /var/log/sssonector.log
```

### Process Status
Check process status:
```bash
ps aux | grep sssonector
```

### Network Statistics
View interface statistics:
```bash
ip -s link show tun0
```

## Best Practices

1. **Security**
   - Keep certificates in a secure location
   - Use proper file permissions
   - Regularly rotate certificates
   - Monitor logs for unauthorized access attempts

2. **Performance**
   - Adjust MTU based on network conditions
   - Monitor bandwidth usage
   - Configure rate limits appropriately

3. **Maintenance**
   - Regularly check for updates
   - Monitor system resources
   - Keep logs rotated
   - Clean up temporary files

## Development Mode

For testing and development, temporary certificates can be used:

```bash
sudo sssonector -mode server -test-without-certs -config config.yaml
```

Note: Temporary certificates expire after 15 seconds and should never be used in production.
