# SSSonector Windows Service Installation Script
$ErrorActionPreference = "Stop"

# Service parameters
$ServiceName = "SSSonector"
$DisplayName = "SSSonector SSL Tunnel Service"
$Description = "Secure SSL tunnel service for remote office connectivity"
$BinaryPath = "`"$PSScriptRoot\sssonector.exe`" -config `"$env:ProgramData\SSSonector\config.yaml`""

# Stop and remove existing service if it exists
if (Get-Service -Name $ServiceName -ErrorAction SilentlyContinue) {
    Write-Host "Stopping existing service..."
    Stop-Service -Name $ServiceName -Force
    Write-Host "Removing existing service..."
    sc.exe delete $ServiceName
    Start-Sleep -Seconds 2
}

# Create service
Write-Host "Creating service..."
$service = New-Service -Name $ServiceName `
    -DisplayName $DisplayName `
    -Description $Description `
    -BinaryPathName $BinaryPath `
    -StartupType Automatic

# Set recovery options (restart on failure)
Write-Host "Configuring service recovery options..."
sc.exe failure $ServiceName reset= 86400 actions= restart/60000/restart/60000/restart/60000

# Configure service dependencies
Write-Host "Configuring service dependencies..."
sc.exe config $ServiceName depend= Tcpip

# Set service account
Write-Host "Configuring service account..."
sc.exe config $ServiceName obj= "NT AUTHORITY\SYSTEM"

# Configure extended service properties
Write-Host "Configuring extended service properties..."
sc.exe privs $ServiceName SeCreateGlobalPrivilege/SeImpersonatePrivilege

# Start service
Write-Host "Starting service..."
Start-Service -Name $ServiceName

# Verify service status
$service = Get-Service -Name $ServiceName
Write-Host "Service status: $($service.Status)"

if ($service.Status -ne "Running") {
    Write-Error "Service failed to start"
    exit 1
}

Write-Host "Service installation completed successfully"
