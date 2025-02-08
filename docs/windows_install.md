# Windows Installation Guide

This guide provides instructions for installing and configuring SSSonector on Windows systems. Please note that Windows support is currently in basic mode, with TUN interface implementation planned for future releases.

## System Requirements

### Minimum Requirements
- Windows 10 (1909) or later
- Windows Server 2016 or later
- 2GB RAM
- 200MB disk space
- Administrator privileges

### Recommended Requirements
- Windows 10 (21H2) or later
- Windows Server 2019 or later
- 4GB RAM
- 500MB disk space
- Dedicated network interface

## Pre-Installation Steps

1. Install Required Software:
   - [TAP-Windows Adapter V9](https://build.openvpn.net/downloads/releases/tap-windows-9.24.2-I601-Win10.exe)
   - [Visual C++ Redistributable 2019](https://aka.ms/vs/16/release/vc_redist.x64.exe)

2. Verify TAP Adapter Installation:
```powershell
Get-NetAdapter | Where-Object { $_.InterfaceDescription -like "*TAP-Windows*" }
```

## Installation Methods

### Method 1: Binary Installation (Recommended)

1. Download the latest release:
```powershell
# Using PowerShell
Invoke-WebRequest -Uri "https://github.com/o3willard-AI/SSSonector/releases/download/v2.0.0/sssonector_2.0.0_windows_amd64.exe" -OutFile "sssonector.exe"
```

2. Install to Program Files:
```powershell
# Create installation directory
New-Item -ItemType Directory -Path "C:\Program Files\SSSonector" -Force

# Move executable
Move-Item -Path "sssonector.exe" -Destination "C:\Program Files\SSSonector"

# Add to PATH
$env:Path += ";C:\Program Files\SSSonector"
[Environment]::SetEnvironmentVariable("Path", $env:Path, [EnvironmentVariableTarget]::Machine)
```

### Method 2: Building from Source

1. Install Go:
```powershell
# Download Go installer
Invoke-WebRequest -Uri "https://go.dev/dl/go1.21.6.windows-amd64.msi" -OutFile "go_installer.msi"

# Install Go
Start-Process -Wait -FilePath "msiexec.exe" -ArgumentList "/i go_installer.msi /quiet"

# Refresh environment
$env:Path = [System.Environment]::GetEnvironmentVariable("Path","Machine") + ";" + [System.Environment]::GetEnvironmentVariable("Path","User")
```

2. Clone and build:
```powershell
git clone https://github.com/o3willard-AI/SSSonector.git
cd SSSonector
go build -o sssonector.exe .\cmd\tunnel
```

## Configuration Examples

### Example 1: Basic Client Setup
```yaml
mode: "client"
network:
  interface: "sssonector0"
  address: "10.0.0.2/24"
  mtu: 1500
tunnel:
  cert_file: "C:\\ProgramData\\SSSonector\\certs\\client.crt"
  key_file: "C:\\ProgramData\\SSSonector\\certs\\client.key"
  ca_file: "C:\\ProgramData\\SSSonector\\certs\\ca.crt"
  server_address: "server.example.com"
  server_port: 8443
monitor:
  enabled: true
  log_file: "C:\\ProgramData\\SSSonector\\logs\\client.log"
```

### Example 2: High-Performance Client Setup
```yaml
mode: "client"
network:
  interface: "sssonector0"
  address: "10.0.0.2/24"
  mtu: 1500
tunnel:
  cert_file: "C:\\ProgramData\\SSSonector\\certs\\client.crt"
  key_file: "C:\\ProgramData\\SSSonector\\certs\\client.key"
  ca_file: "C:\\ProgramData\\SSSonector\\certs\\ca.crt"
  server_address: "server.example.com"
  server_port: 8443
monitor:
  enabled: true
  log_file: "C:\\ProgramData\\SSSonector\\logs\\client.log"
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

## Windows Service Setup

1. Create service using PowerShell:
```powershell
# Create service
New-Service -Name "SSSonector" `
            -DisplayName "SSSonector Tunnel Service" `
            -Description "Secure tunnel service for network connectivity" `
            -BinaryPathName "C:\Program Files\SSSonector\sssonector.exe -config C:\ProgramData\SSSonector\config.yaml" `
            -StartupType Automatic

# Set recovery options
$action = New-ScheduledTaskAction -Execute "C:\Program Files\SSSonector\sssonector.exe" -Argument "-config C:\ProgramData\SSSonector\config.yaml"
$trigger = New-ScheduledTaskTrigger -AtStartup
$principal = New-ScheduledTaskPrincipal -UserId "SYSTEM" -LogonType ServiceAccount -RunLevel Highest
$settings = New-ScheduledTaskSettingsSet -RestartCount 3 -RestartInterval (New-TimeSpan -Minutes 1)
Register-ScheduledTask -TaskName "SSSonector" -Action $action -Trigger $trigger -Principal $principal -Settings $settings
```

2. Start the service:
```powershell
Start-Service -Name "SSSonector"
```

## Firewall Configuration

1. Allow tunnel traffic:
```powershell
# Allow tunnel port
New-NetFirewallRule -DisplayName "SSSonector Tunnel" `
                    -Direction Inbound `
                    -Action Allow `
                    -Protocol TCP `
                    -LocalPort 8443

# Allow SNMP monitoring (if enabled)
New-NetFirewallRule -DisplayName "SSSonector SNMP" `
                    -Direction Inbound `
                    -Action Allow `
                    -Protocol UDP `
                    -LocalPort 10161
```

## Performance Tuning

1. Network adapter optimization:
```powershell
# Disable TCP auto-tuning (if causing issues)
netsh int tcp set global autotuninglevel=disabled

# Set network adapter parameters
Set-NetAdapterAdvancedProperty -Name "sssonector0" -DisplayName "Receive Buffers" -DisplayValue "2048"
Set-NetAdapterAdvancedProperty -Name "sssonector0" -DisplayName "Transmit Buffers" -DisplayValue "2048"
```

2. System optimization:
```powershell
# Optimize network settings
Set-ItemProperty -Path "HKLM:\SYSTEM\CurrentControlSet\Services\Tcpip\Parameters" -Name "TcpTimedWaitDelay" -Value 30
Set-ItemProperty -Path "HKLM:\SYSTEM\CurrentControlSet\Services\Tcpip\Parameters" -Name "MaxUserPort" -Value 65534
```

## Troubleshooting

### Common Issues

1. Service Start Fails
```powershell
# Check service status
Get-Service -Name "SSSonector"

# View application logs
Get-EventLog -LogName Application -Source "SSSonector"

# Check configuration
Test-Path "C:\ProgramData\SSSonector\config.yaml"
```

2. Network Connectivity Issues
```powershell
# Test network connectivity
Test-NetConnection -ComputerName "server.example.com" -Port 8443

# Check interface status
Get-NetAdapter | Where-Object { $_.InterfaceDescription -like "*TAP-Windows*" }

# View detailed network statistics
Get-NetTCPConnection | Where-Object { $_.LocalPort -eq 8443 }
```

3. Performance Issues
```powershell
# Monitor network performance
Get-Counter -Counter "\Network Interface(*)\Bytes Total/sec"

# Check system resources
Get-Process sssonector | Select-Object CPU, WorkingSet, HandleCount
```

## Monitoring Setup

### Windows Performance Counters
```powershell
# Create performance counter category
New-PerfCounterCategory -Name "SSSonector" -Help "SSSonector Performance Counters"

# Add counters
New-PerfCounter -Name "BytesIn" -CategoryName "SSSonector" -CounterType NumberOfItems64 -Help "Bytes received"
New-PerfCounter -Name "BytesOut" -CategoryName "SSSonector" -CounterType NumberOfItems64 -Help "Bytes sent"
```

### SNMP Monitoring
```yaml
monitor:
  enabled: true
  snmp_enabled: true
  snmp_address: "0.0.0.0"
  snmp_port: 10161
  snmp_community: "public"
```

## Support and Resources

- Documentation: https://docs.sssonector.io
- Windows-specific Issues: https://github.com/o3willard-AI/SSSonector/labels/windows
- Community Support: https://community.sssonector.io/c/windows
- Security Updates: https://security.sssonector.io/windows

## Known Limitations

1. TUN Interface Support
   - Currently using TAP adapter as fallback
   - Native TUN support planned for future releases
   - Limited MTU options available

2. Performance Considerations
   - Lower throughput compared to Linux
   - Higher CPU usage due to TAP implementation
   - Limited rate limiting precision

3. Monitoring Capabilities
   - Partial SNMP implementation
   - Some metrics may be unavailable
   - Performance counter limitations
