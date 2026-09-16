#Requires -RunAsAdministrator
<#
 SSSonector Install Script (Windows)
 Downloads and installs SSSonector from GitHub releases, mirroring install.sh.

 Usage:
   powershell -File scripts\install.ps1
   powershell -File scripts\install.ps1 -Version v1.0.0 -Arch amd64 -AddPath y

 Parameters:
   -Version  Version to install (default: latest release from the GitHub API).
   -Arch     Architecture: amd64 or arm64 (default: detected from the OS).
   -AddPath  "y" to add the SSSonector binary directory to the machine PATH.
             If omitted, the script prompts (interactive) or skips (non-interactive).
   -NoService  If set, do not register the Windows service.
#>

param(
    [string]$Version = $env:SSSONECTOR_VERSION,
    [string]$Arch = $env:SSSONECTOR_ARCH,
    [string]$AddPath = "",
    [switch]$NoService
)

# Configuration
$Repo = "o3willard-AI/SSSonector"
$ReleaseBase = "https://github.com/${Repo}/releases/download"
$TemplateUrl = "https://raw.githubusercontent.com/${Repo}/main/templates"

$InstallDir = Join-Path $env:ProgramFiles "SSSonector"
$BinaryPath = Join-Path $InstallDir "sssonector.exe"
$ConfigPath = Join-Path $env:ProgramData "SSSonector"
$LogPath = Join-Path $env:ProgramData "SSSonector\logs"
$CertPath = Join-Path $env:ProgramData "SSSonector\certs"
$TemplatesDir = Join-Path $ConfigPath "templates"

$ServiceName = "SSSonector"
$DisplayName = "SSSonector Secure Tunnel Service"
$Description = "Enterprise-grade secure tunnel service"

# --- Helpers ----------------------------------------------------------------

function Log-Info([string]$Msg) { Write-Host "[INFO] $Msg" }
function Log-Step([string]$Msg) { Write-Host "==> $Msg" }
function Log-Error([string]$Msg) { [Console]::Error.WriteLine("[ERROR] $Msg") }

function Get-Arch {
    # Map the OS architecture to the install.sh naming (amd64/arm64).
    $machine = (uname -m) 2>/dev/null
    if ($null -eq $machine -or $machine -eq "") { $machine = $env:PROCESSOR_ARCHITECTURE }

    $machine = $machine.ToLower()
    if ($machine -eq "amd64" -or $machine -eq "x86_64" -or $machine -eq "x86-64" -or $machine -eq "x64") {
        return "amd64"
    }
    if ($machine -eq "aarch64" -or $machine -eq "arm64") {
        return "arm64"
    }
    Log-Error "Unsupported architecture: $machine"
    exit 1
}

function Get-LatestVersion {
    $headers = @{ "User-Agent" = "sssonector-installer" }
    try {
        $release = Invoke-RestMethod -Uri "https://api.github.com/repos/${Repo}/releases/latest" -Headers $headers
        return $release.tag_name
    }
    catch {
        Log-Error "Failed to fetch latest version from GitHub: $($_.Exception.Message)"
        exit 1
    }
}

# Downloads the binary and SHA256SUMS, verifies the checksum, and returns the
# path to the verified temporary binary (or aborts the install on a mismatch).
# Parameters are injectable so this function can be unit-tested in isolation.
# Fetches a remote URL or a local file path (file:// or bare filesystem path)
# into $OutFile. Local fetches let the checksum logic be exercised in tests
# without network access; production uses http(s) via Invoke-WebRequest.
function Fetch-Content {
    param(
        [string]$Url,
        [string]$OutFile
    )
    if ($Url.StartsWith("file://")) {
        Copy-Item -Path $Url.Substring(7) -Destination $OutFile -Force
    }
    elseif (Test-Path $Url) {
        Copy-Item -Path $Url -Destination $OutFile -Force
    }
    else {
        Invoke-WebRequest -Uri $Url -OutFile $OutFile -UseBasicParsing
    }
}

function Download-Verify-Binary {
    param(
        [string]$SumsUrl,
        [string]$BinaryUrl,
        [string]$BinaryName
    )

    $tmpDir = New-Item -ItemType Directory -Path (Join-Path $env:TEMP ("sssonector-" + (Get-Date -Format "yyyyMMddHHmmss"))) -Force
    $tmpSums = Join-Path $tmpDir.FullName "SHA256SUMS"
    $tmpBin  = Join-Path $tmpDir.FullName "sssonector.exe"

    Log-Step "Downloading SHA256SUMS from $SumsUrl..."
    try {
        Fetch-Content -Url $SumsUrl -OutFile $tmpSums
    }
    catch {
        Log-Error "Failed to download SHA256SUMS from $SumsUrl"
        Remove-Item -Recurse -Force $tmpDir.FullName
        exit 1
    }

    Log-Step "Downloading binary from $BinaryUrl..."
    try {
        Fetch-Content -Url $BinaryUrl -OutFile $tmpBin
    }
    catch {
        Log-Error "Failed to download binary from $BinaryUrl"
        Remove-Item -Recurse -Force $tmpDir.FullName
        exit 1
    }

    Log-Step "Verifying binary checksum..."
    $expected = ""
    foreach ($line in (Get-Content $tmpSums)) {
        # SHA256SUMS lines are "<hash>  <filename>"; match the bare filename.
        $parts = ($line -split "\s+")
        if ($parts.Count -ge 2 -and $parts[-1] -eq $BinaryName) {
            $expected = $parts[0].Trim().ToLower()
            break
        }
    }
    if ($expected -eq "") {
        Log-Error "No checksum found for $BinaryName in SHA256SUMS"
        Remove-Item -Recurse -Force $tmpDir.FullName
        exit 1
    }

    $actual = (Get-FileHash -Path $tmpBin -Algorithm SHA256).Hash.ToLower()

    if ($actual -ne $expected) {
        Log-Error "Checksum mismatch: expected $expected, got $actual. Binary NOT installed."
        Remove-Item -Recurse -Force $tmpDir.FullName
        exit 1
    }

    Remove-Item -Force $tmpSums
    return $tmpBin
}

function Install-Binary([string]$TmpBinary) {
    Log-Step "Installing binary to $InstallDir..."
    if (-not (Test-Path $InstallDir)) { New-Item -ItemType Directory -Path $InstallDir | Out-Null }
    Copy-Item -Path $TmpBinary -Destination $BinaryPath -Force
    Remove-Item -Force $TmpBinary
    Log-Info "Binary installed: $BinaryPath"
}

function Download-Templates {
    Log-Step "Downloading configuration templates..."
    if (-not (Test-Path $TemplatesDir)) { New-Item -ItemType Directory -Path $TemplatesDir | Out-Null }

    foreach ($name in @("server.yaml.template", "client.yaml.template")) {
        try {
            Invoke-WebRequest -Uri "${TemplateUrl}/$name" -OutFile (Join-Path $TemplatesDir $name) -UseBasicParsing
        }
        catch {
            Log-Error "Failed to download template $name"
            exit 1
        }
    }
    Log-Info "Templates installed: $TemplatesDir"
}

# --- Main -------------------------------------------------------------------

# Ensure running as administrator
$currentPrincipal = New-Object Security.Principal.WindowsPrincipal([Security.Principal.WindowsIdentity]::GetCurrent())
if (-not $currentPrincipal.IsInRole([Security.Principal.WindowsBuiltInRole]::Administrator)) {
    Log-Error "This script must be run as Administrator"
    exit 1
}

$Arch = Get-Arch
if ($Version -eq "") {
    $Version = Get-LatestVersion
}

Log-Step "SSSonector $Version Installer for Windows ($Arch)"
Write-Host ""

# Create required directories
Log-Step "Creating directories..."
$Directories = @(
    $ConfigPath,
    $LogPath,
    $CertPath,
    $TemplatesDir,
    $InstallDir
)
foreach ($Dir in $Directories) {
    if (-not (Test-Path $Dir)) { New-Item -ItemType Directory -Path $Dir | Out-Null }
}

# Download + verify + install binary (asset naming matches install.sh)
$BinaryName = "sssonector-windows-${Arch}.exe"
$BinaryUrl = "${ReleaseBase}/${Version}/${BinaryName}"
$SumsUrl = "${ReleaseBase}/${Version}/SHA256SUMS"

$tmpBinary = Download-Verify-Binary -SumsUrl $SumsUrl -BinaryUrl $BinaryUrl -BinaryName $BinaryName
Install-Binary $tmpBinary

# Download the config templates from the repo
Download-Templates

# Install example config if not exists
$ConfigFile = Join-Path $ConfigPath "config.yaml"
if (-not (Test-Path $ConfigFile)) {
    Copy-Item -Path (Join-Path $TemplatesDir "server.yaml.template") -Destination $ConfigFile
}

# --- Service registration (if requested) -----------------------------------
if (-not $NoService) {
    # Set up service account
    $ServiceAccount = "NT SERVICE\${ServiceName}"
    Write-Host "Setting up service account..."

    # Create service (idempotent: remove any existing instance first)
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
}

# --- Add binary directory to machine PATH ----------------------------------
if ($AddPath -eq "") {
    # Prompt only in an interactive terminal; otherwise default to skip.
    if ([Console]::IsTerminal()) {
        $AddPath = Read-Host "Add SSSonector to the system PATH? (y/N)"
    }
    else {
        $AddPath = "n"
    }
}

if ($AddPath -Match "^(y|Y|yes|Yes|YES)$") {
    $BinPath = Split-Path $BinaryPath -Parent
    $CurrentPath = [Environment]::GetEnvironmentVariable("Path", "Machine")
    if ($null -eq $CurrentPath -or -not $CurrentPath.ToLower().Contains($BinPath.ToLower())) {
        [Environment]::SetEnvironmentVariable("Path", "$CurrentPath;$BinPath", "Machine")
        Write-Host "Added $BinPath to PATH. Please restart your shell for changes to take effect."
    }
    else {
        Write-Host "$BinPath is already on the system PATH."
    }
}

# Start service if requested
if (-not $NoService) {
    $StartService = Read-Host "Start service now? (y/N)"
    if ($StartService -eq "y" -or $StartService -eq "Y") {
        Write-Host "Starting service..."
        Start-Service -Name $ServiceName
        Write-Host "Service started. Check status with: Get-Service $ServiceName"
    } else {
        Write-Host "Service installed but not started. Start manually with: Start-Service $ServiceName"
    }
}

Write-Host "`nInstallation complete!"
Write-Host "`nNext steps:"
Write-Host "1. Edit configuration: $ConfigFile"
Write-Host "2. Install certificates in: $CertPath"
Write-Host "3. Start service: Start-Service $ServiceName"
Write-Host "4. Check status: Get-Service $ServiceName"
Write-Host "5. View logs: Get-EventLog -LogName Application -Source $ServiceName"