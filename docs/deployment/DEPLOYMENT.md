# SSSonector Deployment Guide

## Overview

This guide covers the deployment of SSSonector in various environments, including bare metal, containerized, and cloud deployments.

## Prerequisites

- Go 1.21 or later
- Docker 24.0 or later (for containerized deployment)
- Docker Compose 2.0 or later (for containerized deployment)
- systemd (for Linux service deployment)
- OpenSSL 3.0 or later (for certificate management)

## Installation Methods

### 1. Docker Deployment (Recommended)

```bash
# Clone repository
git clone https://github.com/o3willard-AI/SSSonector.git
cd SSSonector

# Build and start services
docker-compose up -d

# Check service status
docker-compose ps
```

Configuration:
1. Edit `config/config.json` for service settings
2. Modify `docker-compose.yml` for container settings
3. Update `monitoring/prometheus/prometheus.yml` for metrics
4. Adjust `monitoring/grafana/dashboards/sssonector.json` for visualization

### 2. Bare Metal Installation

#### Linux (systemd)

```bash
# Install from source
make build

# Install service
sudo make install

# Start service
sudo systemctl enable sssonector
sudo systemctl start sssonector

# Check status
sudo systemctl status sssonector
```

#### macOS (launchd)

```bash
# Install from source
make build

# Install service
sudo make install-macos

# Start service
sudo launchctl load /Library/LaunchDaemons/com.o3willard.sssonector.plist

# Check status
sudo launchctl list | grep sssonector
```

#### Windows (Service)

```powershell
# Install from source
make build

# Install service
.\scripts\install.ps1

# Start service
Start-Service SSonector

# Check status
Get-Service SSonector
```

### 3. Cloud Deployment

#### Kubernetes

1. Apply configuration:
```bash
kubectl apply -f deploy/kubernetes/
```

2. Verify deployment:
```bash
kubectl get pods -l app=sssonector
```

3. Check logs:
```bash
kubectl logs -l app=sssonector
```

## Configuration

### 1. Basic Configuration

```json
{
  "mode": "server",
  "network": {
    "interface": "eth0",
    "mtu": 1500
  },
  "tunnel": {
    "protocol": "tcp",
    "encryption": "aes-256-gcm",
    "compression": "none"
  }
}
```

### 2. Security Configuration

```json
{
  "security": {
    "tls": {
      "cert_file": "/etc/sssonector/certs/server.crt",
      "key_file": "/etc/sssonector/certs/server.key",
      "ca_file": "/etc/sssonector/certs/ca.crt"
    },
    "auth_method": "certificate"
  }
}
```

### 3. Monitoring Configuration

```json
{
  "monitor": {
    "enabled": true,
    "interval": 60,
    "log_level": "info",
    "metrics_enabled": true
  }
}
```

## Security Considerations

### 1. Certificate Management

```bash
# Generate CA
openssl req -x509 -new -nodes -key ca.key -sha256 -days 1825 -out ca.crt

# Generate server certificate
openssl req -new -nodes -key server.key -out server.csr
openssl x509 -req -in server.csr -CA ca.crt -CAkey ca.key -CAcreateserial -out server.crt
```

### 2. Firewall Configuration

```bash
# Allow SSSonector ports
sudo ufw allow 8080/tcp  # HTTP API
sudo ufw allow 8443/tcp  # HTTPS API
```

### 3. SELinux Configuration

```bash
# Build and install policy
cd security/selinux
make -f /usr/share/selinux/devel/Makefile
semodule -i sssonector.pp
```

### 4. AppArmor Configuration

```bash
# Install profile
sudo cp security/apparmor/usr.local.bin.sssonector /etc/apparmor.d/
sudo apparmor_parser -r /etc/apparmor.d/usr.local.bin.sssonector
```

## Monitoring Setup

### 1. Prometheus Configuration

```yaml
scrape_configs:
  - job_name: 'sssonector'
    static_configs:
      - targets: ['localhost:8080']
    metrics_path: '/metrics'
```

### 2. Grafana Setup

1. Add Prometheus data source
2. Import dashboard from `monitoring/grafana/dashboards/sssonector.json`
3. Configure alerts

## Maintenance

### 1. Backup

```bash
# Backup configuration
tar czf sssonector-config-$(date +%Y%m%d).tar.gz /etc/sssonector/

# Backup certificates
tar czf sssonector-certs-$(date +%Y%m%d).tar.gz /etc/sssonector/certs/
```

### 2. Updates

```bash
# Update from source
git pull
make build
sudo make install

# Update containers
docker-compose pull
docker-compose up -d
```

### 3. Log Management

```bash
# Configure log rotation
sudo cp init/logrotate/sssonector /etc/logrotate.d/

# View logs
journalctl -u sssonector.service
```

## Troubleshooting

### 1. Service Issues

```bash
# Check service status
systemctl status sssonector

# View logs
journalctl -u sssonector -f

# Check configuration
sssonectorctl validate --config /etc/sssonector/config.json
```

### 2. Connection Issues

```bash
# Test connectivity
sssonectorctl test --target example.com:8443

# Check certificates
sssonectorctl cert verify

# Monitor traffic
tcpdump -i any port 8443
```

### 3. Performance Issues

```bash
# Enable debug logging
sssonectorctl debug --enable

# Profile CPU usage
sssonectorctl profile cpu

# Monitor metrics
curl http://localhost:8080/metrics
```

## Best Practices

1. Security:
   - Use certificate authentication
   - Enable TLS 1.3
   - Regular certificate rotation
   - Proper file permissions

2. Monitoring:
   - Set up alerts
   - Monitor resource usage
   - Regular log review
   - Performance tracking

3. Maintenance:
   - Regular backups
   - Scheduled updates
   - Log rotation
   - Health checks

4. Performance:
   - Tune system limits
   - Optimize network settings
   - Monitor resource usage
   - Regular profiling

## Support

- GitHub Issues: https://github.com/o3willard-AI/SSSonector/issues
- Documentation: https://github.com/o3willard-AI/SSSonector/docs
- Security: security@sssonector.example.com
