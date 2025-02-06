# Certificate Management Guide

## Overview

SSSonector uses X.509 certificates for secure authentication and encryption. This guide covers certificate management features, including generation, validation, and rotation.

## Command Line Flags

SSSonector provides five key certificate management flags:

1. `-test-without-certs`: Run with temporary certificates
   - Generates ephemeral certificates for testing
   - Not suitable for production use
   - Certificates are automatically cleaned up

2. `-generate-certs-only`: Generate certificates without starting service
   - Creates production-grade certificates
   - Useful for pre-deployment setup
   - Generates both client and server certificates

3. `-keyfile`: Specify certificate directory
   - Custom location for certificate storage
   - Supports both relative and absolute paths
   - Default: `/etc/sssonector/certs`

4. `-keygen`: Generate production certificates
   - Creates long-lived production certificates
   - Includes full certificate chain
   - Configurable key sizes and algorithms

5. `-validate-certs`: Validate existing certificates
   - Checks certificate validity
   - Verifies certificate chain
   - Reports expiration status

## Certificate Structure

### Directory Layout
```
/etc/sssonector/certs/
├── ca/
│   ├── ca.crt
│   └── ca.key
├── server/
│   ├── server.crt
│   └── server.key
└── client/
    ├── client.crt
    └── client.key
```

### File Permissions
```bash
# CA certificates
ca.crt: 644 (rw-r--r--)
ca.key: 600 (rw-------)

# Server certificates
server.crt: 644 (rw-r--r--)
server.key: 600 (rw-------)

# Client certificates
client.crt: 644 (rw-r--r--)
client.key: 600 (rw-------)
```

## Certificate Generation

### Production Certificates
```bash
# Generate complete certificate set
sssonector -keygen

# Specify custom options
sssonector -keygen \
  -key-type rsa \
  -key-size 4096 \
  -days 365 \
  -country "US" \
  -org "Example Corp"
```

### Testing Certificates
```bash
# Generate temporary certificates
sssonector -test-without-certs

# Generate test certificates without starting service
sssonector -generate-certs-only -test
```

### Custom Location
```bash
# Generate certificates in custom location
sssonector -keygen -keyfile /path/to/certs

# Use existing certificates from custom location
sssonector -keyfile /path/to/certs
```

## Certificate Validation

### Basic Validation
```bash
# Validate all certificates
sssonector -validate-certs

# Validate specific certificate
sssonector -validate-certs -cert /path/to/cert.crt
```

### Detailed Validation
```bash
# Show detailed validation information
sssonector -validate-certs -verbose

# Check certificate chain
sssonector -validate-certs -check-chain

# Verify against CA
sssonector -validate-certs -ca /path/to/ca.crt
```

## Certificate Rotation

### Automatic Rotation
The certificate manager now supports automatic certificate rotation:

```yaml
certificates:
  auto_rotate: true
  rotation_threshold_days: 30
  backup_enabled: true
  backup_location: "/etc/sssonector/certs/backup"
```

### Manual Rotation
```bash
# Rotate all certificates
sssonector -rotate-certs

# Rotate specific certificate
sssonector -rotate-cert -cert server
```

## Monitoring

### Certificate Status
```bash
# View certificate status
sssonector -cert-status

# Monitor expiration
sssonector -cert-monitor
```

### Metrics
The monitoring system now includes certificate-related metrics:

```yaml
monitor:
  cert_metrics:
    enabled: true
    check_interval: 3600
    alert_threshold_days: 30
```

## Security Considerations

### Key Protection
- Use secure permissions (600 for private keys)
- Store keys in secure location
- Use hardware security modules when available

### Certificate Policies
- Regular rotation (recommended: 1 year)
- Strong key sizes (RSA 4096, ECC P-384)
- Proper chain of trust

### Best Practices
1. Regular validation checks
2. Automated rotation
3. Secure backup storage
4. Audit logging
5. Access control

## Troubleshooting

### Common Issues

1. Certificate Validation Failures
   ```bash
   # Check certificate validity
   openssl verify -CAfile ca.crt cert.crt
   
   # View certificate details
   openssl x509 -in cert.crt -text -noout
   ```

2. Permission Problems
   ```bash
   # Fix permissions
   chmod 600 private.key
   chmod 644 public.crt
   ```

3. Chain Issues
   ```bash
   # Verify chain
   openssl verify -verbose -CAfile ca.crt -untrusted intermediate.crt cert.crt
   ```

### Logging
Enhanced certificate-related logging is available:

```bash
# Enable certificate debugging
sssonector -log-level debug -cert-debug

# View certificate operations
tail -f /var/log/sssonector/cert.log
```

## Integration

### SNMP Monitoring
Certificate status is now available via SNMP:

```bash
# Query certificate status
snmpwalk -v2c -c public localhost .1.3.6.1.4.1.X.cert

# Monitor expiration
snmpget -v2c -c public localhost .1.3.6.1.4.1.X.cert.expiry
```

### Metrics Integration
Certificate metrics are exposed for monitoring systems:

```bash
# Prometheus metrics
cert_expiry_days{cert="server"} 180
cert_validation_status{cert="client"} 1
```

## Backup and Recovery

### Backup Procedures
```bash
# Backup certificates
sssonector -backup-certs

# Restore from backup
sssonector -restore-certs -backup backup.tar.gz
```

### Emergency Recovery
```bash
# Generate emergency certificates
sssonector -emergency-certs

# Restore from last known good
sssonector -restore-last-good
