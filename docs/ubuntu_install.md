# Ubuntu Installation and Testing Guide

This guide explains how to install and test SSSonector on Ubuntu VMs using VirtualBox.

## Prerequisites

1. VirtualBox installed on your host machine
2. Ubuntu Server 22.04 LTS ISO downloaded
3. At least 2GB RAM and 20GB storage available for each VM

## Setting Up VirtualBox VMs

### Create Server VM

1. Open VirtualBox and click "New"
2. Configure the VM:
   - Name: sssonector-server
   - Type: Linux
   - Version: Ubuntu (64-bit)
   - Memory: 2048 MB
   - Create a virtual hard disk (20 GB)

3. Configure Network:
   - Adapter 1: NAT (for internet access)
   - Adapter 2: Host-only Adapter (for VM-to-VM communication)

4. Install Ubuntu Server:
   - Start the VM
   - Select Ubuntu Server ISO
   - Follow installation prompts
   - Install OpenSSH server when prompted

### Create Client VM

1. Repeat the same steps but name it "sssonector-client"
2. Use the same network configuration

## Installation Steps

### On Both VMs

1. Update system packages:
```bash
sudo apt update
sudo apt upgrade -y
```

2. Install required dependencies:
```bash
sudo apt install -y openssl
```

3. Download and install the package:

⚠️ **Important:** The installer package is distributed through [GitHub Releases](https://github.com/o3willard-AI/SSSonector/releases/tag/v1.0.0). Always use the GitHub Releases URL for downloading.

```bash
# Download the package
wget https://github.com/o3willard-AI/SSSonector/releases/download/v1.0.0/sssonector_1.0.0_amd64.deb

# Verify the download URL worked (should show sssonector_1.0.0_amd64.deb)
ls sssonector_1.0.0_amd64.deb

# Install the package
```bash
sudo dpkg -i sssonector_1.0.0_amd64.deb
```

## Configuration

### Server VM

1. Generate certificates:
```bash
cd /etc/sssonector/certs
sudo openssl req -x509 -newkey rsa:4096 -keyout server.key -out server.crt -days 365 -nodes -subj "/CN=sssonector-server"
sudo openssl req -x509 -newkey rsa:4096 -keyout client.key -out client.crt -days 365 -nodes -subj "/CN=sssonector-client"
sudo chmod 600 *.key
sudo chmod 644 *.crt
```

2. Configure server:
```bash
sudo nano /etc/sssonector/config.yaml
```

Update with:
```yaml
mode: "server"
network:
  interface: "tun0"
  address: "10.0.0.1/24"
  mtu: 1500
tunnel:
  cert_file: "/etc/sssonector/certs/server.crt"
  key_file: "/etc/sssonector/certs/server.key"
  ca_file: "/etc/sssonector/certs/client.crt"
  listen_address: "0.0.0.0"
  listen_port: 8443
```

### Client VM

1. Copy certificates from server:
```bash
sudo mkdir -p /etc/sssonector/certs
# Replace SERVER_IP with the server's Host-only adapter IP
scp ubuntu@SERVER_IP:/etc/sssonector/certs/* /etc/sssonector/certs/
sudo chmod 600 /etc/sssonector/certs/*.key
sudo chmod 644 /etc/sssonector/certs/*.crt
```

2. Configure client:
```bash
sudo nano /etc/sssonector/config.yaml
```

Update with:
```yaml
mode: "client"
network:
  interface: "tun0"
  address: "10.0.0.2/24"
  mtu: 1500
tunnel:
  cert_file: "/etc/sssonector/certs/client.crt"
  key_file: "/etc/sssonector/certs/client.key"
  ca_file: "/etc/sssonector/certs/server.crt"
  server_address: "SERVER_IP"  # Replace with server's Host-only adapter IP
  server_port: 8443
```

## Testing

1. Start the server:
```bash
# On server VM
sudo systemctl start sssonector
sudo systemctl status sssonector
```

2. Start the client:
```bash
# On client VM
sudo systemctl start sssonector
sudo systemctl status sssonector
```

3. Test connectivity:
```bash
# On client VM
ping 10.0.0.1
```

4. Monitor logs:
```bash
# On both VMs
sudo journalctl -u sssonector -f
```

## Troubleshooting

1. Network Issues:
   - Verify both VMs can ping each other through Host-only network
   - Check firewall rules: `sudo ufw status`
   - Ensure port 8443 is accessible: `sudo netstat -tulpn | grep 8443`

2. Certificate Issues:
   - Verify certificate permissions
   - Check certificate paths in config
   - Validate certificate dates: `openssl x509 -in cert.crt -text`

3. Service Issues:
   - Check service status: `systemctl status sssonector`
   - View logs: `journalctl -u sssonector`
   - Verify config file permissions

## Cleanup

To remove the test environment:

1. Stop services:
```bash
# On both VMs
sudo systemctl stop sssonector
```

2. Uninstall package:
```bash
sudo apt remove sssonector
```

3. Remove configuration:
```bash
sudo rm -rf /etc/sssonector
sudo rm -rf /var/log/sssonector
