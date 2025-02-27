# SSSonector Troubleshooting Guide

This guide provides solutions to common issues encountered when using SSSonector.

## Connection Issues

### Server and Client Cannot Connect

**Symptoms**:
- Client cannot connect to server
- Error messages about connection failures

**Possible Causes**:
1. Firewall blocking the connection
2. Incorrect server address or port
3. Server not running
4. Certificate issues

**Solutions**:
1. Check that the server is running: `ps aux | grep sssonector`
2. Verify the server address and port in the client configuration
3. Check firewall rules to ensure the port is open: `sudo iptables -L -n`
4. Verify certificates are valid and in the correct location
5. Enable debug logging for more detailed error messages: `logging.level: debug`

### Tunnel Established but No Traffic Flows

**Symptoms**:
- Client connects to server successfully
- Tunnel interfaces are created
- No traffic flows through the tunnel

**Possible Causes**:
1. IP forwarding not enabled
2. Firewall blocking tunnel traffic
3. Routing issues
4. MTU issues

**Solutions**:
1. Check that IP forwarding is enabled: `cat /proc/sys/net/ipv4/ip_forward`
2. Enable IP forwarding if needed: `echo 1 | sudo tee /proc/sys/net/ipv4/ip_forward`
3. Check firewall rules for the TUN interface: `sudo iptables -L -n`
4. Add necessary firewall rules:
   ```bash
   sudo iptables -A FORWARD -i tun0 -o eth0 -j ACCEPT
   sudo iptables -A FORWARD -i eth0 -o tun0 -j ACCEPT
   sudo iptables -t nat -A POSTROUTING -o eth0 -j MASQUERADE
   ```
5. Check routing tables: `ip route show`
6. Try adjusting the MTU: `network.mtu: 1400`

### One-Way Communication

**Symptoms**:
- Server can ping client but client cannot ping server
- Or client can ping server but server cannot ping client

**Possible Causes**:
1. Asymmetric routing
2. Firewall rules blocking traffic in one direction
3. NAT issues

**Solutions**:
1. Check firewall rules on both server and client
2. Verify NAT configuration
3. Check routing tables on both server and client
4. Use packet capture to identify where packets are being dropped:
   ```bash
   sudo tcpdump -i tun0 -n
   ```

## Certificate Issues

### Certificate Validation Failures

**Symptoms**:
- Client cannot connect to server
- Error messages about certificate validation failures

**Possible Causes**:
1. Certificates not generated correctly
2. Certificates expired
3. Incorrect certificate paths
4. CA certificate not trusted

**Solutions**:
1. Regenerate certificates: `./sssonector --generate-certs --cert-dir /path/to/certs --server-ip <server_ip>`
2. Verify certificate validity: `openssl x509 -in certs/server.crt -text -noout`
3. Check certificate paths in configuration files
4. Ensure CA certificate is trusted by both server and client

### Certificate Generation Failures

**Symptoms**:
- Certificate generation fails
- Error messages about OpenSSL

**Possible Causes**:
1. OpenSSL not installed
2. Insufficient permissions
3. Invalid server IP

**Solutions**:
1. Install OpenSSL: `sudo apt-get install openssl`
2. Run with sudo or as root: `sudo ./sssonector --generate-certs --cert-dir /path/to/certs --server-ip <server_ip>`
3. Verify server IP is valid

## Performance Issues

### High Latency

**Symptoms**:
- Tunnel works but with high latency
- Slow response times for applications using the tunnel

**Possible Causes**:
1. Network congestion
2. MTU issues
3. CPU or memory constraints
4. Logging overhead

**Solutions**:
1. Check network conditions: `ping <server_ip>`
2. Adjust MTU: `network.mtu: 1400`
3. Monitor system resources: `top`
4. Reduce logging level: `logging.level: warning`

### Low Throughput

**Symptoms**:
- Tunnel works but with low throughput
- Slow file transfers or streaming

**Possible Causes**:
1. Network bandwidth limitations
2. MTU issues
3. CPU or memory constraints
4. TLS overhead

**Solutions**:
1. Check network bandwidth: `iperf -c <server_ip>`
2. Adjust MTU: `network.mtu: 1500`
3. Monitor system resources: `top`
4. Optimize TLS settings: `security.tls.min_version: "1.3"`

### Packet Loss

**Symptoms**:
- Intermittent connection issues
- Retransmissions
- Timeouts

**Possible Causes**:
1. Network congestion
2. MTU issues
3. Buffer overflows

**Solutions**:
1. Check network conditions: `ping <server_ip>`
2. Adjust MTU: `network.mtu: 1400`
3. Monitor packet loss: `ping -c 100 <server_ip> | grep loss`
4. Use packet capture to identify where packets are being dropped:
   ```bash
   sudo tcpdump -i tun0 -n
   ```

## System Issues

### Permission Denied

**Symptoms**:
- Error messages about permission denied
- Cannot create TUN interface

**Possible Causes**:
1. Not running as root or with sufficient privileges
2. TUN module not loaded
3. File permission issues

**Solutions**:
1. Run with sudo or as root: `sudo ./sssonector -config config.yaml`
2. Load TUN module: `sudo modprobe tun`
3. Check file permissions: `ls -la certs/`
4. Fix file permissions: `sudo chmod 600 certs/*.key && sudo chmod 644 certs/*.crt`

### Resource Exhaustion

**Symptoms**:
- SSSonector crashes or stops working
- Error messages about resource limits

**Possible Causes**:
1. Too many open files
2. Memory exhaustion
3. CPU overload

**Solutions**:
1. Increase file descriptor limits: `ulimit -n 65536`
2. Monitor memory usage: `free -m`
3. Monitor CPU usage: `top`
4. Restart SSSonector if necessary

### TUN Interface Issues

**Symptoms**:
- Cannot create TUN interface
- Error messages about TUN interface

**Possible Causes**:
1. TUN module not loaded
2. Insufficient permissions
3. Interface name conflict

**Solutions**:
1. Load TUN module: `sudo modprobe tun`
2. Run with sudo or as root: `sudo ./sssonector -config config.yaml`
3. Check if interface already exists: `ip link show`
4. Use a different interface name: `interface: sssonector0`

## Configuration Issues

### Invalid Configuration

**Symptoms**:
- SSSonector fails to start
- Error messages about configuration

**Possible Causes**:
1. Syntax errors in configuration file
2. Missing required fields
3. Invalid values

**Solutions**:
1. Check configuration file syntax
2. Verify all required fields are present
3. Ensure values are valid
4. Enable debug logging for more detailed error messages: `logging.level: debug`

### Configuration File Not Found

**Symptoms**:
- SSSonector fails to start
- Error messages about configuration file not found

**Possible Causes**:
1. Configuration file not in expected location
2. Incorrect path specified

**Solutions**:
1. Verify configuration file exists
2. Specify full path to configuration file: `./sssonector -config /path/to/config.yaml`
3. Place configuration file in one of the default locations:
   - `./sssonector.yaml` in the current directory
   - `./sssonector.yml` in the current directory
   - `/etc/sssonector/sssonector.yaml`
   - `/etc/sssonector/sssonector.yml`

## Logging and Debugging

### Enabling Debug Logging

To get more detailed information about what's happening, enable debug logging:

```yaml
logging:
  level: debug
  file: /var/log/sssonector.log
```

Or use the environment variable:

```bash
export SSSONECTOR_LOGGING_LEVEL=debug
```

### Common Debug Log Messages

- **"Starting SSSonector in [server/client] mode"**: Indicates SSSonector is starting up
- **"Listening on [address:port]"**: Server is listening for connections
- **"Connecting to [address:port]"**: Client is connecting to server
- **"TLS handshake successful"**: TLS connection established
- **"Created TUN interface [name]"**: TUN interface created
- **"Starting transfer"**: Packet forwarding started
- **"Transfer complete"**: Packet forwarding stopped
- **"Error: [message]"**: Error occurred

### Packet Capture

To capture and analyze packets flowing through the tunnel:

```bash
sudo tcpdump -i tun0 -n -w /tmp/tunnel.pcap
```

To analyze the captured packets:

```bash
sudo tcpdump -r /tmp/tunnel.pcap -n
```

### Process Monitoring

To monitor SSSonector processes:

```bash
ps aux | grep sssonector
```

To monitor resource usage:

```bash
top -p $(pgrep -d',' -f sssonector)
```

## Common Error Messages

### "Failed to create TUN interface"

**Possible Causes**:
1. TUN module not loaded
2. Insufficient permissions
3. Interface name conflict

**Solutions**:
1. Load TUN module: `sudo modprobe tun`
2. Run with sudo or as root: `sudo ./sssonector -config config.yaml`
3. Use a different interface name: `interface: sssonector0`

### "Failed to bind to [address:port]"

**Possible Causes**:
1. Port already in use
2. Insufficient permissions
3. Invalid address

**Solutions**:
1. Check if port is already in use: `sudo netstat -tulpn | grep <port>`
2. Use a different port
3. Run with sudo or as root: `sudo ./sssonector -config config.yaml`

### "Certificate verification failed"

**Possible Causes**:
1. Certificates not generated correctly
2. Certificates expired
3. Incorrect certificate paths
4. CA certificate not trusted

**Solutions**:
1. Regenerate certificates: `./sssonector --generate-certs --cert-dir /path/to/certs --server-ip <server_ip>`
2. Verify certificate validity: `openssl x509 -in certs/server.crt -text -noout`
3. Check certificate paths in configuration files
4. Ensure CA certificate is trusted by both server and client

### "Failed to establish tunnel"

**Possible Causes**:
1. Network connectivity issues
2. Firewall blocking connection
3. Certificate issues
4. Server not running

**Solutions**:
1. Check network connectivity: `ping <server_ip>`
2. Check firewall rules: `sudo iptables -L -n`
3. Verify certificates
4. Ensure server is running

## Advanced Troubleshooting

### Kernel Parameters

If you're experiencing performance issues or connectivity problems, check and adjust kernel parameters:

```bash
# Check current parameters
sysctl -a | grep net.ipv4.tcp

# Adjust parameters for better performance
sudo sysctl -w net.ipv4.tcp_rmem="4096 87380 16777216"
sudo sysctl -w net.ipv4.tcp_wmem="4096 65536 16777216"
sudo sysctl -w net.ipv4.tcp_congestion_control=bbr
```

### Firewall Configuration

For detailed firewall configuration:

```bash
# Allow SSSonector server port
sudo iptables -A INPUT -p tcp --dport 443 -j ACCEPT
sudo iptables -A OUTPUT -p tcp --sport 443 -j ACCEPT

# Allow forwarding between interfaces
sudo iptables -A FORWARD -i tun0 -o eth0 -j ACCEPT
sudo iptables -A FORWARD -i eth0 -o tun0 -j ACCEPT

# Enable NAT for outgoing connections
sudo iptables -t nat -A POSTROUTING -o eth0 -j MASQUERADE
```

### Routing Tables

For complex routing scenarios:

```bash
# Add route for tunnel network
sudo ip route add 10.0.0.0/24 dev tun0

# Add route for specific destination through tunnel
sudo ip route add 192.168.1.0/24 via 10.0.0.1
```

## Getting Help

If you're still experiencing issues after trying the solutions in this guide, check the following resources:

1. SSSonector documentation
2. GitHub issues
3. Community forums
4. Contact support

## Reporting Issues

When reporting issues, include the following information:

1. SSSonector version: `./sssonector --version`
2. Operating system and version
3. Configuration file (with sensitive information redacted)
4. Debug logs
5. Steps to reproduce the issue
6. Expected behavior
7. Actual behavior
