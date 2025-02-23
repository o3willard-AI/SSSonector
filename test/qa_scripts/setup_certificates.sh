#!/bin/bash

# Setup certificates for SSSonector test environment
# This script generates and distributes certificates to server and client machines

set -euo pipefail

SERVER_IP=${SERVER_IP:-"192.168.50.210"}
CLIENT_IP=${CLIENT_IP:-"192.168.50.211"}
PROJECT_ROOT="/home/sblanken/Desktop/go/src/github.com/o3willard-AI/SSSonector"
CERT_DIR="$PROJECT_ROOT/certs"

# Color codes for output
GREEN='\033[0;32m'
RED='\033[0;31m'
NC='\033[0m'

log_info() {
    echo -e "${GREEN}[INFO]${NC} $1"
}

log_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

# Create certificates directory
mkdir -p "$CERT_DIR"
cd "$CERT_DIR"

# Generate CA key and certificate
log_info "Generating CA certificate..."
openssl genrsa -out ca.key 4096
openssl req -new -x509 -days 365 -key ca.key -out ca.crt -subj "/CN=SSSonector Test CA"

# Generate server key and CSR
log_info "Generating server certificate..."
openssl genrsa -out server.key 4096
openssl req -new -key server.key -out server.csr -subj "/CN=server"

# Sign server certificate
openssl x509 -req -days 365 -in server.csr -CA ca.crt -CAkey ca.key -CAcreateserial -out server.crt

# Generate client key and CSR
log_info "Generating client certificate..."
openssl genrsa -out client.key 4096
openssl req -new -key client.key -out client.csr -subj "/CN=client"

# Sign client certificate
openssl x509 -req -days 365 -in client.csr -CA ca.crt -CAkey ca.key -CAcreateserial -out client.crt

# Set proper permissions
chmod 600 *.key
chmod 644 *.crt

# Function to deploy certificates to a host
deploy_certs() {
    local host=$1
    local mode=$2
    log_info "Deploying certificates to $host..."
    
    # Create certificates directory and set permissions
    ssh "$host" "
        sudo rm -rf /etc/sssonector/certs
        sudo mkdir -p /etc/sssonector/certs
        sudo chown root:root /etc/sssonector/certs
        sudo chmod 755 /etc/sssonector/certs
    "
    
    # Copy certificates
    scp ca.crt "${host}:/tmp/ca.crt"
    scp "${mode}.key" "${host}:/tmp/${mode}.key"
    scp "${mode}.crt" "${host}:/tmp/${mode}.crt"
    
    # Move certificates to proper location and set permissions
    ssh "$host" "
        sudo mv /tmp/ca.crt /etc/sssonector/certs/ca.crt
        sudo mv /tmp/${mode}.key /etc/sssonector/certs/${mode}.key
        sudo mv /tmp/${mode}.crt /etc/sssonector/certs/${mode}.crt
        sudo chown root:root /etc/sssonector/certs/*
        sudo chmod 600 /etc/sssonector/certs/${mode}.key
        sudo chmod 644 /etc/sssonector/certs/ca.crt
        sudo chmod 644 /etc/sssonector/certs/${mode}.crt
        ls -l /etc/sssonector/certs/
    "
}

# Deploy certificates
deploy_certs "$SERVER_IP" "server" || {
    log_error "Failed to deploy server certificates"
    exit 1
}

deploy_certs "$CLIENT_IP" "client" || {
    log_error "Failed to deploy client certificates"
    exit 1
}

log_info "Certificate setup complete"
