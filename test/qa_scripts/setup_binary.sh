#!/bin/bash

# Setup SSSonector binary
# This script builds and installs the SSSonector binary on both server and client

set -euo pipefail

SERVER_IP=${SERVER_IP:-"192.168.50.210"}
CLIENT_IP=${CLIENT_IP:-"192.168.50.211"}
PROJECT_ROOT="/home/sblanken/Desktop/go/src/github.com/o3willard-AI/SSSonector"

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

# Function to build binary
build_binary() {
    log_info "Building SSSonector binary..."
    cd "$PROJECT_ROOT"
    go build -o sssonector ./cmd/tunnel/main.go || {
        log_error "Failed to build binary"
        return 1
    }
    log_info "Binary built successfully"
}

# Function to install binary on a host
install_binary() {
    local host=$1
    log_info "Installing binary on $host..."
    
    # Copy binary
    scp "$PROJECT_ROOT/sssonector" "${host}:/tmp/sssonector"
    
    # Install binary
    ssh "$host" "
        sudo mv /tmp/sssonector /usr/bin/sssonector
        sudo chown root:root /usr/bin/sssonector
        sudo chmod 755 /usr/bin/sssonector
        
        # Verify installation
        if ! command -v sssonector &>/dev/null; then
            echo 'Error: Binary not properly installed'
            exit 1
        fi
        
        # Verify executable
        if ! sssonector -version &>/dev/null; then
            echo 'Error: Binary not executable'
            exit 1
        fi
    "
    
    log_info "Binary installed successfully on $host"
}

# Build binary
build_binary || exit 1

# Install on both hosts
install_binary "$SERVER_IP" || {
    log_error "Failed to install binary on server"
    exit 1
}

install_binary "$CLIENT_IP" || {
    log_error "Failed to install binary on client"
    exit 1
}

log_info "Binary setup complete"
