# SSSonector Prerequisites

This document outlines the system prerequisites required for SSSonector to operate correctly.

## System Requirements

- Linux system with TUN module support
- OpenSSL for certificate operations
- Root/sudo access for network operations

## Network Requirements

### IP Forwarding

IP forwarding is **required** for SSSonector to function properly. This allows packets to be forwarded between network interfaces, which is essential for tunnel operation.

#### Checking IP Forwarding Status

```bash
cat /proc/sys/net/ipv4/ip_forward
```

If the output is `1`, IP forwarding is enabled. If it's `0`, IP forwarding is disabled.

#### Enabling IP Forwarding Temporarily

```bash
echo 1 | sudo tee /proc/sys/net/ipv4/ip_forward
```

#### Enabling IP Forwarding Permanently

```bash
echo "net.ipv4.ip_forward = 1" | sudo tee -a /etc/sysctl.conf
sudo sysctl -p
```

### Firewall Rules

Proper firewall rules are necessary to allow traffic to flow between the tunnel interface and the physical network interface. The following iptables rules are recommended:

#### Allow Forwarding Between Interfaces

```bash
# Replace eth0 with your main network interface
sudo iptables -A FORWARD -i tun0 -o eth0 -j ACCEPT
sudo iptables -A FORWARD -i eth0 -o tun0 -j ACCEPT
```

#### Enable NAT for Outgoing Connections

```bash
# Replace eth0 with your main network interface
sudo iptables -t nat -A POSTROUTING -o eth0 -j MASQUERADE
```

#### Making Firewall Rules Persistent

```bash
sudo sh -c 'iptables-save > /etc/iptables.rules'
```

To load these rules at boot time, add the following to `/etc/rc.local` or create a systemd service:

```bash
iptables-restore < /etc/iptables.rules
```

## Port Requirements

SSSonector requires the following ports to be available:

- **TCP 443**: Default HTTPS port for secure communication
- **TCP 8443**: Alternative port for SSSonector server
- **TCP 9090-9093**: Monitoring ports (if monitoring is enabled)

## TUN Interface

SSSonector creates and uses a TUN interface (typically named `tun0`). Ensure that:

1. The TUN kernel module is loaded
2. The user running SSSonector has permission to create TUN interfaces

To check if the TUN module is loaded:

```bash
lsmod | grep tun
```

To load the TUN module:

```bash
sudo modprobe tun
```

## Verification

You can verify that your system meets all prerequisites by running the verification tool:

```bash
sudo verify-environment --modules system,network
```

This will check all system and network requirements and report any issues.

## Troubleshooting

If you encounter issues with SSSonector, check the following:

1. Ensure IP forwarding is enabled
2. Verify that the necessary firewall rules are in place
3. Check that the TUN module is loaded
4. Ensure the user has sufficient permissions
5. Verify that required ports are available

For detailed troubleshooting, refer to the [Troubleshooting Guide](TROUBLESHOOTING.md).
