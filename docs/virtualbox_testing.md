# VirtualBox Testing Guide for SSSonector

This guide provides step-by-step instructions for setting up a test environment using VirtualBox to validate SSSonector functionality.

## Prerequisites

- VirtualBox 6.1 or later
- Ubuntu Server 22.04 LTS ISO
- At least 8GB RAM available for VMs
- 50GB free disk space

## Network Setup

1. Create Host-Only Network:
   - VirtualBox Manager → File → Host Network Manager
   - Create new network:
     ```
     Name: vboxnet0
     IPv4: 192.168.56.1
     Mask: 255.255.255.0
     DHCP: Disabled
     ```

## VM Configuration

### Server VM

1. Create new VM:
   ```
   Name: sssonector-server
   Type: Linux
   Version: Ubuntu (64-bit)
   Memory: 2048 MB
   CPU: 2 cores
   Disk: 20 GB (dynamically allocated)
   ```

2. Network Configuration:
   - Adapter 1: NAT (for internet access)
   - Adapter 2: Host-only (vboxnet0)

3. Install Ubuntu Server:
   - Language: English
   - Network: Use default settings
   - Storage: Use entire disk
   - Profile:
     ```
     Name: sssonector
     Server name: sssonector-server
     Username: sssonector
     Password: [secure-password]
     ```

4. Post-Installation:
   ```bash
   # Update system
   sudo apt update && sudo apt upgrade -y

   # Set static IP for host-only adapter
   sudo nano /etc/netplan/00-installer-config.yaml
   ```
   Add:
   ```yaml
   network:
     ethernets:
       enp0s3:
         dhcp4: true
       enp0s8:
         dhcp4: false
         addresses: [192.168.56.10/24]
     version: 2
   ```
   Apply:
   ```bash
   sudo netplan apply
   ```

### Client VM

1. Create new VM:
   ```
   Name: sssonector-client
   Type: Linux
   Version: Ubuntu (64-bit)
   Memory: 2048 MB
   CPU: 2 cores
   Disk: 20 GB (dynamically allocated)
   ```

2. Network Configuration:
   - Adapter 1: NAT (for internet access)
   - Adapter 2: Host-only (vboxnet0)

3. Install Ubuntu Server (same as server VM but with different hostname)
   ```
   Server name: sssonector-client
   ```

4. Post-Installation:
   ```bash
   # Update system
   sudo apt update && sudo apt upgrade -y

   # Set static IP for host-only adapter
   sudo nano /etc/netplan/00-installer-config.yaml
   ```
   Add:
   ```yaml
   network:
     ethernets:
       enp0s3:
         dhcp4: true
       enp0s8:
         dhcp4: false
         addresses: [192.168.56.11/24]
     version: 2
   ```
   Apply:
   ```bash
   sudo netplan apply
   ```

## Software Installation

### Server VM

1. Install SSSonector:
   ```bash
   # Download package
   wget https://github.com/o3willard-AI/SSSonector/dist/v1.0.0/sssonector_1.0.0_amd64.deb

   # Install
   sudo dpkg -i sssonector_1.0.0_amd64.deb
   sudo apt-get install -f
   ```

2. Generate certificates:
   ```bash
   sudo sssonector-cli generate-certs
   ```

3. Configure server:
   ```bash
   sudo nano /etc/sssonector/config.yaml
   ```
   Update:
   ```yaml
   mode: "server"
   network:
     interface: "tun0"
     address: "10.0.0.1/24"
   tunnel:
     listen_address: "192.168.56.10"
     listen_port: 8443
   ```

4. Start service:
   ```bash
   sudo systemctl start sssonector
   sudo systemctl enable sssonector
   ```

### Client VM

1. Install SSSonector:
   ```bash
   # Download package
   wget https://github.com/o3willard-AI/SSSonector/dist/v1.0.0/sssonector_1.0.0_amd64.deb

   # Install
   sudo dpkg -i sssonector_1.0.0_amd64.deb
   sudo apt-get install -f
   ```

2. Copy certificates from server:
   ```bash
   # On server
   sudo tar czf /tmp/certs.tar.gz -C /etc/sssonector/certs .
   scp /tmp/certs.tar.gz sssonector@192.168.56.11:/tmp/

   # On client
   sudo tar xzf /tmp/certs.tar.gz -C /etc/sssonector/certs/
   sudo chown -R root:root /etc/sssonector/certs
   sudo chmod 600 /etc/sssonector/certs/*.key
   sudo chmod 644 /etc/sssonector/certs/*.crt
   ```

3. Configure client:
   ```bash
   sudo nano /etc/sssonector/config.yaml
   ```
   Update:
   ```yaml
   mode: "client"
   network:
     interface: "tun0"
     address: "10.0.0.2/24"
   tunnel:
     server_address: "192.168.56.10"
     server_port: 8443
   ```

4. Start service:
   ```bash
   sudo systemctl start sssonector
   sudo systemctl enable sssonector
   ```

## Testing

### Basic Connectivity

1. From client VM:
   ```bash
   # Ping server's tunnel IP
   ping 10.0.0.1

   # Check connection details
   sssonector-cli status
   ```

### Performance Testing

1. Install iperf3:
   ```bash
   # On both VMs
   sudo apt install iperf3
   ```

2. Run tests:
   ```bash
   # On server
   iperf3 -s

   # On client
   iperf3 -c 10.0.0.1 -t 30
   ```

### Bandwidth Throttling

1. Configure throttling:
   ```bash
   # On both VMs
   sudo nano /etc/sssonector/config.yaml
   ```
   Add:
   ```yaml
   throttle:
     upload_kbps: 1024   # 1 Mbps
     download_kbps: 1024 # 1 Mbps
   ```

2. Restart services:
   ```bash
   sudo systemctl restart sssonector
   ```

3. Verify throttling:
   ```bash
   # On client
   iperf3 -c 10.0.0.1 -t 30
   ```

### Connection Resilience

1. Test automatic reconnection:
   ```bash
   # On server
   sudo systemctl restart sssonector

   # On client
   # Watch logs
   sudo journalctl -u sssonector -f
   ```

2. Test network interruption:
   ```bash
   # On server
   sudo ip link set enp0s8 down
   sleep 30
   sudo ip link set enp0s8 up
   ```

## Monitoring

### SNMP Monitoring

1. Install SNMP tools:
   ```bash
   sudo apt install snmp snmpd
   ```

2. Query metrics:
   ```bash
   # On either VM
   snmpwalk -v2c -c public localhost .1.3.6.1.4.1.XXXXX
   ```

### Log Analysis

1. View service logs:
   ```bash
   sudo journalctl -u sssonector -f
   ```

2. Check system logs:
   ```bash
   sudo tail -f /var/log/syslog | grep sssonector
   ```

## Cleanup

1. Stop services:
   ```bash
   # On both VMs
   sudo systemctl stop sssonector
   sudo systemctl disable sssonector
   ```

2. Remove packages:
   ```bash
   sudo apt remove sssonector
   sudo apt autoremove
   ```

3. Clean up files:
   ```bash
   sudo rm -rf /etc/sssonector
   sudo rm -rf /var/log/sssonector
   ```

## Troubleshooting

### Common Issues

1. Connection Failures
   - Check firewall status: `sudo ufw status`
   - Verify certificate permissions
   - Check network interfaces: `ip addr show`

2. Performance Issues
   - Check CPU usage: `top`
   - Monitor memory: `free -m`
   - Check disk I/O: `iostat`

3. Certificate Problems
   - Verify dates: `openssl x509 -in /etc/sssonector/certs/server.crt -text`
   - Check permissions: `ls -l /etc/sssonector/certs/`

### Debug Mode

Run service with debug logging:
```bash
sudo systemctl stop sssonector
sudo sssonector -config /etc/sssonector/config.yaml -debug
