#!/bin/bash
set -e

# Configuration
CERT_DIR="certs"
DAYS=365
KEY_SIZE=4096
COUNTRY="EU"
STATE="Europe"
LOCALITY="European Union"
ORGANIZATION="SSSonector"
SERVER_CN="sssonector-server"
CLIENT_CN="sssonector-client"

# Create certificates directory
mkdir -p "$CERT_DIR"
cd "$CERT_DIR"

echo "Generating certificates in $CERT_DIR..."

# Generate CA key and certificate
echo "Generating CA certificate..."
openssl genrsa -out ca.key $KEY_SIZE
openssl req -new -x509 -days $DAYS -key ca.key -out ca.crt -subj "/C=$COUNTRY/ST=$STATE/L=$LOCALITY/O=$ORGANIZATION/CN=SSSonector CA"

# Generate server key and CSR
echo "Generating server certificate..."
openssl genrsa -out server.key $KEY_SIZE
openssl req -new -key server.key -out server.csr -subj "/C=$COUNTRY/ST=$STATE/L=$LOCALITY/O=$ORGANIZATION/CN=$SERVER_CN"

# Generate client key and CSR
echo "Generating client certificate..."
openssl genrsa -out client.key $KEY_SIZE
openssl req -new -key client.key -out client.csr -subj "/C=$COUNTRY/ST=$STATE/L=$LOCALITY/O=$ORGANIZATION/CN=$CLIENT_CN"

# Create config file for certificate extensions
cat > openssl.cnf << EOF
[ v3_server ]
basicConstraints = CA:FALSE
keyUsage = digitalSignature, keyEncipherment
extendedKeyUsage = serverAuth
subjectAltName = DNS:$SERVER_CN,DNS:localhost,IP:127.0.0.1

[ v3_client ]
basicConstraints = CA:FALSE
keyUsage = digitalSignature, keyEncipherment
extendedKeyUsage = clientAuth
subjectAltName = DNS:$CLIENT_CN
EOF

# Sign server certificate
echo "Signing server certificate..."
openssl x509 -req -days $DAYS -in server.csr -CA ca.crt -CAkey ca.key -CAcreateserial \
    -out server.crt -extfile openssl.cnf -extensions v3_server

# Sign client certificate
echo "Signing client certificate..."
openssl x509 -req -days $DAYS -in client.csr -CA ca.crt -CAkey ca.key -CAcreateserial \
    -out client.crt -extfile openssl.cnf -extensions v3_client

# Clean up CSRs and config
rm -f *.csr openssl.cnf

# Set permissions
chmod 600 *.key
chmod 644 *.crt

echo "Certificate generation complete!"
echo
echo "Generated files:"
ls -l

echo
echo "Certificate details:"
echo "==================="
echo
echo "CA Certificate:"
openssl x509 -in ca.crt -text -noout | grep "Subject:"
echo
echo "Server Certificate:"
openssl x509 -in server.crt -text -noout | grep -E "Subject:|DNS:|IP Address:"
echo
echo "Client Certificate:"
openssl x509 -in client.crt -text -noout | grep -E "Subject:|DNS:"

echo
echo "Verifying certificate chain..."
echo "=============================="
echo
echo "Server certificate:"
openssl verify -CAfile ca.crt server.crt
echo
echo "Client certificate:"
openssl verify -CAfile ca.crt client.crt

echo
echo "Testing TLS 1.3 compatibility..."
echo "==============================="
openssl ciphers -v 'TLSv1.3' | grep TLS_

echo
echo "Next steps:"
echo "1. Copy certificates to /etc/sssonector/certs/"
echo "2. Set correct permissions:"
echo "   sudo chown -R root:root /etc/sssonector/certs/"
echo "   sudo chmod 600 /etc/sssonector/certs/*.key"
echo "   sudo chmod 644 /etc/sssonector/certs/*.crt"
