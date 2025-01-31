# Linux Installation Guide for SSSonector

This guide covers installation instructions for various Linux distributions.

⚠️ **Important:** All installer packages are distributed through [GitHub Releases](https://github.com/o3willard-AI/SSSonector/releases/tag/v1.0.0). Always use the GitHub Releases URLs for downloading packages. Do not use repository URLs (like `/blob/main/dist/...`) as they will not work with wget or other download tools.

## Debian/Ubuntu

```bash
# Download the package
wget https://github.com/o3willard-AI/SSSonector/releases/download/v1.0.0/sssonector_1.0.0_amd64.deb

# Verify the download URL worked (should show sssonector_1.0.0_amd64.deb)
ls sssonector_1.0.0_amd64.deb

# Install the package
sudo dpkg -i sssonector_1.0.0_amd64.deb
sudo apt-get install -f  # Install dependencies if needed
```

## Red Hat Enterprise Linux (RHEL) / Rocky Linux

### Using RPM Package

```bash
# Download the package
wget https://github.com/o3willard-AI/SSSonector/releases/download/v1.0.0/sssonector-1.0.0-1.x86_64.rpm

# Verify the download URL worked (should show sssonector-1.0.0-1.x86_64.rpm)
ls sssonector-1.0.0-1.x86_64.rpm

# Install the package
sudo dnf install sssonector-1.0.0-1.x86_64.rpm
```

### From Source

1. Install build dependencies:
```bash
# RHEL/Rocky Linux 8
sudo dnf groupinstall "Development Tools"
sudo dnf install golang openssl-devel systemd-devel

# RHEL/Rocky Linux 9
sudo dnf groupinstall "Development Tools"
sudo dnf install golang openssl-devel systemd-devel
```

2. Build and install:
```bash
git clone https://github.com/o3willard-AI/SSSonector.git
cd SSSonector
make
sudo make install
```

## Post-Installation Steps

### 1. Generate or Install Certificates

```bash
# Generate test certificates (for development)
./scripts/generate-certs.sh

# Copy certificates
sudo mkdir -p /etc/sssonector/certs
sudo cp certs/ca.crt certs/server.crt certs/server.key /etc/sssonector/certs/
sudo chmod 600 /etc/sssonector/certs/*.key
sudo chmod 644 /etc/sssonector/certs/*.crt
```

### 2. Configure the Service

#### Server Mode
```bash
sudo cp /etc/sssonector/server.yaml.example /etc/sssonector/config.yaml
sudo vi /etc/sssonector/config.yaml

# Configure with your settings:
mode: "server"
network:
  interface: "tun0"
  address: "10.0.0.1"
  mtu: 1500
tunnel:
  cert_file: "/etc/sssonector/certs/server.crt"
  key_file: "/etc/sssonector/certs/server.key"
  ca_file: "/etc/sssonector/certs/ca.crt"
  listen_address: "0.0.0.0"
  listen_port: 8443
```

#### Client Mode
```bash
sudo cp /etc/sssonector/client.yaml.example /etc/sssonector/config.yaml
sudo vi /etc/sssonector/config.yaml

# Configure with your settings:
mode: "client"
network:
  interface: "tun0"
  address: "10.0.0.2"
  mtu: 1500
tunnel:
  cert_file: "/etc/sssonector/certs/client.crt"
  key_file: "/etc/sssonector/certs/client.key"
  ca_file: "/etc/sssonector/certs/ca.crt"
  server_address: "tunnel.example.com"
  server_port: 8443
```

### 3. Start and Enable the Service

```bash
# Start the service
sudo systemctl start sssonector

# Enable auto-start on boot
sudo systemctl enable sssonector

# Check service status
sudo systemctl status sssonector
```

### 4. View Logs

```bash
# View service logs
sudo journalctl -u sssonector -f

# View application logs
sudo tail -f /var/log/sssonector/service.log
```

## Firewall Configuration

### RHEL/Rocky Linux (firewalld)

```bash
# Allow SSL tunnel port (server mode)
sudo firewall-cmd --permanent --add-port=8443/tcp
sudo firewall-cmd --reload

# Allow forwarding (if needed)
sudo firewall-cmd --permanent --add-masquerade
sudo firewall-cmd --reload
```

### Ubuntu/Debian (ufw)

```bash
# Allow SSL tunnel port (server mode)
sudo ufw allow 8443/tcp

# Enable forwarding (if needed)
echo "net.ipv4.ip_forward=1" | sudo tee -a /etc/sysctl.conf
sudo sysctl -p
```

## Troubleshooting

### SELinux (RHEL/Rocky Linux)

If you encounter permission issues on RHEL/Rocky Linux, you may need to configure SELinux:

```bash
# Check SELinux status
sestatus

# Allow network access for the service
sudo semanage port -a -t http_port_t -p tcp 8443

# Allow TUN device access
sudo semanage permissive -a sssonector_t
```

### Common Issues

1. TUN/TAP Device Issues:
```bash
# Check if TUN module is loaded
lsmod | grep tun

# Load TUN module if needed
sudo modprobe tun

# Make it persistent
echo "tun" | sudo tee -a /etc/modules
```

2. Permission Issues:
```bash
# Check file permissions
ls -l /etc/sssonector/certs/
ls -l /var/log/sssonector/

# Fix permissions if needed
sudo chown -R sssonector:sssonector /etc/sssonector/certs/
sudo chmod 600 /etc/sssonector/certs/*.key
sudo chmod 644 /etc/sssonector/certs/*.crt
```

3. Network Issues:
```bash
# Check if service is listening
sudo netstat -tulpn | grep sssonector

# Test connectivity
nc -zv server_ip 8443

# Check routing
ip route show
```

## Uninstallation

### Debian/Ubuntu
```bash
sudo apt-get remove sssonector
sudo apt-get purge sssonector  # Remove configuration files
```

### RHEL/Rocky Linux
```bash
sudo dnf remove sssonector
sudo rm -rf /etc/sssonector  # Remove configuration files
