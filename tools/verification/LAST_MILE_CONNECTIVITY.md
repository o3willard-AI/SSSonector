# SSSonector Last Mile Connectivity Guide

This guide provides information about last mile connectivity issues between SSSonector client and server modes after they are connected, and how to resolve them.

## Overview

SSSonector establishes a tunnel between two endpoints, allowing them to communicate securely over the public internet without a VPN. However, after the tunnel is established, there can be issues with the "last mile" connectivity, where packets are not properly transmitted between the tunnel endpoints.

## Symptoms

The following symptoms indicate a last mile connectivity issue:

1. SSSonector client and server successfully connect (tunnel interfaces are created)
2. Tunnel interfaces (tun0) are up on both client and server
3. Ping between tunnel endpoints (10.0.0.1 and 10.0.0.2) fails with 100% packet loss
4. TCP connections through the tunnel may also fail

## Investigation

We used the SSSonector investigation tools to systematically identify the cause of the last mile connectivity issue. The investigation included:

1. **Firewall Rules Investigation**: Checked and added firewall rules for ICMP traffic.
2. **Routing Tables Verification**: Verified and fixed routing tables.
3. **Kernel Parameters Investigation**: Checked and adjusted kernel parameters.
4. **Packet Capture Analysis**: Captured and analyzed packets to identify where packets are lost.
5. **MTU Investigation**: Tested with different MTU values and Path MTU Discovery settings.
6. **Packet Filtering Investigation**: Checked for packet filtering rules that might be blocking traffic.

## Root Causes

Based on our investigation, we identified several potential root causes for last mile connectivity issues:

### 1. Firewall Rules

The most common cause is missing INPUT rules for ICMP traffic or for the tun0 interface. While the FORWARD chain may have the necessary rules to forward traffic between interfaces, the INPUT chain may be blocking ICMP packets or traffic on the tun0 interface.

### 2. Kernel Parameters

Several kernel parameters can affect tunnel connectivity:

- **Reverse Path Filtering**: When set to strict mode (1), it can block packets from the tunnel interface.
- **ICMP Echo Ignore**: When enabled, it can block ICMP echo requests (ping).
- **IP Forwarding**: Must be enabled for packets to be forwarded between interfaces.

### 3. MTU Issues

If the MTU of the tunnel interface is too large, packets may be fragmented and lost. This is especially common when the tunnel passes through networks with smaller MTU values.

### 4. Routing Issues

Sometimes the routing tables may not have explicit routes for the tunnel endpoints, causing packets to be routed incorrectly.

## Solution

We've created a script that systematically addresses each potential root cause and tests connectivity after each fix. The script is located at:

```
/home/sblanken/Desktop/go/src/github.com/o3willard-AI/SSSonector/tools/verification/fix_last_mile_connectivity.sh
```

The script performs the following fixes in sequence:

### Fix 1: Add ICMP Rules to INPUT Chain

```bash
# On server
sudo iptables -A INPUT -p icmp -j ACCEPT
sudo iptables -A INPUT -i tun0 -j ACCEPT

# On client
sudo iptables -A INPUT -p icmp -j ACCEPT
sudo iptables -A INPUT -i tun0 -j ACCEPT
```

### Fix 2: Adjust Kernel Parameters

```bash
# On server and client
sudo sysctl -w net.ipv4.conf.all.rp_filter=0
sudo sysctl -w net.ipv4.conf.default.rp_filter=0
sudo sysctl -w net.ipv4.conf.tun0.rp_filter=0
sudo sysctl -w net.ipv4.icmp_echo_ignore_broadcasts=0
```

### Fix 3: Adjust MTU

```bash
# On server and client
sudo ip link set dev tun0 mtu 1400
```

### Fix 4: Add Explicit Routes

```bash
# On server
sudo ip route add 10.0.0.2/32 dev tun0

# On client
sudo ip route add 10.0.0.1/32 dev tun0
```

## Usage

To fix last mile connectivity issues, run the script:

```bash
cd /home/sblanken/Desktop/go/src/github.com/o3willard-AI/SSSonector/tools/verification
./fix_last_mile_connectivity.sh
```

The script will:

1. Test baseline connectivity
2. Apply each fix in sequence
3. Test connectivity after each fix
4. Report which fix resolved the issue

## Permanent Fixes

To make the fixes permanent, you should:

### 1. Add Firewall Rules to iptables Configuration

Edit `/etc/iptables/rules.v4` or use `iptables-save` to save the rules:

```bash
sudo iptables-save > /etc/iptables/rules.v4
```

### 2. Add Kernel Parameters to sysctl Configuration

Edit `/etc/sysctl.conf` and add:

```
net.ipv4.conf.all.rp_filter=0
net.ipv4.conf.default.rp_filter=0
net.ipv4.conf.tun0.rp_filter=0
net.ipv4.icmp_echo_ignore_broadcasts=0
```

Then apply the changes:

```bash
sudo sysctl -p
```

### 3. Add MTU Setting to Network Configuration

Edit the network configuration file for your distribution to set the MTU for the tun0 interface.

### 4. Add Routes to Network Configuration

Edit the network configuration file for your distribution to add the explicit routes.

## Conclusion

Last mile connectivity issues in SSSonector are often caused by firewall rules, kernel parameters, MTU settings, or routing issues. By systematically addressing each potential cause, you can resolve these issues and ensure reliable connectivity through the tunnel.

The `fix_last_mile_connectivity.sh` script provides a convenient way to identify and fix these issues, and the permanent fixes ensure that the fixes persist across reboots.
