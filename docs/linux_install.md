# Linux Installation Guide

This guide provides detailed instructions for installing and configuring SSSonector on Linux systems.

## System Requirements

### Minimum Requirements
- Linux kernel 4.4 or later
- TUN/TAP kernel module support
- iproute2 package
- Administrative (sudo) privileges
- 1GB RAM
- 100MB disk space

### Recommended Requirements
- Linux kernel 5.10 or later
- 4GB RAM
- 500MB disk space
- Dedicated network interface

### Supported Distributions
- Ubuntu 20.04 LTS or later
- CentOS 7/8
- RHEL 8/9
- Debian 11/12
- Fedora 35+

## Pre-Installation Steps

1. Verify TUN/TAP support:
```bash
# Check if TUN module is loaded
lsmod | grep tun

# Load TUN module if not present
sudo modprobe tun

# Make TUN module load on boot
echo "tun" | sudo tee /etc/modules-load.d/tun.conf
```

2. Install required packages:
```bash
# Ubuntu/Debian
sudo apt-get update
sudo apt-get install -y build-essential iproute2 net-tools

# CentOS/RHEL
sudo dnf groupinstall "Development Tools"
sudo dnf install iproute net-tools
```

## Installation Methods

### Method 1: Binary Installation (Recommended)

1. Download the latest release:
```bash
# For x86_64 systems
wget https://github.com/o3willard-AI/SSSonector/releases/download/v2.0.0/sssonector_2.0.0_linux_amd64
chmod +x sssonector_2.0.0_linux_amd64

# For ARM64 systems
wget https://github.com/o3willard-AI/SSSonector/releases/download/v2.0.0/sssonector_2.0.0_linux_arm64
chmod +x sssonector_2.0.0_linux_arm64
```

2. Install the binary:
```bash
sudo mv sssonector_2.0.0_linux_* /usr/local/bin/sssonector
```

### Method 2: Building from Source

1. Install Go:
```bash
wget https://go.dev/dl/go1.21.6.linux-amd64.tar.gz
sudo tar -C /usr/local -xzf go1.21.6.linux-amd64.tar.gz
export PATH=$PATH:/usr/local/go/bin
```

2. Clone and build:
```bash
git clone https://github.com/o3willard-AI/SSSonector.git
cd SSSonector
make build
sudo make install
```

## Configuration Examples

Complete reference examples live in `configs/` and `templates/`. The
current schema requires `metadata.schema_version: "2.0.0"`; configs are
validated at startup and unknown or invalid values abort startup rather
than being ignored.

### Example 1: Basic Server Setup
```yaml
metadata:
  schema_version: "2.0.0"
  environment: production
type: server
config:
  mode: server
  logging:
    level: info
    format: json
  network:
    name: tun0
    interface: tun0
    address: "10.0.0.1/24"
    mtu: 1500
  tunnel:
    listen_address: "0.0.0.0"
    listen_port: 8443
  auth:
    cert_file: "/etc/sssonector/certs/server.crt"
    key_file: "/etc/sssonector/certs/server.key"
    ca_file: "/etc/sssonector/certs/ca.crt"
```

### Example 2: High-Performance Client Setup
```yaml
metadata:
  schema_version: "2.0.0"
  environment: production
type: client
config:
  mode: client
  logging:
    level: info
    format: json
  network:
    name: tun0
    interface: tun0
    address: "10.0.0.2/24"
    mtu: 9000  # Jumbo frames
  tunnel:
    server_address: "server.example.com"
    server_port: 8443
  auth:
    cert_file: "/etc/sssonector/certs/client.crt"
    key_file: "/etc/sssonector/certs/client.key"
    ca_file: "/etc/sssonector/certs/ca.crt"
throttle:
  enabled: true
  rate: 909090909  # ~1 Gbps effective after the 1.1 TCP overhead factor
```

See the [Configuration Guide](configuration_guide.md) for monitoring,
SNMP, Prometheus, facade, and hot-reload options.

## Systemd Service Setup

1. Create service file:
```bash
sudo tee /etc/systemd/system/sssonector.service << 'EOL'
[Unit]
Description=SSSonector Tunnel Service
After=network.target

[Service]
Type=simple
User=root
ExecStart=/usr/local/bin/sssonector -config /etc/sssonector/config.yaml
Restart=always
RestartSec=5
LimitNOFILE=65536

[Install]
WantedBy=multi-user.target
EOL
```

2. Enable and start service:
```bash
sudo systemctl daemon-reload
sudo systemctl enable sssonector
sudo systemctl start sssonector
```

## Firewall Configuration

### UFW (Ubuntu)
```bash
sudo ufw allow 8443/tcp  # Tunnel port
sudo ufw allow 10161/udp # SNMP monitoring (if enabled)
```

### FirewallD (CentOS/RHEL)
```bash
sudo firewall-cmd --permanent --add-port=8443/tcp
sudo firewall-cmd --permanent --add-port=10161/udp
sudo firewall-cmd --reload
```

## Performance Tuning

1. Network stack optimization:
```bash
# Add to /etc/sysctl.conf
net.core.rmem_max = 16777216
net.core.wmem_max = 16777216
net.ipv4.tcp_rmem = 4096 87380 16777216
net.ipv4.tcp_wmem = 4096 87380 16777216
```

2. Interface optimization:
```bash
# Set interface txqueuelen
sudo ip link set tun0 txqueuelen 10000

# Enable TCP BBR
echo "net.core.default_qdisc=fq" | sudo tee -a /etc/sysctl.conf
echo "net.ipv4.tcp_congestion_control=bbr" | sudo tee -a /etc/sysctl.conf
sudo sysctl -p
```

## Troubleshooting

### Common Issues

1. TUN Interface Creation Fails
```bash
# Check TUN module
lsmod | grep tun
# Load if missing
sudo modprobe tun
# Check device permissions
ls -l /dev/net/tun
```

2. Connection Issues
```bash
# Check service status
systemctl status sssonector
# View logs
journalctl -u sssonector -f
# Check network connectivity
ping server.example.com
```

3. Performance Issues
```bash
# Monitor network throughput
iftop -i tun0
# Check system resources
top
# View detailed metrics
sssonector -metrics
```

## Monitoring Integration

### Prometheus Integration
```yaml
monitor:
  enabled: true
  prometheus_enabled: true
  prometheus_port: 9091
  metrics_path: "/metrics"
```

### Grafana Dashboard
```bash
# Import dashboard
grafana-cli plugins install grafana-piechart-panel
curl -o sssonector-dashboard.json https://raw.githubusercontent.com/o3willard-AI/SSSonector/main/monitoring/grafana/dashboard.json
```

## Support and Resources

- Documentation: https://docs.sssonector.io
- GitHub Issues: https://github.com/o3willard-AI/SSSonector/issues
- Community Forum: https://community.sssonector.io
- Security Updates: https://security.sssonector.io
