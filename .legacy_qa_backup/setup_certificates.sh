#!/bin/bash

# Setup certificates for SSSonector testing
# This script generates and distributes TLS certificates for secure communication

set -euo pipefail

SERVER_HOST="sblanken@192.168.50.210"
CLIENT_HOST="sblanken@192.168.50.211"
CERT_DIR="/tmp/sssonector_certs"

# Color codes for output
GREEN='\033[0;32m'
RED='\033[0;31m'
YELLOW='\033[1;33m'
NC='\033[0m'

log_info() {
    echo -e "${GREEN}[INFO]${NC} $1"
}

log_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

log_warn() {
    echo -e "${YELLOW}[WARN]${NC} $1"
}

# Generate CA and certificates
generate_certificates() {
    log_info "Generating certificates..."
    
    # Create temporary directory
    rm -rf "$CERT_DIR"
    mkdir -p "$CERT_DIR"
    cd "$CERT_DIR"
    
    # Generate CA private key and certificate
    openssl genrsa -out ca.key 4096
    openssl req -new -x509 -days 365 -key ca.key -out ca.crt -subj "/CN=SSSonector Test CA"
    
    # Generate server private key and CSR
    openssl genrsa -out server.key 2048
    openssl req -new -key server.key -out server.csr -subj "/CN=sssonector-server"
    
    # Generate client private key and CSR
    openssl genrsa -out client.key 2048
    openssl req -new -key client.key -out client.csr -subj "/CN=sssonector-client"
    
    # Create config file for certificate extensions
    cat > extfile.cnf <<EOF
authorityKeyIdentifier=keyid,issuer
basicConstraints=CA:FALSE
keyUsage = digitalSignature, nonRepudiation, keyEncipherment, dataEncipherment
extendedKeyUsage = serverAuth, clientAuth
EOF
    
    # Sign server certificate
    openssl x509 -req -days 365 -in server.csr -CA ca.crt -CAkey ca.key -CAcreateserial \
        -out server.crt -extfile extfile.cnf
    
    # Sign client certificate
    openssl x509 -req -days 365 -in client.csr -CA ca.crt -CAkey ca.key -CAcreateserial \
        -out client.crt -extfile extfile.cnf
    
    # Set permissions
    chmod 600 *.key
    chmod 644 *.crt
}

# Install certificates on a host
install_certificates() {
    local host=$1
    local role=$2
    log_info "Installing certificates on $host..."
    
    # Create certificate directory
    ssh "$host" "mkdir -p ~/sssonector/certs"
    
    # Copy certificates
    scp "$CERT_DIR/ca.crt" "$host:~/sssonector/certs/"
    scp "$CERT_DIR/$role.key" "$host:~/sssonector/certs/"
    scp "$CERT_DIR/$role.crt" "$host:~/sssonector/certs/"
    
    # Set permissions
    ssh "$host" "
        chmod 700 ~/sssonector/certs
        chmod 600 ~/sssonector/certs/*.key
        chmod 644 ~/sssonector/certs/*.crt
    "
}

# Cleanup temporary files
cleanup() {
    log_info "Cleaning up temporary files..."
    rm -rf "$CERT_DIR"
}

# Main execution
main() {
    log_info "Starting certificate setup..."
    
    # Generate certificates
    generate_certificates
    
    # Install on server
    install_certificates "$SERVER_HOST" "server"
    
    # Install on client
    install_certificates "$CLIENT_HOST" "client"
    
    # Cleanup
    cleanup
    
    log_info "Certificate setup complete"
}

# Execute main function
main
