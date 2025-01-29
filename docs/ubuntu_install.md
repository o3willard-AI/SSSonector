# Ubuntu Installation Guide for SSSonector

This guide provides detailed instructions for installing and configuring SSSonector on Ubuntu systems.

## System Requirements

- Ubuntu 20.04 LTS or later
- Administrative (sudo) access
- At least 100MB free disk space
- Network interface with TUN/TAP support

## Installation Methods

### Method 1: Using DEB Package (Recommended)

1. Install required dependencies:
```bash
sudo apt-get update
sudo apt-get install -y \
    iproute2 \
    net-tools \
    openssl \
    snmp \
    snmpd
```

2. Download and install SSSonector:
```bash
# Download the package
wget https://github.com/o3willard-AI/SSSonector/releases/download/v1.0.0/sssonector_1.0.0_amd64.deb

# Install the package
sudo dpkg -i sssonector_1.0.0_amd64.deb

# Install any missing dependencies
sudo apt-get install -f
```

### Method 2: Building from Source

1. Install build dependencies:
```bash
sudo apt-get update
sudo apt-get install -y \
    build-essential \
    golang \
    git \
    make \
    openssl \
    libssl-dev \
    pkg-config
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
  interface: "tun0"
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

# Configure firewall
sudo ufw allow 8443/tcp
sudo ufw reload

# Enable IP forwarding
echo "net.ipv4.ip_forward=1" | sudo tee -a /etc/sysctl.conf
sudo sysctl -p

# Start service
sudo systemctl enable sssonector
sudo systemctl start sssonector
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
  interface: "tun0"
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
sudo systemctl enable sssonector
sudo systemctl start sssonector
```

### Example 2: Multi-Site Hub and Spoke

This example connects multiple branch offices to a central HQ.

#### HQ Server (Hub)
```bash
sudo tee /etc/sssonector/config.yaml << 'EOF'
mode: "server"
network:
  interface: "tun0"
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

#### Branch Office 1 (Spoke)
```bash
sudo tee /etc/sssonector/config.yaml << 'EOF'
mode: "client"
network:
  interface: "tun0"
  address: "10.0.0.2"
  mtu: 1500
tunnel:
  cert_file: "/etc/sssonector/certs/branch1.crt"
  key_file: "/etc/sssonector/certs/branch1.key"
  ca_file: "/etc/sssonector/certs/ca.crt"
  server_address: "hq.example.com"
  server_port: 8443
  retry_interval: 5
  bandwidth_limit: 5242880  # 5 MB/s
EOF
```

## Monitoring and Management

### View Service Status
```bash
# Check service status
sudo systemctl status sssonector

# View logs
sudo journalctl -u sssonector -f

# View SNMP metrics
snmpwalk -v2c -c public localhost:161 .1.3.6.1.4.1
```

### Network Testing
```bash
# Check interface
ip addr show tun0

# Test connectivity
ping 10.0.1.1  # From client to server
traceroute 10.0.1.1

# Monitor bandwidth
sudo apt-get install iftop
sudo iftop -i tun0
```

## Troubleshooting

### 1. Service Won't Start
```bash
# Check logs for errors
sudo journalctl -u sssonector -n 50

# Verify permissions
sudo ls -l /etc/sssonector/certs/
sudo chown -R sssonector:sssonector /etc/sssonector/certs/
sudo chmod 600 /etc/sssonector/certs/*.key
```

### 2. Connection Issues
```bash
# Check TUN module
lsmod | grep tun
sudo modprobe tun

# Verify network configuration
ip route show
sudo iptables -L -n -v

# Test server connectivity
nc -zv hq.example.com 8443
```

### 3. Performance Issues
```bash
# Monitor system resources
top -p $(pgrep sssonector)

# Check network throughput
sudo apt-get install iperf3
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
sudo systemctl restart sssonector
```

## Uninstallation

```bash
# Stop and disable service
sudo systemctl stop sssonector
sudo systemctl disable sssonector

# Remove package
sudo apt-get remove sssonector
sudo apt-get purge sssonector  # Also removes configuration

# Clean up directories
sudo rm -rf /etc/sssonector
sudo rm -rf /var/log/sssonector
