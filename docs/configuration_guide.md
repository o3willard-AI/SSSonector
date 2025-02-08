# SSSonector Configuration Guide

This guide provides detailed information about configuring SSSonector, including complete examples and explanations for both server and client setups.

## Configuration File Structure

SSSonector uses YAML configuration files with the following main sections:
- `mode`: Determines if running as server or client
- `network`: Network interface configuration
- `tunnel`: SSL tunnel and certificate settings
- `monitor`: Monitoring and SNMP configuration
- `throttle`: Rate limiting settings

## Basic Configuration Examples

### Server Configuration
```yaml
mode: "server"

network:
  interface: "tun0"
  address: "10.0.0.1/24"  # Server uses first address in tunnel network
  mtu: 1500               # Standard MTU, adjust if needed

tunnel:
  cert_file: "/etc/sssonector/certs/server.crt"
  key_file: "/etc/sssonector/certs/server.key"
  ca_file: "/etc/sssonector/certs/ca.crt"
  listen_address: "0.0.0.0"  # Listen on all interfaces
  listen_port: 8443
  max_clients: 10
  upload_kbps: 10240     # 10 Mbps upload limit
  download_kbps: 10240   # 10 Mbps download limit

monitor:
  enabled: true
  log_file: "/var/log/sssonector/server.log"
  snmp_enabled: true
  snmp_address: "0.0.0.0"
  snmp_port: 10161       # Non-standard port to avoid conflicts
  snmp_community: "public"
  snmp_version: "2c"
  update_interval: 30

throttle:
  enabled: true
  rate_limit: 1000000    # 1 MB/s
  burst_limit: 2000000   # 2 MB/s burst
```

### Client Configuration
```yaml
mode: "client"

network:
  interface: "tun0"
  address: "10.0.0.2/24"  # Client uses unique address in tunnel network
  mtu: 1500

tunnel:
  cert_file: "/etc/sssonector/certs/client.crt"
  key_file: "/etc/sssonector/certs/client.key"
  ca_file: "/etc/sssonector/certs/ca.crt"
  server_address: "192.168.50.210"  # Server's public IP
  server_port: 8443
  upload_kbps: 10240
  download_kbps: 10240

monitor:
  enabled: true
  log_file: "/var/log/sssonector/client.log"
  snmp_enabled: true
  snmp_address: "0.0.0.0"
  snmp_port: 10162       # Different port from server
  snmp_community: "public"
  snmp_version: "2c"
  update_interval: 30

throttle:
  enabled: true
  rate_limit: 1000000
  burst_limit: 2000000
```

## Section Details

### Network Configuration
- `interface`: Name of the TUN interface to create
- `address`: IP address/netmask for the tunnel interface
- `mtu`: Maximum Transmission Unit (default: 1500)

### Tunnel Configuration
- `cert_file`, `key_file`, `ca_file`: Paths to SSL certificates
  * Can be absolute paths or relative to config directory
  * Default location: /etc/sssonector/certs/
- `listen_address`, `listen_port`: Server listening settings
- `server_address`, `server_port`: Client connection settings
- `max_clients`: Maximum concurrent client connections (server only)
- `upload_kbps`, `download_kbps`: Bandwidth limits in Kbps

### Monitor Configuration
- `enabled`: Enable/disable monitoring
- `log_file`: Path to monitoring log file
- `snmp_enabled`: Enable SNMP monitoring
- `snmp_address`: SNMP listening address
- `snmp_port`: SNMP port (default: 161)
- `snmp_community`: SNMP community string
- `snmp_version`: SNMP version (supported: 1, 2c)
- `update_interval`: Metrics update interval in seconds

### Throttle Configuration
- `enabled`: Enable/disable rate limiting
- `rate_limit`: Sustained rate limit in bytes/sec
- `burst_limit`: Burst rate limit in bytes/sec

## Path Resolution Rules

1. Certificate paths:
   - Absolute paths are used as-is
   - Relative paths are resolved from config file location
   - Default paths use /etc/sssonector/certs/

2. Log files:
   - Absolute paths are used as-is
   - Relative paths are resolved from current directory
   - Default location: /var/log/sssonector/

## Common Issues and Solutions

1. Certificate Loading Fails
   ```
   Problem: "failed to load certificate"
   Solution: Ensure paths are correct and files have proper permissions (600)
   ```

2. Network Interface Creation Fails
   ```
   Problem: "failed to create tun device"
   Solution: Run with sufficient privileges (root/sudo)
   ```

3. SNMP Binding Fails
   ```
   Problem: "failed to bind SNMP agent"
   Solution: Check if port is available, may need to change port number
   ```

## Tested Configurations

The following configurations have been validated in our QA environment:

1. Basic Server-Client Setup
   - Server (192.168.50.210):
     ```yaml
     mode: "server"
     network:
       interface: "tun0"
       address: "10.0.0.1/24"
       mtu: 1500
     tunnel:
       cert_file: "/etc/sssonector/certs/server.crt"
       key_file: "/etc/sssonector/certs/server.key"
       ca_file: "/etc/sssonector/certs/ca.crt"
       listen_address: "0.0.0.0"
       listen_port: 8443
       max_clients: 10
       upload_kbps: 10240
       download_kbps: 10240
     ```

   - Client (192.168.50.211):
     ```yaml
     mode: "client"
     network:
       interface: "tun0"
       address: "10.0.0.2/24"
       mtu: 1500
     tunnel:
       cert_file: "/etc/sssonector/certs/client.crt"
       key_file: "/etc/sssonector/certs/client.key"
       ca_file: "/etc/sssonector/certs/ca.crt"
       server_address: "192.168.50.210"
       server_port: 8443
       upload_kbps: 10240
       download_kbps: 10240
     ```

2. Monitoring Setup
   - Server SNMP Configuration:
     ```yaml
     monitor:
       enabled: true
       log_file: "/var/log/sssonector/server.log"
       snmp_enabled: true
       snmp_address: "0.0.0.0"
       snmp_port: 10161
       snmp_community: "public"
       snmp_version: "2c"
       update_interval: 30
     ```

   - Client SNMP Configuration:
     ```yaml
     monitor:
       enabled: true
       log_file: "/var/log/sssonector/client.log"
       snmp_enabled: true
       snmp_address: "0.0.0.0"
       snmp_port: 10162
       snmp_community: "public"
       snmp_version: "2c"
       update_interval: 30
     ```

3. Rate Limiting Configuration
   - Both server and client:
     ```yaml
     throttle:
       enabled: true
       rate_limit: 1000000    # 1 MB/s
       burst_limit: 2000000   # 2 MB/s burst
     ```

## Configuration Testing

Before deploying, validate your configuration:

1. Test configuration syntax:
   ```bash
   sssonector -validate-config /etc/sssonector/config.yaml
   ```

2. Test with temporary certificates (useful for initial setup):
   ```bash
   # On server
   sudo sssonector -test-without-certs -config /etc/sssonector/config.yaml

   # On client (after server is running)
   sudo sssonector -test-without-certs -config /etc/sssonector/config.yaml
   ```

3. Validate certificate setup:
   ```bash
   sssonector -validate-certs -config /etc/sssonector/config.yaml
   ```

4. Verify SNMP monitoring:
   ```bash
   # Check SNMP agent status
   snmpwalk -v2c -c public localhost:10161 .1.3.6.1.4.1.54321

   # Monitor metrics in real-time
   snmptrapd -f -Lo -c /etc/snmp/snmptrapd.conf
   ```

## Best Practices

1. Security
   - Use absolute paths for certificates
   - Set restrictive file permissions (600 for keys)
   - Change default SNMP community strings
   - Use non-standard ports when possible

2. Performance
   - Set appropriate MTU for your network
   - Configure rate limits based on available bandwidth
   - Adjust burst limits for better performance
   - Monitor system metrics for optimization

3. Monitoring
   - Enable detailed logging in production
   - Configure SNMP monitoring for better visibility
   - Set appropriate update intervals
   - Use ntopng for traffic analysis

4. Testing
   - Validate configurations before deployment
   - Test with temporary certificates first
   - Verify SNMP connectivity
   - Check rate limiting effectiveness
