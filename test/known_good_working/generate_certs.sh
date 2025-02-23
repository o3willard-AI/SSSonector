#!/bin/bash
set -e

# Generate CA key and certificate
openssl genrsa -out ca.key 4096
openssl req -new -x509 -days 365 -key ca.key -out ca.crt -subj "/CN=SSSonector CA"

# Generate server key and CSR
openssl genrsa -out server.key 4096
openssl req -new -key server.key -out server.csr -subj "/CN=sssonector-server"

# Sign server certificate with CA
openssl x509 -req -days 365 -in server.csr -CA ca.crt -CAkey ca.key -CAcreateserial -out server.crt

# Clean up CSR
rm server.csr

# Set permissions
chmod 600 *.key
chmod 644 *.crt
