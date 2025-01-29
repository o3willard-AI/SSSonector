#!/bin/bash
set -e

# Configuration
CERT_DIR="certs"
DAYS=3650  # 10 years
KEY_SIZE=4096
COUNTRY="US"
STATE="California"
LOCALITY="San Francisco"
ORGANIZATION="SSSonector"
ORG_UNIT="Development"
CA_CN="SSSonector Root CA"
SERVER_CN="SSSonector Server"
CLIENT_CN="SSSonector Client"

# Create certificates directory
mkdir -p "$CERT_DIR"
cd "$CERT_DIR"

# Generate CA private key and certificate
echo "Generating CA certificate..."
openssl genrsa -out ca.key "$KEY_SIZE"
openssl req -new -x509 -days "$DAYS" -key ca.key -out ca.crt -subj "/C=$COUNTRY/ST=$STATE/L=$LOCALITY/O=$ORGANIZATION/OU=$ORG_UNIT/CN=$CA_CN"

# Generate server private key and CSR
echo "Generating server certificate..."
openssl genrsa -out server.key "$KEY_SIZE"
openssl req -new -key server.key -out server.csr -subj "/C=$COUNTRY/ST=$STATE/L=$LOCALITY/O=$ORGANIZATION/OU=$ORG_UNIT/CN=$SERVER_CN"

# Create server certificate extensions file
cat > server.ext << EOF
authorityKeyIdentifier=keyid,issuer
basicConstraints=CA:FALSE
keyUsage = digitalSignature, nonRepudiation, keyEncipherment, dataEncipherment
subjectAltName = @alt_names

[alt_names]
DNS.1 = localhost
DNS.2 = tunnel.example.com
IP.1 = 127.0.0.1
EOF

# Sign server certificate
openssl x509 -req -days "$DAYS" -in server.csr -CA ca.crt -CAkey ca.key -CAcreateserial \
    -out server.crt -extfile server.ext

# Generate client private key and CSR
echo "Generating client certificate..."
openssl genrsa -out client.key "$KEY_SIZE"
openssl req -new -key client.key -out client.csr -subj "/C=$COUNTRY/ST=$STATE/L=$LOCALITY/O=$ORGANIZATION/OU=$ORG_UNIT/CN=$CLIENT_CN"

# Create client certificate extensions file
cat > client.ext << EOF
authorityKeyIdentifier=keyid,issuer
basicConstraints=CA:FALSE
keyUsage = digitalSignature, nonRepudiation, keyEncipherment, dataEncipherment
extendedKeyUsage = clientAuth
EOF

# Sign client certificate
openssl x509 -req -days "$DAYS" -in client.csr -CA ca.crt -CAkey ca.key -CAcreateserial \
    -out client.crt -extfile client.ext

# Clean up CSR files
rm -f server.csr client.csr server.ext client.ext

# Set permissions
chmod 600 *.key
chmod 644 *.crt

echo "Certificate generation complete!"
echo "Files generated in $CERT_DIR/:"
ls -l

# Instructions
echo
echo "To use these certificates:"
echo "1. Copy ca.crt, server.crt, and server.key to the server's /etc/sssonector/certs/ directory"
echo "2. Copy ca.crt, client.crt, and client.key to the client's /etc/sssonector/certs/ directory"
echo "3. Ensure correct permissions: chmod 600 for .key files, chmod 644 for .crt files"
