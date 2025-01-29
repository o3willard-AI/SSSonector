# Requires -RunAsAdministrator

$ServiceName = "SSSonector"
$Description = "Secure Scalable SSL Connector Service"
$BinaryPath = "`"C:\Program Files\SSSonector\sssonector.exe`" --config `"C:\ProgramData\SSSonector\config.yaml`""

# Create necessary directories
New-Item -Path "C:\Program Files\SSSonector" -ItemType Directory -Force
New-Item -Path "C:\ProgramData\SSSonector" -ItemType Directory -Force
New-Item -Path "C:\ProgramData\SSSonector\certs" -ItemType Directory -Force
New-Item -Path "C:\ProgramData\SSSonector\logs" -ItemType Directory -Force

# Copy files
Copy-Item -Path ".\sssonector.exe" -Destination "C:\Program Files\SSSonector\" -Force
Copy-Item -Path ".\configs\*.yaml" -Destination "C:\ProgramData\SSSonector\" -Force

# Set permissions
$Acl = Get-Acl "C:\ProgramData\SSSonector"
$Ar = New-Object System.Security.AccessControl.FileSystemAccessRule("NT AUTHORITY\NETWORK SERVICE", "Modify", "ContainerInherit,ObjectInherit", "None", "Allow")
$Acl.SetAccessRule($Ar)
Set-Acl "C:\ProgramData\SSSonector" $Acl

# Remove existing service if it exists
if (Get-Service $ServiceName -ErrorAction SilentlyContinue) {
    Write-Host "Removing existing service..."
    Stop-Service $ServiceName
    sc.exe delete $ServiceName
    Start-Sleep -Seconds 2
}

# Create new service
Write-Host "Creating service..."
sc.exe create $ServiceName binPath= $BinaryPath start= auto
sc.exe description $ServiceName $Description
sc.exe failure $ServiceName reset= 86400 actions= restart/30000/restart/30000/restart/30000

# Configure service account and dependencies
sc.exe config $ServiceName obj= "NT AUTHORITY\NETWORK SERVICE" password= ""
sc.exe config $ServiceName depend= Tcpip/Afd/NetBT

# Create scheduled task for certificate renewal checks
$Action = New-ScheduledTaskAction -Execute "C:\Program Files\SSSonector\sssonector.exe" -Argument "--check-certs"
$Trigger = New-ScheduledTaskTrigger -Daily -At 12am
$Principal = New-ScheduledTaskPrincipal -UserID "NT AUTHORITY\NETWORK SERVICE" -LogonType ServiceAccount -RunLevel Highest
Register-ScheduledTask -TaskName "SSSonector-CertRenewal" -Action $Action -Trigger $Trigger -Principal $Principal -Description "Check and renew SSL certificates" -Force

Write-Host "Starting service..."
Start-Service $ServiceName

Write-Host "Service installation complete!"
Write-Host "Service Name: $ServiceName"
Write-Host "Status: $((Get-Service $ServiceName).Status)"
