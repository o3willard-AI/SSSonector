# SSSonector Troubleshooting Guide

This guide provides solutions for common issues that may arise when running and maintaining SSSonector.

## Service Control Issues

### Service Won't Start

1. Check socket permissions:
```bash
ls -l /var/run/sssonector.sock
```
- Socket should have correct ownership and permissions (typically 660)
- Directory should be writable by the service user

2. Check service status:
```bash
sssonectorctl status
```
- Look for specific error messages
- Verify service state

3. Common causes:
- Socket file already exists (remove stale socket)
- Insufficient permissions
- Port conflicts
- Configuration errors

### Connection Refused

1. Verify socket exists:
```bash
ls -l /var/run/sssonector.sock
```

2. Check service is running:
```bash
ps aux | grep sssonector
```

3. Common causes:
- Service not running
- Wrong socket path
- Permission issues
- SELinux/AppArmor restrictions

### Command Timeouts

1. Check service load:
```bash
sssonectorctl metrics
```

2. Verify system resources:
```bash
top
df -h
```

3. Common causes:
- High system load
- Resource exhaustion
- Deadlocks
- Network issues

## Configuration Issues

### Invalid Configuration

1. Validate config file:
```bash
sssonectorctl validate --config=/etc/sssonector/config.json
```

2. Check config permissions:
```bash
ls -l /etc/sssonector/config.json
```

3. Common issues:
- Syntax errors
- Invalid values
- Missing required fields
- Permission problems

### Hot Reload Failures

1. Check reload status:
```bash
sssonectorctl reload
```

2. Verify config changes:
```bash
diff /etc/sssonector/config.json /etc/sssonector/config.json.bak
```

3. Common causes:
- Invalid new configuration
- Service in wrong state
- Resource constraints
- Lock contention

## Monitoring Issues

### Missing Metrics

1. Check monitoring status:
```bash
sssonectorctl metrics
```

2. Verify Prometheus endpoint:
```bash
curl http://localhost:9090/metrics
```

3. Common causes:
- Monitoring not enabled
- Wrong port configuration
- Network restrictions
- Permission issues

### Health Check Failures

1. Run health check:
```bash
sssonectorctl health
```

2. Check component status:
```bash
sssonectorctl status
```

3. Common issues:
- Resource exhaustion
- Network connectivity
- Configuration problems
- Component failures

## Security Issues

### Certificate Problems

1. Check certificate status:
```bash
sssonectorctl status
```

2. Verify certificate files:
```bash
ls -l /etc/sssonector/certs/
```

3. Common causes:
- Expired certificates
- Missing CA certificates
- Permission issues
- Wrong certificate paths

### Authentication Failures

1. Check auth logs:
```bash
journalctl -u sssonector
```

2. Verify credentials:
```bash
sssonectorctl verify-auth
```

3. Common issues:
- Invalid credentials
- Expired certificates
- Wrong authentication method
- Permission problems

## Network Issues

### Tunnel Problems

1. Check tunnel status:
```bash
sssonectorctl status
```

2. Verify network interface:
```bash
ip link show
```

3. Common causes:
- Interface conflicts
- Route problems
- Firewall rules
- MTU issues

### Performance Issues

1. Check rate limiting:
```bash
sssonectorctl metrics
```

2. Monitor network performance:
```bash
iftop -i tun0
```

3. Common issues:
- Rate limiting too aggressive
- Network congestion
- Resource constraints
- Configuration problems

## System Integration

### Systemd Issues

1. Check service status:
```bash
systemctl status sssonector
```

2. View service logs:
```bash
journalctl -u sssonector -n 100
```

3. Common problems:
- Wrong service configuration
- Permission issues
- Dependency failures
- Resource limits

### Platform-Specific Issues

#### Linux
- SELinux/AppArmor profiles
- Namespace configuration
- Capability requirements
- CGroup constraints

#### Windows
- Service registration
- Network adapter permissions
- Registry access
- Windows Firewall rules

#### macOS
- System extensions
- Network extension permissions
- Keychain access
- Gatekeeper restrictions

## Debugging Tips

1. Enable debug logging:
```bash
sssonectorctl set-log-level debug
```

2. Collect diagnostics:
```bash
sssonectorctl diagnostics
```

3. Monitor real-time:
```bash
sssonectorctl monitor --follow
```

4. Generate support bundle:
```bash
sssonectorctl support-bundle
```

## Common Error Codes

- `not_running`: Service is not running
- `already_running`: Service is already running
- `invalid_command`: Invalid command received
- `invalid_config`: Invalid configuration
- `internal_error`: Internal service error

## Getting Help

1. Check documentation:
- Implementation guide
- API reference
- Configuration guide

2. Generate debug info:
```bash
sssonectorctl debug-info > debug.txt
```

3. Contact support:
- Include debug info
- Describe steps to reproduce
- Provide configuration
- Include relevant logs
