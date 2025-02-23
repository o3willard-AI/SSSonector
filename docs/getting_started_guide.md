# SSSonector Getting Started Guide

## Overview
This guide provides a step-by-step introduction to SSSonector, helping new users get up and running quickly with basic setup and common use cases.

## Table of Contents
1. [Quick Start](#quick-start)
2. [Installation](#installation)
3. [Basic Configuration](#basic-configuration)
4. [First Connection](#first-connection)
5. [Common Use Cases](#common-use-cases)
6. [Next Steps](#next-steps)

## Quick Start

### Prerequisites
- Go 1.19 or later
- Linux, Windows, or macOS
- Root/Administrator access
- Basic networking knowledge

### One-Line Installation
```bash
go install github.com/o3willard-AI/SSSonector@latest
```

### Basic Usage
```bash
# Start server
sssonector -config server.yaml

# Start client
sssonector -config client.yaml
```

## Installation

### From Source
```bash
# Clone repository
git clone https://github.com/o3willard-AI/SSSonector.git
cd SSSonector

# Build binary
go build -o sssonector ./cmd/tunnel

# Install binary
sudo mv sssonector /usr/local/bin/
```

### Using Package Manager
```bash
# Homebrew (macOS)
brew install sssonector

# APT (Ubuntu/Debian)
sudo apt-get install sssonector

# YUM (RHEL/CentOS)
sudo yum install sssonector
```

### Directory Setup
```bash
# Create configuration directory
sudo mkdir -p /etc/sssonector/{config,certs}

# Create log directory
sudo mkdir -p /var/log/sssonector

# Create state directory
sudo mkdir -p /var/lib/sssonector

# Set permissions
sudo chown -R sssonector:sssonector /etc/sssonector
sudo chown -R sssonector:sssonector /var/log/sssonector
sudo chown -R sssonector:sssonector /var/lib/sssonector
```

## Basic Configuration

### Server Configuration
Create `/etc/sssonector/config/server.yaml`:
```yaml
type: server
version: 2.0.0
config:
  mode: server
  network:
    interface: tun0
    address: 10.0.0.1/24
    mtu: 1500
  security:
    tls:
      enabled: true
      cert_file: /etc/sssonector/certs/server.crt
      key_file: /etc/sssonector/certs/server.key
      ca_file: /etc/sssonector/certs/ca.crt
```

### Client Configuration
Create `/etc/sssonector/config/client.yaml`:
```yaml
type: client
version: 2.0.0
config:
  mode: client
  network:
    interface: tun0
    address: 10.0.0.2/24
    mtu: 1500
  security:
    tls:
      enabled: true
      cert_file: /etc/sssonector/certs/client.crt
      key_file: /etc/sssonector/certs/client.key
      ca_file: /etc/sssonector/certs/ca.crt
  tunnel:
    server_address: 192.168.1.100
    server_port: 8080
```

### Generate Certificates
```bash
# Generate CA certificate
openssl req -x509 -new -nodes -key ca.key -sha256 -days 365 -out ca.crt

# Generate server certificate
openssl req -new -nodes -key server.key -out server.csr
openssl x509 -req -in server.csr -CA ca.crt -CAkey ca.key -CAcreateserial -out server.crt

# Generate client certificate
openssl req -new -nodes -key client.key -out client.csr
openssl x509 -req -in client.csr -CA ca.crt -CAkey ca.key -CAcreateserial -out client.crt
```

## First Connection

### Start Server
```bash
# Start server in foreground
sssonector -config /etc/sssonector/config/server.yaml

# Start server as service
sudo systemctl start sssonector-server
```

### Start Client
```bash
# Start client in foreground
sssonector -config /etc/sssonector/config/client.yaml

# Start client as service
sudo systemctl start sssonector-client
```

### Verify Connection
```bash
# Check TUN interface
ip addr show tun0

# Test connectivity
ping 10.0.0.1  # From client
ping 10.0.0.2  # From server

# Check connection status
sssonector status
```

## Common Use Cases

### Basic Tunnel
```yaml
# Server configuration
tunnel:
  mode: simple
  interface: tun0
  address: 10.0.0.1/24

# Client configuration
tunnel:
  mode: simple
  server: 192.168.1.100
  interface: tun0
  address: 10.0.0.2/24
```

### Load Balanced Tunnel
```yaml
# Server configuration
tunnel:
  mode: load_balanced
  servers:
    - address: 192.168.1.100
      weight: 1
    - address: 192.168.1.101
      weight: 1
  interface: tun0
  address: 10.0.0.1/24

# Client configuration
tunnel:
  mode: load_balanced
  servers:
    - 192.168.1.100:8080
    - 192.168.1.101:8080
  interface: tun0
  address: 10.0.0.2/24
```

### Rate Limited Tunnel
```yaml
# Server configuration
tunnel:
  mode: rate_limited
  rate_limit:
    enabled: true
    rate: 1000
    burst: 100
  interface: tun0
  address: 10.0.0.1/24

# Client configuration
tunnel:
  mode: rate_limited
  server: 192.168.1.100
  rate_limit:
    enabled: true
    rate: 1000
    burst: 100
  interface: tun0
  address: 10.0.0.2/24
```

## Next Steps

### Advanced Features
1. High Availability Setup
   - Multiple servers
   - Automatic failover
   - Load balancing

2. Performance Tuning
   - Buffer optimization
   - Connection pooling
   - Rate limiting

3. Security Hardening
   - Certificate management
   - Access control
   - Network isolation

### Additional Resources
1. Documentation
   - [Security Guide](security_guide.md)
   - [API Reference](api_reference.md)
   - [Architecture Guide](architecture_guide.md)

2. Configuration
   - [Advanced Configuration Guide](advanced_configuration_guide.md)
   - [Performance Tuning Guide](performance_tuning_guide.md)
   - [Monitoring Guide](monitoring_guide.md)

3. Deployment
   - [Deployment Patterns Guide](deployment_patterns_guide.md)
   - [Troubleshooting Guide](troubleshooting_guide.md)

### Community
- GitHub Issues: Report bugs and request features
- Discussions: Ask questions and share experiences
- Contributing: Guidelines for contributing to SSSonector

## Troubleshooting Tips

### Common Issues

1. Connection Failed
```bash
# Check service status
systemctl status sssonector

# Check logs
tail -f /var/log/sssonector/sssonector.log

# Verify network
ip addr show tun0
```

2. Certificate Problems
```bash
# Check certificate validity
openssl verify -CAfile ca.crt server.crt
openssl verify -CAfile ca.crt client.crt

# Check certificate permissions
ls -l /etc/sssonector/certs/
```

3. Network Issues
```bash
# Check routing
ip route show

# Check firewall
sudo iptables -L

# Test connectivity
ping 10.0.0.1
traceroute 10.0.0.1
```

### Debug Mode
```bash
# Enable debug logging
sssonector -config config.yaml -debug

# Monitor debug output
tail -f /var/log/sssonector/debug.log
```

## Best Practices

### Security
1. Use strong certificates
2. Enable TLS
3. Implement access control
4. Regular updates

### Performance
1. Optimize MTU size
2. Configure buffers
3. Monitor resources
4. Regular maintenance

### Monitoring
1. Enable logging
2. Configure metrics
3. Set up alerts
4. Regular checks

## Quick Reference

### Commands
```bash
# Start service
sssonector -config config.yaml

# Show status
sssonector status

# Show version
sssonector -version

# Show help
sssonector -help
```

### Configuration
```yaml
# Minimal configuration
config:
  mode: server/client
  network:
    interface: tun0
    address: 10.0.0.1/24
  security:
    tls:
      enabled: true
```

### Logs
```bash
# View logs
tail -f /var/log/sssonector/sssonector.log

# View error logs
grep ERROR /var/log/sssonector/sssonector.log

# View debug logs
tail -f /var/log/sssonector/debug.log
```

### Maintenance
```bash
# Rotate logs
logrotate /etc/logrotate.d/sssonector

# Clean old logs
find /var/log/sssonector -type f -mtime +30 -delete

# Backup configuration
tar czf sssonector-config-backup.tar.gz /etc/sssonector/
