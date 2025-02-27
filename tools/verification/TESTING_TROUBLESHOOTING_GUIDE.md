# SSSonector Testing Troubleshooting Guide

This guide provides solutions to common issues encountered during SSSonector testing and certification.

## Common Testing Issues

### Tunnel Establishment Failures

If the tunnel fails to establish during testing:

1. **Check IP Forwarding**
   ```bash
   # Verify IP forwarding is enabled
   sysctl net.ipv4.ip_forward
   
   # Enable IP forwarding if needed
   sudo sysctl -w net.ipv4.ip_forward=1
   ```

2. **Verify Firewall Rules**
   ```bash
   # Check for blocking rules
   sudo iptables -L -n
   
   # Allow traffic on port 8443
   sudo iptables -A INPUT -p tcp --dport 8443 -j ACCEPT
   sudo iptables -A OUTPUT -p tcp --sport 8443 -j ACCEPT
   ```

3. **Check TUN Module**
   ```bash
   # Verify TUN module is loaded
   lsmod | grep tun
   
   # Load TUN module if needed
   sudo modprobe tun
   ```

### Packet Transmission Issues

If packet transmission fails during testing:

1. **Check Routing Tables**
   ```bash
   # Verify routes
   ip route show
   
   # Add route if needed
   sudo ip route add 10.0.0.0/24 dev tun0
   ```

2. **Verify MTU Settings**
   ```bash
   # Check MTU
   ip link show tun0
   
   # Set MTU if needed
   sudo ip link set tun0 mtu 1500
   ```

3. **Test Basic Connectivity**
   ```bash
   # Test ping through tunnel
   ping -c 3 10.0.0.1  # From client
   ping -c 3 10.0.0.2  # From server
   ```

### Certificate Issues

If certificate validation fails:

1. **Check Certificate Permissions**
   ```bash
   # Verify permissions
   ls -la /opt/sssonector/certs/
   
   # Fix permissions if needed
   sudo chmod 600 /opt/sssonector/certs/*.key
   sudo chmod 644 /opt/sssonector/certs/*.crt
   ```

2. **Verify Certificate Validity**
   ```bash
   # Check certificate
   openssl x509 -in /opt/sssonector/certs/server.crt -text -noout
   ```

3. **Regenerate Certificates**
   ```bash
   # Clean up existing certificates
   sudo rm -rf /opt/sssonector/certs/*
   
   # Deploy SSSonector again to generate new certificates
   ./deploy_sssonector.sh
   ```

## Debugging Techniques

### Packet Capture

To capture and analyze packets:

```bash
# Capture packets on TUN interface
sudo tcpdump -i tun0 -n -w /tmp/tunnel.pcap

# Analyze captured packets
sudo tcpdump -r /tmp/tunnel.pcap -n
```

### Log Analysis

To analyze SSSonector logs:

```bash
# View server logs
sudo tail -f /opt/sssonector/log/server.log

# View client logs
sudo tail -f /opt/sssonector/log/client.log

# Search for specific errors
grep "ERROR" /opt/sssonector/log/server.log
```

### Process Monitoring

To monitor SSSonector processes:

```bash
# Check running processes
ps aux | grep sssonector

# Monitor resource usage
top -p $(pgrep -d',' -f sssonector)
```

## Certification Troubleshooting

If certification fails:

1. **Verify Test Environment**
   - Ensure QA servers are accessible
   - Verify network connectivity between servers
   - Check system resources (CPU, memory, disk space)

2. **Check Test Parameters**
   - Verify packet sizes are appropriate
   - Ensure timing thresholds are realistic
   - Check for conflicting network traffic

3. **Review Test Reports**
   - Analyze timing measurements
   - Check packet transmission statistics
   - Verify bandwidth metrics

## Performance Optimization

To optimize SSSonector performance during testing:

1. **Adjust Buffer Sizes**
   - Increase buffer sizes for higher throughput
   - Decrease buffer sizes for lower latency

2. **Tune Network Parameters**
   ```bash
   # Increase TCP buffer sizes
   sudo sysctl -w net.core.rmem_max=16777216
   sudo sysctl -w net.core.wmem_max=16777216
   
   # Adjust TCP congestion control
   sudo sysctl -w net.ipv4.tcp_congestion_control=bbr
   ```

3. **Optimize System Resources**
   - Close unnecessary applications
   - Prioritize SSSonector processes
   - Disable power management features

## Conclusion

This troubleshooting guide provides solutions to common issues encountered during SSSonector testing and certification. By following these steps, you can identify and resolve issues quickly, ensuring successful testing and certification of SSSonector.

For more information, refer to the [QA Methodology 2025](QA_METHODOLOGY_2025.md) document and [Minimal Functionality Test](MINIMAL_FUNCTIONALITY_TEST.md) documentation.
