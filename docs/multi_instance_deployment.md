# Multi-Instance Deployment Guide

This guide explains how to deploy multiple SSSonector instances on a single server to support multiple clients.

## Overview

Each SSSonector instance provides a point-to-point tunnel between a server and one client. To support multiple clients, run multiple server instances - one per client connection.

```
┌─────────────────────────────────────────────────────────────────┐
│                        SERVER (192.168.1.10)                     │
│                                                                  │
│  ┌──────────────┐  ┌──────────────┐  ┌──────────────┐           │
│  │  Instance    │  │  Instance    │  │  Instance    │           │
│  │  client-a    │  │  client-b    │  │  client-c    │           │
│  │  tun1        │  │  tun2        │  │  tun3        │           │
│  │  10.0.1.1    │  │  10.0.2.1    │  │  10.0.3.1    │           │
│  │  port 8443   │  │  port 8444   │  │  port 8445   │           │
│  └──────┬───────┘  └──────┬───────┘  └──────┬───────┘           │
└─────────┼────────────────┼────────────────┼────────────────────┘
          │                │                │
          │ TLS            │ TLS            │ TLS
          ▼                ▼                ▼
     ┌─────────┐      ┌─────────┐      ┌─────────┐
     │Client A │      │Client B │      │Client C │
     │10.0.1.2 │      │10.0.2.2 │      │10.0.3.2 │
     └─────────┘      └─────────┘      └─────────┘
```

## Quick Start

### Option 1: Using the Install Script (Recommended)

The install script can create additional instances on subsequent runs:

```bash
# Create first server instance
curl -fsSL https://raw.githubusercontent.com/o3willard-AI/SSSonector/main/install.sh | \
  sudo SSSONECTOR_MODE=server \
       SSSONECTOR_INSTANCE=client-a \
       SSSONECTOR_ADDRESS=10.0.1.1/24 \
       SSSONECTOR_PORT=8443 \
       bash

# Create second server instance
curl -fsSL https://raw.githubusercontent.com/o3willard-AI/SSSonector/main/install.sh | \
  sudo SSSONECTOR_MODE=server \
       SSSONECTOR_INSTANCE=client-b \
       SSSONECTOR_ADDRESS=10.0.2.1/24 \
       SSSONECTOR_PORT=8444 \
       bash

# Create client instance (on client machine)
curl -fsSL https://raw.githubusercontent.com/o3willard-AI/SSSonector/main/install.sh | \
  sudo SSSONECTOR_MODE=client \
       SSSONECTOR_INSTANCE=client-a \
       SSSONECTOR_ADDRESS=10.0.1.2/24 \
       SSSONECTOR_SERVER=192.168.1.10 \
       SSSONECTOR_PORT=8443 \
       bash
```

### Option 2: Using the Instance Manager Script

For manual instance management, use the instance-manager.sh script:

```bash
# Copy the instance manager script
sudo cp scripts/instance-manager.sh /usr/local/bin/sssonector-instance
sudo chmod +x /usr/local/bin/sssonector-instance

# Install systemd template
sudo cp deploy/systemd/sssonector@.service /etc/systemd/system/
sudo systemctl daemon-reload
```

### Create Server Instances

```bash
# Create instance for client-a
sudo sssonector-instance create-server \
  -n client-a \
  -a 10.0.1.1/24 \
  -p 8443

# Create instance for client-b
sudo sssonector-instance create-server \
  -n client-b \
  -a 10.0.2.1/24 \
  -p 8444

# Create instance for client-c
sudo sssonector-instance create-server \
  -n client-c \
  -a 10.0.3.1/24 \
  -p 8445
```

### Create Client Instances

On each client machine:

```bash
# On client-a machine
sudo sssonector-instance create-client \
  -n client-a \
  -a 10.0.1.2/24 \
  -s 192.168.1.10 \
  -p 8443

# On client-b machine
sudo sssonector-instance create-client \
  -n client-b \
  -a 10.0.2.2/24 \
  -s 192.168.1.10 \
  -p 8444

# On client-c machine
sudo sssonector-instance create-client \
  -n client-c \
  -a 10.0.3.2/24 \
  -s 192.168.1.10 \
  -p 8445
```

### Copy Certificates

After creating instances, copy the client certificates to each client:

```bash
# On server - export certificates for client-a
sudo tar czf /tmp/client-a-certs.tar.gz -C /etc/sssonector/instances/client-a/certs .

# Transfer to client-a
scp /tmp/client-a-certs.tar.gz client-a:/tmp/

# On client-a - extract certificates
sudo tar xzf /tmp/client-a-certs.tar.gz -C /etc/sssonector/instances/client-a/certs
```

### Start Services

```bash
# Start instances
sudo systemctl start sssonector@client-a
sudo systemctl start sssonector@client-b
sudo systemctl start sssonector@client-c

# Enable for auto-start
sudo systemctl enable sssonector@client-a
sudo systemctl enable sssonector@client-b
sudo systemctl enable sssonector@client-c
```

### Verify

```bash
# Check status
sudo systemctl status sssonector@client-a

# View logs
journalctl -u sssonector@client-a -f

# List all instances
sudo sssonector-instance list
```

## Instance Manager Commands

| Command | Description |
|---------|-------------|
| `create-server` | Create a server instance |
| `create-client` | Create a client instance |
| `list` | List all instances |
| `remove` | Remove an instance |
| `generate-certs` | Regenerate certificates for an instance |

### Options

| Option | Description | Default |
|--------|-------------|---------|
| `-n, --name` | Instance name (required) | - |
| `-t, --tun` | TUN interface name | Auto (tun1, tun2, ...) |
| `-a, --address` | TUN IP address with CIDR (required) | - |
| `-p, --port` | Listen/Server port | Auto (8443, 8444, ...) |
| `-s, --server` | Server address (client only) | - |
| `-r, --rate` | Rate limit (bytes/sec) | 1048576 (1 MB/s) |
| `-b, --burst` | Burst limit (bytes/sec) | 2097152 (2 MB/s) |
| `--prometheus-port` | Prometheus metrics port | Auto (9090, 9091, ...) |

## Directory Structure

```
/etc/sssonector/
├── instances/
│   ├── client-a/
│   │   ├── config.yaml
│   │   └── certs/
│   │       ├── ca.crt
│   │       ├── ca.key
│   │       ├── server.crt
│   │       ├── server.key
│   │       ├── client.crt
│   │       └── client.key
│   ├── client-b/
│   │   ├── config.yaml
│   │   └── certs/
│   └── client-c/
│       ├── config.yaml
│       └── certs/
└── templates/
    ├── server.yaml.template
    └── client.yaml.template
```

## Manual Configuration

If you prefer to create configurations manually:

### Server Instance

```yaml
# /etc/sssonector/instances/my-client/config.yaml
type: server
config:
  mode: server
  network:
    name: tun1
    address: 10.0.1.1/24
    mtu: 1500
  tunnel:
    listen_address: 0.0.0.0
    listen_port: 8443
  auth:
    cert_file: /etc/sssonector/instances/my-client/certs/server.crt
    key_file: /etc/sssonector/instances/my-client/certs/server.key
    ca_file: /etc/sssonector/instances/my-client/certs/ca.crt
```

### Client Instance

```yaml
# /etc/sssonector/instances/my-client/config.yaml
type: client
config:
  mode: client
  network:
    name: tun1
    address: 10.0.1.2/24
    mtu: 1500
  tunnel:
    server_address: 192.168.1.10
    server_port: 8443
  auth:
    cert_file: /etc/sssonector/instances/my-client/certs/client.crt
    key_file: /etc/sssonector/instances/my-client/certs/client.key
    ca_file: /etc/sssonector/instances/my-client/certs/ca.crt
```

## Firewall Configuration

Open the required ports on the server:

```bash
# Using ufw
sudo ufw allow 8443/tcp  # client-a
sudo ufw allow 8444/tcp  # client-b
sudo ufw allow 8445/tcp  # client-c

# Or using iptables
sudo iptables -A INPUT -p tcp --dport 8443 -j ACCEPT
sudo iptables -A INPUT -p tcp --dport 8444 -j ACCEPT
sudo iptables -A INPUT -p tcp --dport 8445 -j ACCEPT
```

## Monitoring

Each instance exposes Prometheus metrics on its assigned port:

```bash
# Scrape all instances
curl http://localhost:9090/metrics  # client-a
curl http://localhost:9091/metrics  # client-b
curl http://localhost:9092/metrics  # client-c
```

Prometheus configuration:

```yaml
scrape_configs:
  - job_name: 'sssonector'
    static_configs:
      - targets:
          - 'localhost:9090'
          - 'localhost:9091'
          - 'localhost:9092'
        labels:
          instance: client-a
    relabel_configs:
      - source_labels: [__address__]
        target_label: instance_name
        regex: 'localhost:(\d+)'
        replacement: 'instance-${1}'
```

## Troubleshooting

### Instance won't start

```bash
# Check logs
journalctl -u sssonector@client-a -n 50

# Verify config
sssonector -config /etc/sssonector/instances/client-a/config.yaml -validate

# Check if port is in use
sudo netstat -tlnp | grep 8443

# Check if TUN interface exists
ip link show tun1
```

### Connection refused

```bash
# Verify server is listening
sudo netstat -tlnp | grep sssonector

# Check firewall
sudo ufw status

# Test connectivity
telnet 192.168.1.10 8443
```

### Certificate errors

```bash
# Verify certificate files exist
ls -la /etc/sssonector/instances/client-a/certs/

# Check certificate validity
openssl x509 -in /etc/sssonector/instances/client-a/certs/server.crt -text -noout

# Verify certificate chain
openssl verify -CAfile /etc/sssonector/instances/client-a/certs/ca.crt \
  /etc/sssonector/instances/client-a/certs/server.crt
```

## Resource Considerations

| Metric | Per Instance | 10 Instances | 50 Instances |
|--------|--------------|--------------|--------------|
| Memory | ~15 MB | ~150 MB | ~750 MB |
| CPU (idle) | < 1% | < 5% | < 20% |
| File descriptors | ~20 | ~200 | ~1000 |
| TUN interfaces | 1 | 10 | 50 |

For large deployments (50+ instances), consider:
- Increasing system file descriptor limits
- Using a dedicated tunnel server
- Monitoring memory usage

## Upgrading Multi-Instance Deployments

The upgrade script automatically handles multiple instances:

```bash
# Upgrade all instances (stops, upgrades, restarts)
curl -fsSL https://raw.githubusercontent.com/o3willard-AI/SSSonector/main/upgrade.sh | sudo bash

# Upgrade without restarting (manual restart later)
curl -fsSL https://raw.githubusercontent.com/o3willard-AI/SSSonector/main/upgrade.sh | \
  sudo bash -s -- --no-restart

# Manual restart after upgrade
sudo systemctl start sssonector@client-a
sudo systemctl start sssonector@client-b
sudo systemctl start sssonector@client-c

# Or restart all at once
sudo systemctl start 'sssonector@*'
```

The upgrade process:
1. Detects all running instances
2. Downloads new binary
3. Stops each running instance
4. Replaces binary
5. Restarts previously running instances
