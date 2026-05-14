# TLS Certificate Generation Guide

This document explains how to generate TLS certificates for SSSonector, including CA certificates, server certificates with IP Subject Alternative Names (SANs), and client certificates.

## Generating a CA Certificate

To create a Certificate Authority (CA) certificate:

```bash
openssl req -newkey rsa:4096 -nodes -keyout ca.key -x509 -days 3650 -out ca.crt -subj "/C=US/ST=State/L=City/O=Organization/CN=SSSonector CA"
```

## Generating a Server Certificate with IP SANs

When the client connects to the server by IP address (e.g., `192.168.101.220`), Go's TLS verification requires the server certificate to have that IP in its Subject Alternative Names. A cert with only `CN=server.sssonector.local` will fail with:

```
tls: failed to verify certificate: x509: cannot validate certificate for 192.168.101.220 because it doesn't contain any IP SANs
```

To generate a server certificate with IP SANs, first create an OpenSSL configuration file:

```bash
cat > /tmp/san.cnf << EOF
[req]
default_bits = 4096
prompt = no
default_md = sha256
distinguished_name = dn
req_extensions = v3_req

[dn]
C = US
ST = State
L = City
O = Organization
CN = server.sssonector.local

[v3_req]
basicConstraints = CA:FALSE
keyUsage = nonRepudiation, digitalSignature, keyEncipherment
subjectAltName = @alt_names

[alt_names]
DNS.1 = server.sssonector.local
DNS.2 = localhost
IP.1 = 192.168.101.220
IP.2 = 127.0.0.1
IP.3 = ::1
EOF
```

Then generate the server certificate:

```bash
openssl req -newkey rsa:4096 -nodes -keyout server.key -new -x509 -days 3650 -out server.crt -subj "/C=US/ST=State/L=City/O=Organization/CN=server.sssonector.local" -config /tmp/san.cnf -extensions v3_req
```

## Generating a Client Certificate

To generate a client certificate:

```bash
openssl req -newkey rsa:4096 -nodes -keyout client.key -new -x509 -days 3650 -out client.crt -subj "/C=US/ST=State/L=City/O=Organization/CN=client.sssonector.local"
```

## File Permissions

For security, set appropriate file permissions:

- Private keys: `600` (read/write for owner only)
- Certificates: `644` (readable by all, writable by owner)

```bash
chmod 600 ca.key server.key client.key
chmod 644 ca.crt server.crt client.crt
```

## Expected Configuration Paths

The certificates should be placed in the following paths within the SSSonector configuration:

- CA Certificate: `configs/certs/ca.crt`
- Server Certificate: `configs/certs/server.crt`
- Server Key: `configs/certs/server.key`
- Client Certificate: `configs/certs/client.crt`
- Client Key: `configs/certs/client.key`

## Common Pitfalls

1. **Missing IP SANs**: When connecting to the server by IP address, the certificate must contain the IP address in its Subject Alternative Names. Without this, TLS verification will fail.

2. **Incorrect file permissions**: Private keys should have 600 permissions and certificates should have 644 permissions to prevent unauthorized access.

3. **Wrong CN vs SAN usage**: The Common Name (CN) field is deprecated for server identification in modern TLS implementations. Use Subject Alternative Names (SANs) instead.