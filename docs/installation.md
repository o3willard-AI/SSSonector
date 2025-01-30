# SSSonector Installation Guide

## Download Installers

All installer packages are available on the [releases page](https://github.com/o3willard-AI/SSSonector/releases/tag/v1.0.0):

- Linux (Debian/Ubuntu): `sssonector_1.0.0_amd64.deb`
- Linux (RHEL/CentOS): `sssonector-1.0.0-1.x86_64.rpm`
- Windows: `sssonector-1.0.0-setup.exe`
- macOS: See [macOS Build Guide](macos_build.md) for build instructions

## Installation Instructions

### Linux (Debian/Ubuntu)
```bash
# Download the package
wget https://github.com/o3willard-AI/SSSonector/releases/download/v1.0.0/sssonector_1.0.0_amd64.deb

# Install the package
sudo dpkg -i sssonector_1.0.0_amd64.deb
sudo apt-get install -f  # Install any missing dependencies
```

### Linux (RHEL/CentOS)
```bash
# Download the package
wget https://github.com/o3willard-AI/SSSonector/releases/download/v1.0.0/sssonector-1.0.0-1.x86_64.rpm

# Install the package
sudo yum install sssonector-1.0.0-1.x86_64.rpm
```

### Windows
1. Download `sssonector-1.0.0-setup.exe` from the [releases page](https://github.com/o3willard-AI/SSSonector/releases/tag/v1.0.0)
2. Run the installer with administrator privileges
3. Follow the installation wizard

### macOS
The macOS package is not yet available. Please follow the [macOS Build Guide](macos_build.md) to build from source.

## Verifying Installation

After installation, verify that SSSonector is installed correctly:

```bash
# Check version
sssonector --version

# Check service status (Linux)
systemctl status sssonector

# Check service status (macOS)
launchctl list | grep sssonector

# Check service status (Windows)
sc query sssonector
```

## Troubleshooting

If you encounter any issues during installation:

1. Check system requirements:
   - Linux: Debian 11+/Ubuntu 22.04+ or RHEL 8+/CentOS 8+
   - Windows: Windows 10/11 64-bit
   - macOS: macOS 11+ (Big Sur)

2. Common issues:
   - Missing dependencies: Run `sudo apt-get install -f` (Debian/Ubuntu)
   - Permission issues: Ensure you have administrator/root privileges
   - Service not starting: Check system logs for errors

3. For additional help:
   - Check the [troubleshooting guide](troubleshooting.md)
   - Create an issue on GitHub with:
     - Your OS version and architecture
     - Installation method used
     - Error messages or logs
     - Steps to reproduce the issue
