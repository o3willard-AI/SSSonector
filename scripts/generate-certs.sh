#!/bin/bash

# Exit on any error
set -e

CERT_DIR="../certs"
DAYS=365
KEY_SIZE=4096
COUNTRY="US"
STATE="California"
LOCALITY="San Francisco"
ORGANIZATION="SSSonector"
SERVER_CN="sssonector-server"
CLIENT_CN="sssonector-client"

# Create certificates directory if it doesn't exist
mkdir -p "$CERT_DIR"

# Function to generate a certificate
generate_cert() {
    local name=$1
    local cn=$2
    local key_file="$CERT_DIR/$name.key"
    local csr_file="$CERT_DIR/$name.csr"
    local crt_file="$CERT_DIR/$name.crt"

    echo "Generating $name certificate..."

    # Generate private key
    openssl genrsa -out "$key_file" $KEY_SIZE

    # Generate CSR
    openssl req -new -key "$key_file" -out "$csr_file" -subj "/C=$COUNTRY/ST=$STATE/L=$LOCALITY/O=$ORGANIZATION/CN=$cn"

    # Generate self-signed certificate
    openssl x509 -req -days $DAYS -in "$csr_file" -signkey "$key_file" -out "$crt_file"

    # Remove CSR file
    rm "$csr_file"

    # Set permissions
    chmod 600 "$key_file"
    chmod 644 "$crt_file"

    echo "Generated $name certificate files:"
    echo "  Private key: $key_file"
    echo "  Certificate: $crt_file"
    echo
}

echo "Generating SSL certificates for SSSonector..."
echo "Certificates will be valid for $DAYS days"
echo

# Generate server certificate
generate_cert "server" "$SERVER_CN"

# Generate client certificate
generate_cert "client" "$CLIENT_CN"

echo "Certificate generation complete!"
echo "Certificate files are in: $CERT_DIR"
echo
echo "Remember to:"
echo "1. Copy the certificates to /etc/sssonector/certs/"
echo "2. Set proper permissions (600 for .key files, 644 for .crt files)"
echo "3. Update the configuration files with the correct certificate paths"
