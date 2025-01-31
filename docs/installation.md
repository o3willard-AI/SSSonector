# Installation Guide

## Configuration

The SSSonector service uses a YAML configuration file located at `/etc/sssonector/config.yaml`. Here's a complete configuration reference:

```yaml
# Mode can be "server" or "client"
mode: "server"

# Network interface configuration
network:
  # Platform-specific interface names:
  # - Linux: "tun0"
  # - macOS: "utun0"
  # - Windows: "SSSonector0"
  interface: "tun0"
  address: "10.0.0.1/24"
  mtu: 1500

# Tunnel configuration
tunnel:
  # Certificate paths (use forward slashes even on Windows)
  certFile: "/etc/sssonector/certs/server.crt"
  keyFile: "/etc/sssonector/certs/server.key"
  caFile: "/etc/sssonector/certs/ca.crt"

  # Server mode settings
  listenAddress: "0.0.0.0"
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
  level: "info"  # debug, info, warn, error
  filePath: "/var/log/sssonector/service.log"
  maxSize: 100  # MB
  maxBackups: 5
  maxAge: 30    # days
```

## Platform-Specific Installation

### Linux (Debian/Ubuntu)
```bash
# Download from GitHub Releases
wget https://github.com/o3willard-AI/SSSonector/releases/download/v1.0.0/sssonector_1.0.0_amd64.deb

# Install the package
sudo dpkg -i sssonector_1.0.0_amd64.deb
sudo apt-get install -f  # Install dependencies
```

### Linux (RHEL/CentOS)
```bash
# Download from GitHub Releases
wget https://github.com/o3willard-AI/SSSonector/releases/download/v1.0.0/sssonector-1.0.0-1.x86_64.rpm

# Install the package
sudo rpm -i sssonector-1.0.0-1.x86_64.rpm
```

### Windows
1. Download `sssonector-1.0.0-setup.exe` from GitHub Releases
2. Run the installer with administrator privileges
3. TAP driver will be installed automatically

### macOS
```bash
# Download from GitHub Releases
curl -LO https://github.com/o3willard-AI/SSSonector/releases/download/v1.0.0/sssonector-1.0.0-macos.pkg

# Install the package
sudo installer -pkg sssonector-1.0.0-macos.pkg -target /
```

## Certificate Setup

1. Generate CA certificate:
```bash
openssl req -x509 -newkey rsa:4096 -keyout ca.key -out ca.crt -days 365 -nodes \
    -subj "/C=US/ST=California/L=San Francisco/O=MyCompany/CN=SSSonector CA"
```

2. Generate server certificate:
```bash
# Generate key and CSR
openssl req -newkey rsa:4096 -keyout server.key -out server.csr -nodes \
    -subj "/C=US/ST=California/L=San Francisco/O=MyCompany/CN=server.example.com"

# Sign with CA
openssl x509 -req -in server.csr -CA ca.crt -CAkey ca.key -CAcreateserial \
    -out server.crt -days 365 -sha256
```

3. Generate client certificate:
```bash
# Generate key and CSR
openssl req -newkey rsa:4096 -keyout client.key -out client.csr -nodes \
    -subj "/C=US/ST=California/L=San Francisco/O=MyCompany/CN=client.example.com"

# Sign with CA
openssl x509 -req -in client.csr -CA ca.crt -CAkey ca.key -CAcreateserial \
    -out client.crt -days 365 -sha256
```

4. Install certificates:

Linux/macOS:
```bash
sudo mkdir -p /etc/sssonector/certs
sudo cp ca.crt server.crt server.key /etc/sssonector/certs/
sudo chmod 644 /etc/sssonector/certs/*.crt
sudo chmod 600 /etc/sssonector/certs/*.key
```

Windows (PowerShell Admin):
```powershell
New-Item -ItemType Directory -Force -Path "C:\ProgramData\SSSonector\certs"
Copy-Item ca.crt, server.crt, server.key -Destination "C:\ProgramData\SSSonector\certs"
$acl = Get-Acl "C:\ProgramData\SSSonector\certs\server.key"
$acl.SetAccessRuleProtection($true, $false)
$rule = New-Object System.Security.AccessControl.FileSystemAccessRule("SYSTEM","FullControl","Allow")
$acl.AddAccessRule($rule)
Set-Acl "C:\ProgramData\SSSonector\certs\server.key" $acl
```

## Service Management

### Linux (systemd)
```bash
# Start the service
sudo systemctl start sssonector

# Enable at boot
sudo systemctl enable sssonector

# Check status
sudo systemctl status sssonector

# View logs
sudo journalctl -u sssonector -f
```

### macOS (launchd)
```bash
# Load the service
sudo launchctl load /Library/LaunchDaemons/com.o3willard.sssonector.plist

# Start the service
sudo launchctl start com.o3willard.sssonector

# Check status
sudo launchctl list | grep sssonector

# View logs
sudo log show --predicate 'processImagePath contains "sssonector"' --last 30m
```

### Windows
```powershell
# Start the service
Start-Service SSSonector

# Set to auto-start
Set-Service SSSonector -StartupType Automatic

# Check status
Get-Service SSSonector

# View logs
Get-EventLog -LogName Application -Source SSSonector -Newest 50
```

## Firewall Configuration

### Linux (ufw)
```bash
sudo ufw allow 8443/tcp
sudo ufw reload
```

### macOS
```bash
sudo /usr/libexec/ApplicationFirewall/socketfilterfw --add /usr/local/bin/sssonector
sudo /usr/libexec/ApplicationFirewall/socketfilterfw --unblock /usr/local/bin/sssonector
```

### Windows
```powershell
New-NetFirewallRule -DisplayName "SSSonector" -Direction Inbound -Action Allow -Protocol TCP -LocalPort 8443
```

## Monitoring Setup

### SNMP Configuration
```yaml
monitor:
  snmpEnabled: true
  snmpAddress: "0.0.0.0"  # Listen on all interfaces
  snmpPort: 161
  snmpCommunity: "public"
```

### Log Rotation
Linux/macOS:
```bash
sudo mkdir -p /var/log/sssonector
sudo chown -R sssonector:sssonector /var/log/sssonector
```

Windows:
```powershell
New-Item -ItemType Directory -Force -Path "C:\ProgramData\SSSonector\logs"
$acl = Get-Acl "C:\ProgramData\SSSonector\logs"
$rule = New-Object System.Security.AccessControl.FileSystemAccessRule("NetworkService","Modify","Allow")
$acl.AddAccessRule($rule)
Set-Acl "C:\ProgramData\SSSonector\logs" $acl
```

## Troubleshooting

### Common Issues

1. Certificate Problems:
```bash
# Check certificate validity
openssl verify -CAfile /etc/sssonector/certs/ca.crt /etc/sssonector/certs/server.crt
openssl x509 -in /etc/sssonector/certs/server.crt -text -noout
```

2. Network Interface Issues:
Linux:
```bash
sudo ip link show tun0
sudo ip addr add 10.0.0.1/24 dev tun0
sudo ip link set tun0 up
```

macOS:
```bash
ifconfig utun0
sudo ifconfig utun0 10.0.0.1 10.0.0.2
```

Windows:
```powershell
Get-NetAdapter | Where-Object {$_.InterfaceDescription -like "*TAP-Windows*"}
```

3. Permission Issues:
Linux/macOS:
```bash
sudo chown -R root:root /etc/sssonector
sudo chmod 755 /etc/sssonector
sudo chmod 700 /etc/sssonector/certs
```

Windows:
```powershell
$paths = @(
    "C:\ProgramData\SSSonector",
    "C:\ProgramData\SSSonector\certs",
    "C:\ProgramData\SSSonector\logs"
)
foreach ($path in $paths) {
    $acl = Get-Acl $path
    $rule = New-Object System.Security.AccessControl.FileSystemAccessRule("NetworkService","Modify","Allow")
    $acl.AddAccessRule($rule)
    Set-Acl $path $acl
}
```

### Debug Mode
```bash
# Linux/macOS
sudo SSSONECTOR_DEBUG=1 sssonector -config /etc/sssonector/config.yaml

# Windows
$env:SSSONECTOR_DEBUG=1; & 'C:\Program Files\SSSonector\sssonector.exe' -config 'C:\ProgramData\SSSonector\config.yaml'
```
