# SSSonector Certificate Management

This document describes the certificate management features in SSSonector, including automatic certificate generation, validation, and testing capabilities.

## Certificate Generation

SSSonector provides built-in certificate generation using the `-keygen` flag:

```bash
# Generate certificates in the current directory
sssonector -keygen

# Generate certificates in a specific directory
sssonector -keygen -keyfile /path/to/certs
```

This will generate the following files:
- `ca.crt`: CA certificate
- `ca.key`: CA private key
- `server.crt`: Server certificate
- `server.key`: Server private key
- `client.crt`: Client certificate
- `client.key`: Client private key

## Certificate Location

By default, SSSonector looks for certificates in the following locations:
1. Current working directory
2. Directory specified by `-keyfile` flag
3. Default system location (/etc/sssonector/certs)

You can specify a custom certificate location using the `-keyfile` flag:
```bash
sssonector -mode server -keyfile /path/to/certs
sssonector -mode client -keyfile /path/to/certs
```

## Certificate Validation

SSSonector performs comprehensive certificate validation:
- Verifies certificate chain against CA
- Validates certificate dates
- Checks key pair matches
- Verifies file permissions
- Validates certificate integrity

If any validation fails, detailed error messages will guide you to resolve the issue.

## Test Mode

SSSonector includes a test mode that uses temporary certificates for quick connectivity testing:

```bash
# Server side
sssonector -mode server -test-without-certs

# Client side
sssonector -mode client -test-without-certs
```

Test mode features:
- Generates temporary certificates valid for 15 seconds
- Automatically cleans up test certificates
- Performs basic connectivity test
- Reports connection status

**Important**: Test mode is for troubleshooting only. Never use test certificates in production.

## Configuration

Certificate paths can be specified in the configuration file:

```yaml
tunnel:
  cert_file: "/etc/sssonector/certs/server.crt"  # or client.crt
  key_file: "/etc/sssonector/certs/server.key"   # or client.key
  ca_file: "/etc/sssonector/certs/ca.crt"
```

The `-keyfile` flag will override these paths if specified.

## Best Practices

1. **Certificate Generation**:
   - Generate certificates on a secure system
   - Keep the CA private key secure
   - Use descriptive common names

2. **Certificate Distribution**:
   - Securely transfer certificates to clients
   - Verify certificate integrity after transfer
   - Never transfer private keys over insecure channels

3. **Certificate Management**:
   - Regularly rotate certificates
   - Monitor certificate expiration
   - Keep backup copies of certificates
   - Use appropriate file permissions

4. **Testing**:
   - Use test mode for initial setup and troubleshooting
   - Verify connectivity with test certificates
   - Always switch to proper certificates for production

## Troubleshooting

Common issues and solutions:

1. **Certificate Not Found**:
   - Verify file paths in configuration
   - Check file permissions
   - Use `-keyfile` to specify correct location

2. **Certificate Validation Failed**:
   - Check certificate dates
   - Verify CA chain
   - Ensure key pairs match
   - Check for file corruption

3. **Connection Failed**:
   - Use test mode to verify basic connectivity
   - Check certificate permissions
   - Verify server address and port
   - Check network connectivity

4. **Permission Issues**:
   - Ensure private keys are mode 600
   - Ensure certificates are mode 644
   - Check directory permissions
