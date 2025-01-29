# Requires -RunAsAdministrator

$ServiceName = "SSSonector"
$DisplayName = "SSSonector Service"
$Description = "Secure SSL tunneling service for network traffic"
$BinaryPath = "`"C:\Program Files\SSSonector\SSSonector.exe`" --config `"C:\ProgramData\SSSonector\config.yaml`""

# Create required directories
$null = New-Item -ItemType Directory -Force -Path "C:\Program Files\SSSonector"
$null = New-Item -ItemType Directory -Force -Path "C:\ProgramData\SSSonector"
$null = New-Item -ItemType Directory -Force -Path "C:\ProgramData\SSSonector\logs"

# Copy files
Copy-Item -Path ".\SSSonector.exe" -Destination "C:\Program Files\SSSonector\" -Force
Copy-Item -Path ".\configs\*.yaml" -Destination "C:\ProgramData\SSSonector\" -Force

# Create service
$service = Get-Service -Name $ServiceName -ErrorAction SilentlyContinue

if ($service) {
    Write-Host "Service already exists. Stopping and removing..."
    Stop-Service -Name $ServiceName -Force
    $service.WaitForStatus('Stopped', '00:00:30')
    sc.exe delete $ServiceName
    Start-Sleep -Seconds 2
}

Write-Host "Creating service..."
$result = sc.exe create $ServiceName binPath= $BinaryPath start= auto DisplayName= $DisplayName
if ($LASTEXITCODE -ne 0) {
    Write-Error "Failed to create service: $result"
    exit 1
}

# Set description
$result = sc.exe description $ServiceName $Description
if ($LASTEXITCODE -ne 0) {
    Write-Error "Failed to set service description: $result"
    exit 1
}

# Configure recovery options (restart on failure)
$result = sc.exe failure $ServiceName reset= 86400 actions= restart/60000/restart/60000/restart/60000
if ($LASTEXITCODE -ne 0) {
    Write-Error "Failed to set service recovery options: $result"
    exit 1
}

# Set required privileges
$result = sc.exe privs $ServiceName SeCreateTokenPrivilege/SeAssignPrimaryTokenPrivilege/SeChangeNotifyPrivilege/SeNetworkLogonRight
if ($LASTEXITCODE -ne 0) {
    Write-Error "Failed to set service privileges: $result"
    exit 1
}

# Start service
Write-Host "Starting service..."
Start-Service -Name $ServiceName
$service = Get-Service -Name $ServiceName
$service.WaitForStatus('Running', '00:00:30')

if ($service.Status -eq 'Running') {
    Write-Host "Service installed and started successfully"
} else {
    Write-Error "Service failed to start. Check system logs for details."
    exit 1
}

# Create scheduled task for certificate renewal checks
$Action = New-ScheduledTaskAction -Execute "C:\Program Files\SSSonector\SSSonector.exe" -Argument "--check-certs"
$Trigger = New-ScheduledTaskTrigger -Daily -At 12am
$Principal = New-ScheduledTaskPrincipal -UserID "NT AUTHORITY\SYSTEM" -LogonType ServiceAccount -RunLevel Highest
$Settings = New-ScheduledTaskSettingsSet -ExecutionTimeLimit (New-TimeSpan -Minutes 5) -RestartCount 3 -RestartInterval (New-TimeSpan -Minutes 1)

Register-ScheduledTask -TaskName "SSSonector Certificate Check" -Action $Action -Trigger $Trigger -Principal $Principal -Settings $Settings -Description "Check SSSonector certificates for expiration" -Force

Write-Host "Installation complete. Service is running and certificate checks are scheduled."
