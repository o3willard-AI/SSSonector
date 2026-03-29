# SSSonector Installation Guide

This guide covers installation, upgrade, and uninstallation of SSSonector.

## Prerequisites

### System Requirements
- Linux (primary platform), macOS, or Windows
- TUN/TAP kernel module support (Linux)
- Administrative privileges for network interface creation

### Platform-specific Requirements

#### Linux
- Ubuntu 20.04+, CentOS 7+, or RHEL 8+
- `sudo` privileges
- TUN/TAP kernel module loaded

#### macOS
- macOS 10.15 (Catalina) or later
- Xcode Command Line Tools
- Network Extension entitlements

#### Windows
- Windows 10 or Server 2016+
- TAP-Windows Adapter V9
- Administrator privileges

## Installation

### One-line Install (Recommended)

```bash
curl -fsSL https://raw.githubusercontent.com/o3willard-AI/SSSonector/main/install.sh | sudo bash
```

The installer will:
1. Detect your OS and architecture
2. Download the latest release binary from GitHub
3. Prompt for configuration (server/client mode, instance name, addresses)
4. Generate TLS certificates
5. Install systemd service template

### Non-interactive Install

For automated deployments, use environment variables:

```bash
# Server mode
curl -fsSL https://raw.githubusercontent.com/o3willard-AI/SSSonector/main/install.sh | \
  sudo SSSONECTOR_MODE=server \
       SSSONECTOR_INSTANCE=tunnel-a \
       SSSONECTOR_ADDRESS=10.0.1.1/24 \
       bash

# Client mode
curl -fsSL https://raw.githubusercontent.com/o3willard-AI/SSSonector/main/install.sh | \
  sudo SSSONECTOR_MODE=client \
       SSSONECTOR_INSTANCE=tunnel-a \
       SSSONECTOR_ADDRESS=10.0.1.2/24 \
       SSSONECTOR_SERVER=your-server-ip \
       bash
```

### Install Options

| Environment Variable | Description | Default |
|---------------------|-------------|---------|
| `SSSONECTOR_VERSION` | Version to install | latest |
| `SSSONECTOR_MODE` | "server" or "client" | interactive |
| `SSSONECTOR_INSTANCE` | Instance name | interactive |
| `SSSONECTOR_ADDRESS` | TUN interface address (CIDR) | interactive |
| `SSSONECTOR_SERVER` | Server address (client mode) | interactive |
| `SSSONECTOR_PORT` | Listen/connect port | auto-assigned |
| `SSSONECTOR_NO_SERVICE` | Don't install systemd service | false |

### Manual Download

Download binaries from the [releases page](https://github.com/o3willard-AI/SSSonector/releases):

| Platform | Architecture | Binary |
|----------|-------------|--------|
| Linux | amd64 | `sssonector-linux-amd64` |
| Linux | arm64 | `sssonector-linux-arm64` |
| macOS | amd64 | `sssonector-darwin-amd64` |
| macOS | arm64 | `sssonector-darwin-arm64` |
| Windows | amd64 | `sssonector-windows-amd64.exe` |

```bash
curl -LO https://github.com/o3willard-AI/SSSonector/releases/latest/download/sssonector-linux-amd64
chmod +x sssonector-linux-amd64
sudo mv sssonector-linux-amd64 /usr/local/bin/sssonector
```

### From Source

```bash
git clone https://github.com/o3willard-AI/SSSonector.git
cd SSSonector
make build
sudo make install
```

## Upgrade

### One-line Upgrade

```bash
curl -fsSL https://raw.githubusercontent.com/o3willard-AI/SSSonector/main/upgrade.sh | sudo bash
```

The upgrade script will:
1. Check current installed version
2. Download the new binary from GitHub releases
3. Stop running instances
4. Replace the binary
5. Restart instances that were running

### Upgrade Options

| Option | Description |
|--------|-------------|
| `--version VERSION` | Upgrade to specific version (default: latest) |
| `--no-restart` | Don't restart services after upgrade |
| `-y, --yes` | Don't ask for confirmation |

### Upgrade Examples

```bash
# Upgrade to latest version
curl -fsSL https://raw.githubusercontent.com/o3willard-AI/SSSonector/main/upgrade.sh | sudo bash

# Upgrade to specific version
curl -fsSL https://raw.githubusercontent.com/o3willard-AI/SSSonector/main/upgrade.sh | \
  sudo bash -s -- --version v1.2.0

# Upgrade without restarting services
curl -fsSL https://raw.githubusercontent.com/o3willard-AI/SSSonector/main/upgrade.sh | \
  sudo bash -s -- --no-restart

# Non-interactive upgrade
curl -fsSL https://raw.githubusercontent.com/o3willard-AI/SSSonector/main/upgrade.sh | \
  sudo bash -s -- -y
```

### Manual Upgrade

```bash
# Stop running instances
sudo systemctl stop sssonector@*

# Download new binary
curl -LO https://github.com/o3willard-AI/SSSonector/releases/latest/download/sssonector-linux-amd64
chmod +x sssonector-linux-amd64

# Replace binary
sudo mv sssonector-linux-amd64 /usr/local/bin/sssonector

# Restart instances
sudo systemctl start sssonector@instance-name
```

## Uninstall

### One-line Uninstall

```bash
# Remove binary and systemd service (preserves instances)
curl -fsSL https://raw.githubusercontent.com/o3willard-AI/SSSonector/main/uninstall.sh | sudo bash
```

### Uninstall Options

| Option | Description |
|--------|-------------|
| (default) | Removes binary and systemd service template, preserves instance configs |
| `--purge` | Removes everything: binary, service, configs, logs, certificates |
| `--all` | Removes all instance configurations |
| `-y, --yes` | Skip confirmation prompt |

### Uninstall Examples

```bash
# Remove binary and service only (preserves configurations)
curl -fsSL https://raw.githubusercontent.com/o3willard-AI/SSSonector/main/uninstall.sh | sudo bash

# Remove everything (complete cleanup)
curl -fsSL https://raw.githubusercontent.com/o3willard-AI/SSSonector/main/uninstall.sh | \
  sudo bash -s -- --purge

# Remove all instances but keep binary
curl -fsSL https://raw.githubusercontent.com/o3willard-AI/SSSonector/main/uninstall.sh | \
  sudo bash -s -- --all

# Non-interactive full uninstall
curl -fsSL https://raw.githubusercontent.com/o3willard-AI/SSSonector/main/uninstall.sh | \
  sudo bash -s -- -y --purge
```

### Manual Uninstall

```bash
# Stop all instances
sudo systemctl stop sssonector@*

# Disable all instances
sudo systemctl disable sssonector@*

# Remove binary
sudo rm /usr/local/bin/sssonector

# Remove systemd service template
sudo rm /etc/systemd/system/sssonector@.service
sudo systemctl daemon-reload

# Optional: Remove all configurations and certificates
sudo rm -rf /etc/sssonector
sudo rm -rf /var/log/sssonector
```

## Post-Installation

### Verify Installation

```bash
# Check version
sssonector -version

# List instances
ls /etc/sssonector/instances/
```

### Start Services

```bash
# Start a specific instance
sudo systemctl start sssonector@<instance-name>

# Enable at boot
sudo systemctl enable sssonector@<instance-name>

# View status
sudo systemctl status sssonector@<instance-name>

# View logs
journalctl -u sssonector@<instance-name> -f
```

## Troubleshooting

### Common Issues

1. **Instance won't start**
   ```bash
   # Check logs
   journalctl -u sssonector@instance-name -n 50
   
   # Check if port is in use
   sudo netstat -tlnp | grep 8443
   
   # Verify configuration
   sssonector -config /etc/sssonector/instances/instance-name/config.yaml -validate
   ```

2. **Connection refused**
   ```bash
   # Verify server is listening
   sudo netstat -tlnp | grep sssonector
   
   # Check firewall
   sudo ufw status
   
   # Test connectivity
   telnet server-ip 8443
   ```

3. **Certificate errors**
   ```bash
   # Verify certificate files exist
   ls -la /etc/sssonector/instances/instance-name/certs/
   
   # Check certificate validity
   openssl x509 -in /etc/sssonector/instances/instance-name/certs/server.crt -text -noout
   
   # Verify certificate chain
   openssl verify -CAfile /etc/sssonector/instances/instance-name/certs/ca.crt \
     /etc/sssonector/instances/instance-name/certs/server.crt
   ```

## Support

- Issues: https://github.com/o3willard-AI/SSSonector/issues
