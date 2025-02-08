# macOS Installation Guide

This guide provides instructions for installing and configuring SSSonector on macOS systems. Please note that macOS support is currently in basic mode, with TUN interface implementation planned for future releases.

## System Requirements

### Minimum Requirements
- macOS 10.15 (Catalina) or later
- Intel or Apple Silicon processor
- 2GB RAM
- 200MB disk space
- Administrator privileges

### Recommended Requirements
- macOS 12 (Monterey) or later
- 4GB RAM
- 500MB disk space
- Dedicated network interface

### Development Requirements
- Xcode Command Line Tools
- Go 1.21 or later
- Network Extension entitlements (for future TUN support)

## Pre-Installation Steps

1. Install Xcode Command Line Tools:
```bash
xcode-select --install
```

2. Install Homebrew (if not installed):
```bash
/bin/bash -c "$(curl -fsSL https://raw.githubusercontent.com/Homebrew/install/HEAD/install.sh)"
```

3. Install dependencies:
```bash
brew install openssl
```

## Installation Methods

### Method 1: Binary Installation (Recommended)

1. Download the appropriate binary:
```bash
# For Intel Macs
curl -LO https://github.com/o3willard-AI/SSSonector/releases/download/v2.0.0/sssonector_2.0.0_darwin_amd64

# For Apple Silicon Macs
curl -LO https://github.com/o3willard-AI/SSSonector/releases/download/v2.0.0/sssonector_2.0.0_darwin_arm64
```

2. Install the binary:
```bash
# Make executable
chmod +x sssonector_2.0.0_darwin_*

# Move to applications directory
sudo mv sssonector_2.0.0_darwin_* /usr/local/bin/sssonector

# Verify installation
sssonector -version
```

### Method 2: Building from Source

1. Install Go:
```bash
brew install go
```

2. Clone and build:
```bash
git clone https://github.com/o3willard-AI/SSSonector.git
cd SSSonector
make build
sudo make install
```

## Configuration Examples

### Example 1: Basic Client Setup
```yaml
mode: "client"
network:
  interface: "utun0"
  address: "10.0.0.2/24"
  mtu: 1500
tunnel:
  cert_file: "/etc/sssonector/certs/client.crt"
  key_file: "/etc/sssonector/certs/client.key"
  ca_file: "/etc/sssonector/certs/ca.crt"
  server_address: "server.example.com"
  server_port: 8443
monitor:
  enabled: true
  log_file: "/var/log/sssonector/client.log"
```

### Example 2: High-Performance Client Setup
```yaml
mode: "client"
network:
  interface: "utun0"
  address: "10.0.0.2/24"
  mtu: 1500
tunnel:
  cert_file: "/etc/sssonector/certs/client.crt"
  key_file: "/etc/sssonector/certs/client.key"
  ca_file: "/etc/sssonector/certs/ca.crt"
  server_address: "server.example.com"
  server_port: 8443
monitor:
  enabled: true
  log_file: "/var/log/sssonector/client.log"
  snmp_enabled: true
  snmp_port: 10161
throttle:
  enabled: true
  rate_limit: 1000000000  # 1 Gbps
  burst_limit: 1200000000 # 1.2 Gbps burst
buffer:
  read_size: 65536
  write_size: 65536
  pool_size: 1024
```

## Launch Daemon Setup

1. Create launch daemon configuration:
```bash
sudo tee /Library/LaunchDaemons/com.sssonector.plist << 'EOL'
<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
    <key>Label</key>
    <string>com.sssonector</string>
    <key>ProgramArguments</key>
    <array>
        <string>/usr/local/bin/sssonector</string>
        <string>-config</string>
        <string>/etc/sssonector/config.yaml</string>
    </array>
    <key>RunAtLoad</key>
    <true/>
    <key>KeepAlive</key>
    <true/>
    <key>StandardErrorPath</key>
    <string>/var/log/sssonector/error.log</string>
    <key>StandardOutPath</key>
    <string>/var/log/sssonector/output.log</string>
</dict>
</plist>
EOL
```

2. Set proper permissions:
```bash
sudo chown root:wheel /Library/LaunchDaemons/com.sssonector.plist
sudo chmod 644 /Library/LaunchDaemons/com.sssonector.plist
```

3. Load the launch daemon:
```bash
sudo launchctl load /Library/LaunchDaemons/com.sssonector.plist
```

## Firewall Configuration

1. Allow tunnel traffic:
```bash
# Add firewall rules
sudo /usr/libexec/ApplicationFirewall/socketfilterfw --add /usr/local/bin/sssonector
sudo /usr/libexec/ApplicationFirewall/socketfilterfw --unblock /usr/local/bin/sssonector

# Allow specific ports
echo "pass in proto tcp from any to any port 8443" | sudo pfctl -f -
echo "pass in proto udp from any to any port 10161" | sudo pfctl -f -  # For SNMP
```

## Performance Tuning

1. Network stack optimization:
```bash
# Increase network buffer sizes
sudo sysctl -w kern.ipc.maxsockbuf=8388608
sudo sysctl -w net.inet.tcp.sendspace=262144
sudo sysctl -w net.inet.tcp.recvspace=262144

# Make changes permanent
cat << 'EOL' | sudo tee /etc/sysctl.conf
kern.ipc.maxsockbuf=8388608
net.inet.tcp.sendspace=262144
net.inet.tcp.recvspace=262144
EOL
```

2. System optimization:
```bash
# Increase maximum file descriptors
sudo launchctl limit maxfiles 65536 65536

# Increase maximum processes
sudo launchctl limit maxproc 2048 2048
```

## Troubleshooting

### Common Issues

1. Service Start Fails
```bash
# Check service status
sudo launchctl list | grep sssonector

# View logs
tail -f /var/log/sssonector/error.log
tail -f /var/log/sssonector/output.log

# Check configuration
sudo cat /etc/sssonector/config.yaml
```

2. Network Connectivity Issues
```bash
# Test network connectivity
ping server.example.com
nc -zv server.example.com 8443

# Check interface status
ifconfig utun0

# View network statistics
netstat -an | grep 8443
```

3. Performance Issues
```bash
# Monitor network throughput
sudo dtrace -n 'fbt::tcp_output:entry { @bytes = sum(args[2]->m_pkthdr.len); }'

# Check system resources
top -pid $(pgrep sssonector)
```

## Monitoring Setup

### SNMP Monitoring
```yaml
monitor:
  enabled: true
  snmp_enabled: true
  snmp_address: "0.0.0.0"
  snmp_port: 10161
  snmp_community: "public"
```

### Integration with macOS Monitoring Tools
```bash
# Install monitoring tools
brew install prometheus
brew install grafana

# Configure Prometheus
cat << 'EOL' > prometheus.yml
scrape_configs:
  - job_name: 'sssonector'
    static_configs:
      - targets: ['localhost:9091']
EOL
```

## Support and Resources

- Documentation: https://docs.sssonector.io
- macOS-specific Issues: https://github.com/o3willard-AI/SSSonector/labels/macos
- Community Support: https://community.sssonector.io/c/macos
- Security Updates: https://security.sssonector.io/macos

## Known Limitations

1. TUN Interface Support
   - Basic network interface support only
   - Native TUN support planned for future releases
   - Limited MTU options available

2. Performance Considerations
   - Lower throughput compared to Linux
   - Higher CPU usage in current implementation
   - Limited rate limiting precision

3. Platform-Specific Issues
   - Network Extension entitlement requirements
   - System Integrity Protection considerations
   - Limited kernel extension support

4. Monitoring Capabilities
   - Partial SNMP implementation
   - Limited system metrics availability
   - DTrace probe limitations
