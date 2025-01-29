# macOS Installation Guide for SSSonector

This guide provides detailed instructions for installing and configuring SSSonector on macOS systems.

## System Requirements

- macOS 11 (Big Sur) or later
- Administrative access
- At least 100MB free disk space
- Xcode Command Line Tools (for building from source)

## Installation Methods

### Method 1: Using PKG Installer (Recommended)

1. Install required dependencies:
```bash
# Install Homebrew if not already installed
/bin/bash -c "$(curl -fsSL https://raw.githubusercontent.com/Homebrew/install/HEAD/install.sh)"

# Install dependencies
brew install openssl@3 snmp
```

2. Download and install SSSonector:
```bash
# Download the package
curl -LO https://github.com/o3willard-AI/SSSonector/releases/download/v1.0.0/SSSonector-1.0.0-macos.pkg

# Install the package
sudo installer -pkg SSSonector-1.0.0-macos.pkg -target /
```

### Method 2: Building from Source

1. Install build dependencies:
```bash
# Install Xcode Command Line Tools
xcode-select --install

# Install Homebrew dependencies
brew install \
    golang \
    openssl@3 \
    pkg-config \
    snmp

# Set OpenSSL paths
export LDFLAGS="-L/opt/homebrew/opt/openssl@3/lib"
export CPPFLAGS="-I/opt/homebrew/opt/openssl@3/include"
export PKG_CONFIG_PATH="/opt/homebrew/opt/openssl@3/lib/pkgconfig"
```

2. Clone and build:
```bash
git clone https://github.com/o3willard-AI/SSSonector.git
cd SSSonector
make
sudo make install
```

## Configuration Examples

### Example 1: Simple Point-to-Point Connection

This example sets up a basic tunnel between two locations.

#### Server Configuration (HQ Office)
```bash
# Generate certificates
sudo mkdir -p /etc/sssonector/certs
cd /etc/sssonector/certs
sudo openssl req -x509 -newkey rsa:4096 -keyout server.key -out server.crt -days 365 -nodes \
    -subj "/C=US/ST=California/L=San Francisco/O=MyCompany/CN=hq.example.com"

# Configure server
sudo tee /etc/sssonector/config.yaml << 'EOF'
mode: "server"
network:
  interface: "utun0"  # macOS uses utun instead of tun
  address: "10.0.1.1"
  mtu: 1500
tunnel:
  cert_file: "/etc/sssonector/certs/server.crt"
  key_file: "/etc/sssonector/certs/server.key"
  ca_file: "/etc/sssonector/certs/ca.crt"
  listen_address: "0.0.0.0"
  listen_port: 8443
  max_clients: 10
monitor:
  snmp_enabled: true
  snmp_address: "127.0.0.1"
  snmp_port: 161
EOF

# Configure macOS firewall
sudo /usr/libexec/ApplicationFirewall/socketfilterfw --add /usr/local/bin/sssonector
sudo /usr/libexec/ApplicationFirewall/socketfilterfw --unblock /usr/local/bin/sssonector

# Enable IP forwarding
sudo sysctl -w net.inet.ip.forwarding=1
echo "net.inet.ip.forwarding=1" | sudo tee -a /etc/sysctl.conf

# Start service
sudo launchctl load /Library/LaunchDaemons/com.o3willard.sssonector.plist
```

#### Client Configuration (Branch Office)
```bash
# Copy certificates from server
sudo mkdir -p /etc/sssonector/certs
sudo scp user@hq.example.com:/etc/sssonector/certs/ca.crt /etc/sssonector/certs/
sudo scp user@hq.example.com:/etc/sssonector/certs/client.* /etc/sssonector/certs/

# Configure client
sudo tee /etc/sssonector/config.yaml << 'EOF'
mode: "client"
network:
  interface: "utun0"  # macOS uses utun instead of tun
  address: "10.0.1.2"
  mtu: 1500
tunnel:
  cert_file: "/etc/sssonector/certs/client.crt"
  key_file: "/etc/sssonector/certs/client.key"
  ca_file: "/etc/sssonector/certs/ca.crt"
  server_address: "hq.example.com"
  server_port: 8443
  retry_interval: 5
monitor:
  snmp_enabled: true
  snmp_address: "127.0.0.1"
  snmp_port: 161
EOF

# Start service
sudo launchctl load /Library/LaunchDaemons/com.o3willard.sssonector.plist
```

### Example 2: Multi-Site Hub and Spoke with Bandwidth Control

#### HQ Server (Hub)
```bash
sudo tee /etc/sssonector/config.yaml << 'EOF'
mode: "server"
network:
  interface: "utun0"
  address: "10.0.0.1"
  mtu: 1500
tunnel:
  cert_file: "/etc/sssonector/certs/server.crt"
  key_file: "/etc/sssonector/certs/server.key"
  ca_file: "/etc/sssonector/certs/ca.crt"
  listen_address: "0.0.0.0"
  listen_port: 8443
  max_clients: 50
  bandwidth_limit: 10485760  # 10 MB/s per client
monitor:
  snmp_enabled: true
  snmp_address: "0.0.0.0"  # Allow remote monitoring
  snmp_port: 161
  snmp_community: "private"
EOF
```

## Monitoring and Management

### View Service Status
```bash
# Check service status
sudo launchctl list | grep sssonector

# View logs
sudo log show --predicate 'processImagePath contains "sssonector"' --last 30m

# View SNMP metrics
snmpwalk -v2c -c public localhost:161 .1.3.6.1.4.1
```

### Network Testing
```bash
# Check interface
ifconfig utun0

# Test connectivity
ping 10.0.1.1  # From client to server
traceroute 10.0.1.1

# Monitor bandwidth (install via Homebrew)
brew install iftop
sudo iftop -i utun0
```

## Troubleshooting

### 1. Service Won't Start
```bash
# Check system logs
sudo log show --predicate 'processImagePath contains "sssonector"' --last 1h

# Verify permissions
sudo ls -l /etc/sssonector/certs/
sudo chown -R root:wheel /etc/sssonector/certs/
sudo chmod 600 /etc/sssonector/certs/*.key
```

### 2. Connection Issues
```bash
# Check network settings
networksetup -listallnetworkservices
scutil --nwi

# Verify routing
netstat -nr
sudo pfctl -s rules  # Check packet filter rules

# Test server connectivity
nc -zv hq.example.com 8443
```

### 3. Performance Issues
```bash
# Monitor system resources
top -pid $(pgrep sssonector)

# Check network throughput
brew install iperf3
# On server
iperf3 -s -p 5201
# On client
iperf3 -c 10.0.1.1 -p 5201
```

## Backup and Recovery

### Backup Configuration
```bash
# Backup certificates and config
sudo tar czf sssonector-backup.tar.gz \
    /etc/sssonector/certs/ \
    /etc/sssonector/config.yaml
```

### Restore Configuration
```bash
# Restore from backup
sudo tar xzf sssonector-backup.tar.gz -C /
sudo launchctl unload /Library/LaunchDaemons/com.o3willard.sssonector.plist
sudo launchctl load /Library/LaunchDaemons/com.o3willard.sssonector.plist
```

## macOS-Specific Considerations

### Security & Privacy
1. System Extensions:
   - Open System Preferences → Security & Privacy
   - Allow system extension from SSSonector

2. Network Extension Permission:
   ```bash
   sudo sqlite3 /Library/Application\ Support/com.apple.TCC/TCC.db \
   "INSERT or REPLACE INTO access VALUES('kTCCServiceSystemPolicyNetworkExtension','com.o3willard.sssonector',0,2,4,1,NULL,NULL,0,'UNUSED',NULL,0,1);"
   ```

### Automatic Updates
To prevent automatic updates from disrupting the tunnel:
```bash
# Disable automatic updates for SSSonector
sudo defaults write /Library/Preferences/com.o3willard.sssonector AutoUpdate -bool false
```

## Uninstallation

```bash
# Stop service
sudo launchctl unload /Library/LaunchDaemons/com.o3willard.sssonector.plist

# Remove files
sudo rm -rf /etc/sssonector
sudo rm -rf /var/log/sssonector
sudo rm -f /usr/local/bin/sssonector
sudo rm -f /Library/LaunchDaemons/com.o3willard.sssonector.plist

# Remove network extension
sudo sqlite3 /Library/Application\ Support/com.apple.TCC/TCC.db \
"DELETE FROM access WHERE client='com.o3willard.sssonector';"
