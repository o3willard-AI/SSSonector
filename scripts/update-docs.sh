#!/bin/bash
set -e

# Get version from Makefile
VERSION=$(grep "VERSION :=" Makefile | cut -d' ' -f3)
DOCS_DIR="docs"
REPO_ROOT=$(cd "$(dirname "$0")/.." && pwd)

echo "Updating documentation for v${VERSION}..."

# Function to update version in markdown files
update_version() {
    local file="$1"
    sed -i "s/\(version \)[0-9]\+\.[0-9]\+\.[0-9]\+/\1${VERSION}/g" "$file"
    sed -i "s/\(v\)[0-9]\+\.[0-9]\+\.[0-9]\+/\1${VERSION}/g" "$file"
}

# Update version numbers in all markdown files
find "$DOCS_DIR" -name "*.md" -type f -exec bash -c 'update_version "$0"' {} \;
update_version "README.md"

# Generate platform-specific installation guides
cat > "$DOCS_DIR/linux_install.md" << EOF
# Linux Installation Guide

## Requirements
- Linux (Debian/Ubuntu/RHEL/CentOS)
- Root privileges
- systemd

## Installation

### Debian/Ubuntu
\`\`\`bash
# Download the package
wget https://github.com/o3willard-AI/SSSonector/releases/download/v${VERSION}/sssonector_${VERSION}_amd64.deb

# Install dependencies
sudo apt-get update
sudo apt-get install -y openssl

# Install package
sudo dpkg -i sssonector_${VERSION}_amd64.deb
sudo apt-get install -f
\`\`\`

### RHEL/CentOS
\`\`\`bash
# Download the package
wget https://github.com/o3willard-AI/SSSonector/releases/download/v${VERSION}/sssonector-${VERSION}-1.x86_64.rpm

# Install package
sudo yum install sssonector-${VERSION}-1.x86_64.rpm
\`\`\`

## Configuration

1. Generate certificates (if not using existing ones):
\`\`\`bash
sudo sssonector-cli generate-certs
\`\`\`

2. Edit configuration:
\`\`\`bash
sudo nano /etc/sssonector/config.yaml
\`\`\`

3. Start service:
\`\`\`bash
sudo systemctl start sssonector
sudo systemctl enable sssonector
\`\`\`

## Verification

1. Check service status:
\`\`\`bash
sudo systemctl status sssonector
\`\`\`

2. View logs:
\`\`\`bash
sudo journalctl -u sssonector -f
\`\`\`

3. Check network interface:
\`\`\`bash
ip addr show tun0
\`\`\`
EOF

cat > "$DOCS_DIR/macos_install.md" << EOF
# macOS Installation Guide

## Requirements
- macOS 10.15 or later
- Administrator privileges

## Installation

1. Download the installer:
\`\`\`bash
curl -LO https://github.com/o3willard-AI/SSSonector/releases/download/v${VERSION}/sssonector-${VERSION}.pkg
\`\`\`

2. Install the package:
\`\`\`bash
sudo installer -pkg sssonector-${VERSION}.pkg -target /
\`\`\`

## Configuration

1. Generate certificates (if not using existing ones):
\`\`\`bash
sudo sssonector-cli generate-certs
\`\`\`

2. Edit configuration:
\`\`\`bash
sudo nano /etc/sssonector/config.yaml
\`\`\`

3. Start service:
\`\`\`bash
sudo launchctl load /Library/LaunchDaemons/com.o3willard.sssonector.plist
\`\`\`

## Verification

1. Check service status:
\`\`\`bash
sudo launchctl list | grep sssonector
\`\`\`

2. View logs:
\`\`\`bash
tail -f /var/log/sssonector/sssonector.log
\`\`\`

3. Check network interface:
\`\`\`bash
ifconfig tun0
\`\`\`
EOF

cat > "$DOCS_DIR/windows_install.md" << EOF
# Windows Installation Guide

## Requirements
- Windows 10 or later (64-bit)
- Administrator privileges
- PowerShell 5.1 or later

## Installation

1. Download the installer from:
   https://github.com/o3willard-AI/SSSonector/releases/download/v${VERSION}/sssonector-${VERSION}-setup.exe

2. Run the installer with administrator privileges

## Configuration

1. Generate certificates (if not using existing ones):
\`\`\`powershell
# Run PowerShell as Administrator
sssonector-cli generate-certs
\`\`\`

2. Edit configuration:
\`\`\`powershell
notepad C:\ProgramData\SSSonector\config.yaml
\`\`\`

3. Start service:
\`\`\`powershell
Start-Service SSSonector
Set-Service SSSonector -StartupType Automatic
\`\`\`

## Verification

1. Check service status:
\`\`\`powershell
Get-Service SSSonector
\`\`\`

2. View logs:
\`\`\`powershell
Get-Content -Path "C:\ProgramData\SSSonector\logs\sssonector.log" -Wait
\`\`\`

3. Check network interface:
\`\`\`powershell
Get-NetAdapter | Where-Object {$_.InterfaceDescription -like "*SSSonector*"}
\`\`\`
EOF

echo "Documentation updated successfully!"
