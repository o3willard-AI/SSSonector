# Requires -RunAsAdministrator

# Configuration
$ServiceName = "SSSonector"
$DisplayName = "SSSonector Secure Tunnel Service"
$Description = "Enterprise-grade secure tunnel service"
$BinaryPath = Join-Path $env:ProgramFiles "SSSonector\sssonector.exe"
$ConfigPath = Join-Path $env:ProgramData "SSSonector"
$LogPath = Join-Path $env:ProgramData "SSSonector\logs"
$CertPath = Join-Path $env:ProgramData "SSSonector\certs"

# Ensure running as administrator
$currentPrincipal = New-Object Security.Principal.WindowsPrincipal([Security.Principal.WindowsIdentity]::GetCurrent())
if (-not $currentPrincipal.IsInRole([Security.Principal.WindowsBuiltInRole]::Administrator)) {
    Write-Error "This script must be run as Administrator"
    exit 1
}

# Create required directories
Write-Host "Creating directories..."
$Directories = @(
    $ConfigPath,
    $LogPath,
    $CertPath,
    (Split-Path $BinaryPath -Parent)
)

foreach ($Dir in $Directories) {
    if (-not (Test-Path $Dir)) {
        New-Item -ItemType Directory -Path $Dir | Out-Null
    }
}

# Copy files
Write-Host "Installing files..."
Copy-Item -Path "bin\sssonector.exe" -Destination $BinaryPath -Force
Copy-Item -Path "bin\sssonectorctl.exe" -Destination (Join-Path $env:ProgramFiles "SSSonector\sssonectorctl.exe") -Force

# Install example config if not exists
$ConfigFile = Join-Path $ConfigPath "config.yaml"
if (-not (Test-Path $ConfigFile)) {
    Copy-Item -Path "config\config.yaml.example" -Destination $ConfigFile
}

# Set up service account
$ServiceAccount = "NT SERVICE\$ServiceName"
Write-Host "Setting up service account..."

# Create service
Write-Host "Installing service..."
$ServiceParams = @{
    Name = $ServiceName
    BinaryPathName = "`"$BinaryPath`" -config `"$ConfigFile`""
    DisplayName = $DisplayName
    Description = $Description
    StartupType = "Automatic"
}

$Service = Get-Service -Name $ServiceName -ErrorAction SilentlyContinue
if ($Service) {
    Write-Host "Service already exists, stopping and removing..."
    Stop-Service -Name $ServiceName -Force
    $Service.WaitForStatus('Stopped', '00:00:30')
    sc.exe delete $ServiceName
    Start-Sleep -Seconds 2
}

New-Service @ServiceParams

# Configure service security
Write-Host "Configuring service security..."
$SD = New-Object Security.SecurityDescriptor
$SD.SetSecurityDescriptorSddlForm("D:(A;;CCLCSWRPWPDTLOCRRC;;;SY)(A;;CCDCLCSWRPWPDTLOCRSDRCWDWO;;;BA)(A;;CCLCSWLOCRRC;;;IU)(A;;CCLCSWLOCRRC;;;SU)")
$Service = Get-WmiObject -Class Win32_Service -Filter "Name='$ServiceName'"
$Service.Change($null, $null, $null, $null, $null, $null, $null, $null, $null, $null, $SD.GetSddlForm("All"))

# Set up permissions
Write-Host "Setting up permissions..."
$Acl = Get-Acl $ConfigPath
$Rule = New-Object System.Security.AccessControl.FileSystemAccessRule($ServiceAccount, "ReadAndExecute", "ContainerInherit,ObjectInherit", "None", "Allow")
$Acl.AddAccessRule($Rule)
Set-Acl $ConfigPath $Acl

$Acl = Get-Acl $LogPath
$Rule = New-Object System.Security.AccessControl.FileSystemAccessRule($ServiceAccount, "Modify", "ContainerInherit,ObjectInherit", "None", "Allow")
$Acl.AddAccessRule($Rule)
Set-Acl $LogPath $Acl

# Configure event log
Write-Host "Configuring event log..."
if (-not [System.Diagnostics.EventLog]::SourceExists($ServiceName)) {
    New-EventLog -LogName Application -Source $ServiceName
}

# Configure firewall
Write-Host "Configuring firewall..."
$FirewallRule = Get-NetFirewallRule -DisplayName $ServiceName -ErrorAction SilentlyContinue
if (-not $FirewallRule) {
    New-NetFirewallRule -DisplayName $ServiceName `
        -Direction Inbound `
        -Action Allow `
        -Protocol TCP `
        -LocalPort 8443 `
        -Program $BinaryPath `
        -Description "Allow incoming connections to $ServiceName"
}

# Start service if requested
$StartService = Read-Host "Start service now? (y/N)"
if ($StartService -eq "y") {
    Write-Host "Starting service..."
    Start-Service -Name $ServiceName
    Write-Host "Service started. Check status with: Get-Service $ServiceName"
} else {
    Write-Host "Service installed but not started. Start manually with: Start-Service $ServiceName"
}

Write-Host "`nInstallation complete!"
Write-Host "`nNext steps:"
Write-Host "1. Edit configuration: $ConfigFile"
Write-Host "2. Install certificates in: $CertPath"
Write-Host "3. Start service: Start-Service $ServiceName"
Write-Host "4. Check status: Get-Service $ServiceName"
Write-Host "5. View logs: Get-EventLog -LogName Application -Source $ServiceName"
Write-Host "`nControl service with: sssonectorctl [command]"

# Add to PATH if requested
$AddPath = Read-Host "Add sssonectorctl to PATH? (y/N)"
if ($AddPath -eq "y") {
    $BinPath = Split-Path $BinaryPath -Parent
    $CurrentPath = [Environment]::GetEnvironmentVariable("Path", "Machine")
    if (-not $CurrentPath.Contains($BinPath)) {
        [Environment]::SetEnvironmentVariable("Path", "$CurrentPath;$BinPath", "Machine")
        Write-Host "Added to PATH. Please restart your shell for changes to take effect."
    }
}
