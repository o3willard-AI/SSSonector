# Windows Installation Guide for SSSonector

This guide provides detailed instructions for installing and configuring SSSonector on Windows systems.

## System Requirements

- Windows 10/11 or Windows Server 2019/2022
- Administrative privileges
- At least 100MB free disk space
- TAP-Windows Adapter V9 (included in installer)

## Installation Methods

### Method 1: Using Windows Installer (Recommended)

1. Download the installer:
   - Visit the [releases page](https://github.com/o3willard-AI/SSSonector/releases)
   - Download `SSSonector-1.0.0-windows-amd64.exe`

2. Run the installer:
   ```powershell
   # Right-click the installer and select "Run as administrator"
   # Or from PowerShell (Admin):
   Start-Process -FilePath "SSSonector-1.0.0-windows-amd64.exe" -Verb RunAs
   ```

3. Follow the installation wizard:
   - Accept the license agreement
   - Choose installation directory (default: `C:\Program Files\SSSonector`)
   - Select components (TAP driver is required)
   - Complete the installation

### Method 2: Building from Source

1. Install build dependencies:
   ```powershell
   # Install Chocolatey if not already installed
   Set-ExecutionPolicy Bypass -Scope Process -Force
   [System.Net.ServicePointManager]::SecurityProtocol = [System.Net.ServicePointManager]::SecurityProtocol -bor 3072
   iex ((New-Object System.Net.WebClient).DownloadString('https://chocolatey.org/install.ps1'))

   # Install dependencies
   choco install -y `
       git `
       golang `
       mingw `
       make `
       openssl `
       nssm `
       wixtoolset
   ```

2. Clone and build:
   ```powershell
   git clone https://github.com/o3willard-AI/SSSonector.git
   cd SSSonector
   make
   make install
   ```

## Configuration Examples

### Example 1: Simple Point-to-Point Connection

This example sets up a basic tunnel between two locations.

#### Server Configuration (HQ Office)
```powershell
# Create directories
New-Item -ItemType Directory -Force -Path "C:\ProgramData\SSSonector\certs"
Set-Location "C:\ProgramData\SSSonector\certs"

# Generate certificates using OpenSSL
openssl req -x509 -newkey rsa:4096 -keyout server.key -out server.crt -days 365 -nodes `
    -subj "/C=US/ST=California/L=San Francisco/O=MyCompany/CN=hq.example.com"

# Create server configuration
@"
mode: "server"
network:
  interface: "SSSonector0"
  address: "10.0.1.1"
  mtu: 1500
tunnel:
  cert_file: "C:/ProgramData/SSSonector/certs/server.crt"
  key_file: "C:/ProgramData/SSSonector/certs/server.key"
  ca_file: "C:/ProgramData/SSSonector/certs/ca.crt"
  listen_address: "0.0.0.0"
  listen_port: 8443
  max_clients: 10
monitor:
  snmp_enabled: true
  snmp_address: "127.0.0.1"
  snmp_port: 161
"@ | Out-File -FilePath "C:\ProgramData\SSSonector\config.yaml" -Encoding UTF8

# Configure Windows Firewall
New-NetFirewallRule -DisplayName "SSSonector Server" `
    -Direction Inbound `
    -Action Allow `
    -Protocol TCP `
    -LocalPort 8443

# Enable IP forwarding
Set-ItemProperty -Path "HKLM:\SYSTEM\CurrentControlSet\Services\Tcpip\Parameters" `
    -Name "IPEnableRouter" -Value 1

# Start service
Start-Service SSSonector
Set-Service SSSonector -StartupType Automatic
```

#### Client Configuration (Branch Office)
```powershell
# Create directories and copy certificates
New-Item -ItemType Directory -Force -Path "C:\ProgramData\SSSonector\certs"
# Copy certificates from server using secure method (e.g., secure file transfer)

# Create client configuration
@"
mode: "client"
network:
  interface: "SSSonector0"
  address: "10.0.1.2"
  mtu: 1500
tunnel:
  cert_file: "C:/ProgramData/SSSonector/certs/client.crt"
  key_file: "C:/ProgramData/SSSonector/certs/client.key"
  ca_file: "C:/ProgramData/SSSonector/certs/ca.crt"
  server_address: "hq.example.com"
  server_port: 8443
  retry_interval: 5
monitor:
  snmp_enabled: true
  snmp_address: "127.0.0.1"
  snmp_port: 161
"@ | Out-File -FilePath "C:\ProgramData\SSSonector\config.yaml" -Encoding UTF8

# Start service
Start-Service SSSonector
Set-Service SSSonector -StartupType Automatic
```

### Example 2: Multi-Site Hub and Spoke with Bandwidth Control

#### HQ Server (Hub)
```powershell
@"
mode: "server"
network:
  interface: "SSSonector0"
  address: "10.0.0.1"
  mtu: 1500
tunnel:
  cert_file: "C:/ProgramData/SSSonector/certs/server.crt"
  key_file: "C:/ProgramData/SSSonector/certs/server.key"
  ca_file: "C:/ProgramData/SSSonector/certs/ca.crt"
  listen_address: "0.0.0.0"
  listen_port: 8443
  max_clients: 50
  bandwidth_limit: 10485760  # 10 MB/s per client
monitor:
  snmp_enabled: true
  snmp_address: "0.0.0.0"  # Allow remote monitoring
  snmp_port: 161
  snmp_community: "private"
"@ | Out-File -FilePath "C:\ProgramData\SSSonector\config.yaml" -Encoding UTF8
```

## Monitoring and Management

### View Service Status
```powershell
# Check service status
Get-Service SSSonector

# View event logs
Get-EventLog -LogName Application -Source SSSonector -Newest 50

# View SNMP metrics (requires SNMP tools)
snmpwalk -v2c -c public localhost:161 .1.3.6.1.4.1
```

### Network Testing
```powershell
# Check interface
Get-NetAdapter | Where-Object {$_.InterfaceDescription -like "*SSSonector*"}

# Test connectivity
Test-NetConnection -ComputerName 10.0.1.1 -Port 8443
tracert 10.0.1.1

# Monitor bandwidth (install via chocolatey)
choco install -y networkminer
# Use NetworkMiner to monitor SSSonector0 interface
```

## Troubleshooting

### 1. Service Won't Start
```powershell
# Check service status and dependencies
Get-Service SSSonector -Include *

# Check event logs
Get-WinEvent -FilterHashtable @{
    LogName = 'Application'
    ProviderName = 'SSSonector'
}

# Verify permissions
$acl = Get-Acl "C:\ProgramData\SSSonector\certs"
$acl | Format-List
```

### 2. Connection Issues
```powershell
# Check TAP adapter
Get-NetAdapter | Where-Object {$_.InterfaceDescription -like "*TAP-Windows*"}

# Verify routing
Get-NetRoute -InterfaceAlias "SSSonector0"
Get-NetFirewallRule | Where-Object {$_.DisplayName -like "*SSSonector*"}

# Test server connectivity
Test-NetConnection -ComputerName hq.example.com -Port 8443
```

### 3. Performance Issues
```powershell
# Monitor system resources
Get-Counter '\Process(sssonector)\% Processor Time'
Get-Counter '\Network Interface(*)\Bytes Total/sec'

# Check network throughput (install via chocolatey)
choco install -y iperf3
# On server
iperf3 -s -p 5201
# On client
iperf3 -c 10.0.1.1 -p 5201
```

## Windows-Specific Considerations

### TAP Adapter Management
```powershell
# List TAP adapters
Get-NetAdapter | Where-Object {$_.InterfaceDescription -like "*TAP-Windows*"}

# Reset TAP adapter
Disable-NetAdapter -Name "SSSonector0" -Confirm:$false
Enable-NetAdapter -Name "SSSonector0" -Confirm:$false
```

### Service Recovery
```powershell
# Configure automatic restart on failure
$action = New-ScAction -Type Restart
$trigger = New-ScTrigger -AtStartup
Set-Service SSSonector -StartupType Automatic `
    -RecoveryActions @($action) `
    -RestartDelay 60000
```

### Windows Updates
```powershell
# Prevent Windows Update from restarting service
Set-ItemProperty -Path "HKLM:\SOFTWARE\Policies\Microsoft\Windows\WindowsUpdate\AU" `
    -Name "NoAutoRebootWithLoggedOnUsers" -Value 1
```

## Backup and Recovery

### Backup Configuration
```powershell
# Backup certificates and config
Compress-Archive -Path "C:\ProgramData\SSSonector\*" `
    -DestinationPath "C:\Backups\sssonector-backup.zip"
```

### Restore Configuration
```powershell
# Stop service
Stop-Service SSSonector

# Restore from backup
Expand-Archive -Path "C:\Backups\sssonector-backup.zip" `
    -DestinationPath "C:\ProgramData\SSSonector"

# Start service
Start-Service SSSonector
```

## Uninstallation

```powershell
# Stop and remove service
Stop-Service SSSonector
Remove-Service SSSonector

# Uninstall software
Start-Process -FilePath "C:\Program Files\SSSonector\uninstall.exe" -Verb RunAs

# Clean up files
Remove-Item -Path "C:\ProgramData\SSSonector" -Recurse -Force
Remove-Item -Path "C:\Program Files\SSSonector" -Recurse -Force

# Remove firewall rules
Remove-NetFirewallRule -DisplayName "SSSonector*"

# Remove TAP adapter
Get-NetAdapter | Where-Object {$_.InterfaceDescription -like "*TAP-Windows*"} | Remove-NetAdapter -Confirm:$false
