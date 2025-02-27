#!/bin/bash

# setup_local_dev.sh
# Sets up local development environment for SSSonector with safety measures
set -euo pipefail

# Source network safety script
source "$(dirname "$0")/network_safety.sh"

# Local development settings
DEV_DIR="dev"
CERT_DIR="${DEV_DIR}/certs"
CONFIG_DIR="${DEV_DIR}/config"
LOG_DIR="${DEV_DIR}/logs"
SERVER_IP="127.0.0.1"
TUN_SERVER_IP="10.0.0.1"
TUN_CLIENT_IP="10.0.0.2"
TUN_NETMASK="255.255.255.0"

# Create local development directory structure
setup_dev_directories() {
    log_info "Creating development directory structure"
    
    mkdir -p "${CERT_DIR}"
    mkdir -p "${CONFIG_DIR}"
    mkdir -p "${LOG_DIR}"
    
    # Copy certificates
    cp certs/ca.{crt,key} "${CERT_DIR}/"
    cp certs/server.{crt,key} "${CERT_DIR}/"
    cp certs/client.{crt,key} "${CERT_DIR}/"
    
    # Set correct permissions
    chmod 600 "${CERT_DIR}"/*.key
    chmod 644 "${CERT_DIR}"/*.crt
}

# Create local configuration files
create_config_files() {
    log_info "Creating configuration files"
    
    # Server configuration
    cat > "${CONFIG_DIR}/server.yaml" << EOF
mode: server
listen: ${SERVER_IP}:443
interface: tun0
address: ${TUN_SERVER_IP}/24
log_level: debug
metrics:
  enabled: true
  port: 9090
security:
  tls:
    enabled: true
    min_version: "1.2"
    cert_file: ${CERT_DIR}/server.crt
    key_file: ${CERT_DIR}/server.key
    ca_file: ${CERT_DIR}/ca.crt
  memory_protections:
    enabled: true
    no_exec: true
    aslr: true
  namespace:
    enabled: true
    network: true
    mount: true
monitoring:
  enabled: true
  interval: 5s
  prometheus:
    enabled: true
    port: 9091
    path: /metrics
EOF

    # Client configuration
    cat > "${CONFIG_DIR}/client.yaml" << EOF
mode: client
server: ${SERVER_IP}:443
interface: tun1
address: ${TUN_CLIENT_IP}/24
log_level: debug
metrics:
  enabled: true
  port: 9092
security:
  tls:
    enabled: true
    min_version: "1.2"
    cert_file: ${CERT_DIR}/client.crt
    key_file: ${CERT_DIR}/client.key
    ca_file: ${CERT_DIR}/ca.crt
  memory_protections:
    enabled: true
    no_exec: true
    aslr: true
  namespace:
    enabled: true
    network: true
    mount: true
monitoring:
  enabled: true
  interval: 5s
  prometheus:
    enabled: true
    port: 9093
    path: /metrics
EOF
}

# Configure TUN support
setup_tun() {
    log_info "Setting up TUN support"
    
    # Check if TUN is built into kernel
    if grep -q "CONFIG_TUN=y" /boot/config-$(uname -r); then
        log_info "TUN support is built into kernel"
        
        # Create /dev/net/tun if it doesn't exist
        if [ ! -e /dev/net/tun ]; then
            log_info "Creating /dev/net/tun device"
            mkdir -p /dev/net
            mknod /dev/net/tun c 10 200
            chmod 666 /dev/net/tun
        fi
        
        # Verify /dev/net/tun exists and has correct permissions
        if [ ! -e /dev/net/tun ]; then
            log_error "Failed to create TUN device"
            return 1
        fi
        
        return 0
    else
        # Check if TUN module is available as a loadable module
        if ! lsmod | grep -q "^tun"; then
            log_info "Loading TUN module"
            modprobe tun || {
                log_error "Failed to load TUN module"
                return 1
            }
        fi
    fi
    
    # Verify TUN functionality
    if ! [ -e /dev/net/tun ] || ! [ -w /dev/net/tun ]; then
        log_error "TUN device not accessible"
        return 1
    fi
    
    log_info "TUN support verified successfully"
}

# Configure IP forwarding
setup_ip_forwarding() {
    log_info "Setting up IP forwarding"
    
    # Enable IP forwarding temporarily
    echo 1 | sudo tee /proc/sys/net/ipv4/ip_forward > /dev/null
    
    # Verify IP forwarding
    if [ "$(cat /proc/sys/net/ipv4/ip_forward)" != "1" ]; then
        log_error "Failed to enable IP forwarding"
        return 1
    fi
}

# Cleanup function
dev_cleanup() {
    log_info "Cleaning up development environment"
    
    # Stop any running instances
    pkill -f sssonector || true
    
    # Remove TUN interfaces
    ip link delete tun0 2>/dev/null || true
    ip link delete tun1 2>/dev/null || true
    
    # Call network safety cleanup
    cleanup
}

# Main function
main() {
    log_info "Starting local development environment setup"
    
    # Register cleanup handler
    trap dev_cleanup EXIT
    
    # Initialize network safety measures
    if ! verify_connectivity; then
        log_error "Initial network verification failed"
        exit 1
    fi
    
    # Create development environment
    setup_dev_directories
    create_config_files
    
    # Setup network components with safety checks
    setup_tun || exit 1
    setup_ip_forwarding || exit 1
    
    # Final verification
    if ! verify_connectivity; then
        log_error "Final network verification failed"
        exit 1
    fi
    
    log_info "Local development environment setup completed successfully"
    log_info "Development directory: ${DEV_DIR}"
    log_info "Server configuration: ${CONFIG_DIR}/server.yaml"
    log_info "Client configuration: ${CONFIG_DIR}/client.yaml"
}

# Run main function if script is executed directly
if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then
    main "$@"
fi
