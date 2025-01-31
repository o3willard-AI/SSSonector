# Installation Guide

## Configuration

The SSSonector service uses a YAML configuration file located at `/etc/sssonector/config.yaml`. Here's a complete configuration reference:

```yaml
# Mode can be "server" or "client"
mode: "server"

# Network interface configuration
network:
  interface: "tun0"
  address: "10.0.0.1/24"
  mtu: 1500

# Tunnel configuration
tunnel:
  # Certificate paths
  certFile: "/etc/sssonector/certs/server.crt"
  keyFile: "/etc/sssonector/certs/server.key"
  caFile: "/etc/sssonector/certs/ca.crt"
  
  # Server mode settings
  listenAddress: "127.0.0.1"
  listenPort: 8443
  maxClients: 10
  
  # Client mode settings (only needed in client mode)
  serverAddress: "server.example.com"
  serverPort: 8443
  
  # Bandwidth control (in Kbps)
  uploadKbps: 10240    # 10 Mbps
  downloadKbps: 10240  # 10 Mbps

# Monitoring configuration
monitor:
  logFile: "/var/log/sssonector/monitor.log"
  snmpEnabled: false
  snmpPort: 161
  snmpCommunity: "public"

# Logging configuration
logging:
  level: "info"
  filePath: "/var/log/sssonector/service.log"
  maxSize: 100  # MB
```

## Platform-Specific Installation

### Linux (Debian/Ubuntu)
```bash
sudo dpkg -i sssonector_1.0.0_amd64.deb
```

### Linux (RHEL/CentOS)
```bash
sudo rpm -i sssonector-1.0.0-1.x86_64.rpm
```

### Windows
Run the installer: `sssonector-1.0.0-setup.exe`

### macOS
```bash
sudo installer -pkg sssonector-1.0.0.pkg -target /
```

## Certificate Setup

1. Generate CA certificate:
```bash
openssl req -x509 -newkey rsa:4096 -keyout ca.key -out ca.crt -days 365 -nodes
```

2. Generate server certificate:
```bash
openssl req -newkey rsa:4096 -keyout server.key -out server.csr -nodes
openssl x509 -req -in server.csr -CA ca.crt -CAkey ca.key -CAcreateserial -out server.crt -days 365
```

3. Place certificates in the appropriate directory:
```bash
sudo mkdir -p /etc/sssonector/certs
sudo cp ca.crt server.crt server.key /etc/sssonector/certs/
sudo chmod 644 /etc/sssonector/certs/*.crt
sudo chmod 600 /etc/sssonector/certs/*.key
```

## Service Management

Start the service:
```bash
sudo systemctl start sssonector
```

Enable at boot:
```bash
sudo systemctl enable sssonector
```

Check status:
```bash
sudo systemctl status sssonector
```

View logs:
```bash
sudo journalctl -u sssonector
```
