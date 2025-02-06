# Certificate Management in SSSonector

## Feature Flags

SSSonector provides several command-line flags for certificate management:

### Core Certificate Flags

1. `-test-without-certs`
   - Purpose: Run in test mode with temporary certificates
   - Use Case: Testing connectivity without valid certificates
   - Example: `sssonector -mode server -test-without-certs -config config.yaml`
   - Note: Certificates expire after 15 seconds; not for production use

2. `-generate-certs-only`
   - Purpose: Generate certificates without starting the service
   - Use Case: Pre-generating certificates for later use
   - Example: `sssonector -generate-certs-only -keyfile /etc/sssonector/certs`
   - Note: Creates both server and client certificates

3. `-keyfile <directory>`
   - Purpose: Specify certificate directory location
   - Use Case: Custom certificate locations
   - Example: `sssonector -mode server -keyfile /custom/cert/path`
   - Default: `/etc/sssonector/certs`

### Additional Certificate Features

4. `-keygen`
   - Purpose: Generate production SSL certificates
   - Use Case: Initial certificate setup
   - Example: `sssonector -keygen -keyfile /etc/sssonector/certs`
   - Note: Creates long-lived certificates for production use

5. `-validate-certs`
   - Purpose: Validate existing certificates
   - Use Case: Certificate verification and troubleshooting
   - Example: `sssonector -validate-certs -keyfile /etc/sssonector/certs`
   - Note: Checks certificate validity and relationships

## Certificate Types

### Temporary Certificates
- Generated with `-test-without-certs`
- 15-second expiration
- Automatically cleaned up
- Not for production use
- Useful for testing and development

### Production Certificates
- Generated with `-keygen`
- Long-lived (1 year by default)
- Requires proper file permissions
- Suitable for production environments
- Should be stored securely

## Certificate File Structure

```
/etc/sssonector/certs/
├── ca.crt     # Certificate Authority certificate
├── ca.key     # CA private key
├── server.crt # Server certificate
├── server.key # Server private key
├── client.crt # Client certificate
└── client.key # Client private key
```

## File Permissions

```bash
# Set proper permissions
chmod 600 /etc/sssonector/certs/*.key  # Private keys
chmod 644 /etc/sssonector/certs/*.crt  # Public certificates
```

## Common Use Cases

### 1. Testing Setup
```bash
# Run server in test mode
sssonector -mode server -test-without-certs -config config.yaml

# Run client in test mode
sssonector -mode client -test-without-certs -config config.yaml
```

### 2. Production Setup
```bash
# Generate certificates
sssonector -keygen -keyfile /etc/sssonector/certs

# Validate certificates
sssonector -validate-certs -keyfile /etc/sssonector/certs

# Run server with production certificates
sssonector -mode server -keyfile /etc/sssonector/certs -config config.yaml
```

### 3. Custom Certificate Location
```bash
# Generate certificates in custom location
sssonector -generate-certs-only -keyfile /custom/path

# Run with custom certificate location
sssonector -mode server -keyfile /custom/path -config config.yaml
```

## Best Practices

1. **Testing**
   - Use `-test-without-certs` for development and testing
   - Don't use temporary certificates in production
   - Clean up test certificates after use

2. **Production**
   - Use `-keygen` for production certificates
   - Set proper file permissions
   - Store certificates securely
   - Validate certificates before use

3. **Maintenance**
   - Regularly check certificate expiration
   - Keep backup copies of certificates
   - Plan for certificate rotation
   - Monitor certificate-related logs

## Troubleshooting

### Common Issues

1. **Certificate Validation Failures**
   ```bash
   # Check certificate validity
   sssonector -validate-certs -keyfile /path/to/certs
   
   # Inspect certificate details
   openssl x509 -in /path/to/certs/server.crt -text -noout
   ```

2. **Permission Issues**
   ```bash
   # Fix permissions
   chmod 600 /path/to/certs/*.key
   chmod 644 /path/to/certs/*.crt
   ```

3. **Certificate Mismatch**
   ```bash
   # Verify certificate chain
   openssl verify -CAfile ca.crt server.crt
   openssl verify -CAfile ca.crt client.crt
   ```

### Debug Tips

1. Check certificate expiration:
   ```bash
   openssl x509 -in server.crt -noout -dates
   ```

2. Verify certificate ownership:
   ```bash
   ls -l /path/to/certs/
   ```

3. Test certificate loading:
   ```bash
   openssl x509 -in server.crt -noout -text
   ```

## Security Considerations

1. **Private Keys**
   - Never share private keys
   - Use strict file permissions
   - Store backups securely

2. **Certificate Authority**
   - Protect CA private key
   - Use separate CA for testing
   - Consider hardware security modules for production

3. **Certificate Management**
   - Implement certificate rotation
   - Monitor expiration dates
   - Use secure storage
   - Maintain certificate inventory
