#!/bin/bash

# deploy.sh
# Deploys SSSonector packages to local development environment
set -euo pipefail

# Import common utilities
source "$(dirname "${BASH_SOURCE[0]}")/../lib/common.sh"

# Local development settings
DEV_DIR="dev"
CERT_DIR="${DEV_DIR}/certs"
CONFIG_DIR="${DEV_DIR}/config"
LOG_DIR="${DEV_DIR}/logs"
SERVER_IP="127.0.0.1"

# Function to copy files
copy_files() {
    local src=$1
    local dest=$2

    log_info "Copying ${src} to ${dest}"
    cp -r "${src}" "${dest}"
}

# Function to execute command
execute() {
    local command=$1

    log_info "Executing command: ${command}"
    bash -c "${command}"
}

# Cleanup function
cleanup() {
    log_info "Cleaning up local development environment"

    # Stop any running instances
    pkill -f sssonector || true

    # Remove TUN interfaces
    ip link delete tun0 2>/dev/null || true
    ip link delete tun1 2>/dev/null || true
}

# Deploy function
deploy() {
    log_info "Deploying to local development environment"

    # Stop any running instances
    cleanup

    # Copy binary
    log_info "Copying binary"
    cp bin/sssonector "${DEV_DIR}/"

    # Copy configuration files
    log_info "Copying configuration files"
    cp test/sanity_check/configs/server.yaml "${CONFIG_DIR}/"
    cp test/sanity_check/configs/client.yaml "${CONFIG_DIR}/"

    # Copy certificates
    log_info "Copying certificates"
    cp certs/* "${CERT_DIR}/"

    # Set correct permissions
    chmod 755 "${DEV_DIR}/sssonector"
    chmod 644 "${CONFIG_DIR}"/*.yaml
    chmod 600 "${CERT_DIR}"/*.key
    chmod 644 "${CERT_DIR}"/*.crt

    log_info "Deployment to local development environment completed"
}

# Main function
main() {
    # Check if cleanup is requested
    if [[ "$1" == "--cleanup" ]]; then
        log_info "Cleaning up existing deployments"
        cleanup
        log_info "Cleanup completed successfully"
        return 0
    fi

    # Deploy
    deploy

    log_info "All operations completed successfully"
    return 0
}

# Run main function
main "$@"
