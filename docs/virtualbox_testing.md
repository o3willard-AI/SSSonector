# VirtualBox Testing Guide for SSSonector

This guide explains how to set up and test SSSonector using two Ubuntu VMs in VirtualBox.

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
   - Adapter 1: NAT
   - Adapter 2: Host-only Adapter

4. Install Ubuntu Server:
   - Start the VM
   - Select Ubuntu Server ISO
   - Follow installation prompts
   - Install OpenSSH server when prompted

### Create Client VM

1. Repeat the same steps but name it "sssonector-client"
2. Use the same network configuration

## Building the Package

1. Clone the repository and build the Debian package:
```bash
git clone https://github.com/o3willard-AI/SSSonector.git
cd SSSonector
./scripts/build-deb.sh
```

The package will be created in `build/deb/sssonector_1.0.0_amd64.deb`

## Testing Setup

### Server VM Setup

1. Copy the Debian package to the server VM:
```bash
scp build/deb/sssonector_1.0.0_amd64.deb ubuntu@sssonector-server:~/
```

2. SSH into the server VM:
```bash
ssh ubuntu@sssonector-server
```

3. Install the package:
```bash
sudo apt update
sudo apt install -y ./sssonector_1.0.0_amd64.deb
```

4. Generate certificates:
```bash
sudo mkdir -p /etc/sssonector/certs
cd /etc/sssonector/certs
sudo openssl req -x509 -newkey rsa:4096 -keyout server.key -out server.crt -days 365 -nodes -subj "/CN=sssonector-server"
sudo openssl req -x509 -newkey rsa:4096 -keyout client.key -out client.crt -days 365 -nodes -subj "/CN=sssonector-client"
sudo chmod 600 *.key
sudo chmod 644 *.crt
```

5. Configure server:
```bash
sudo nano /etc/sssonector/config.yaml
```

Update the configuration:
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

### Client VM Setup

1. Copy the Debian package and certificates from server:
```bash
scp ubuntu@sssonector-server:~/sssonector_1.0.0_amd64.deb ./
scp -r ubuntu@sssonector-server:/etc/sssonector/certs ./certs
```

2. Install the package:
```bash
sudo apt update
sudo apt install -y ./sssonector_1.0.0_amd64.deb
```

3. Copy certificates:
```bash
sudo cp -r certs/* /etc/sssonector/certs/
sudo chmod 600 /etc/sssonector/certs/*.key
sudo chmod 644 /etc/sssonector/certs/*.crt
```

4. Configure client:
```bash
sudo nano /etc/sssonector/config.yaml
```

Update the configuration:
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
  server_address: "sssonector-server"  # Use the server VM's IP address
  server_port: 8443
```

## Running the Tests

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
   - Verify both VMs can ping each other
   - Check firewall rules: `sudo ufw status`
   - Ensure port 8443 is accessible

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
