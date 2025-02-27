#!/bin/bash

# Setup systemd service files for SSSonector
# This script creates and installs systemd service files on both server and client

set -euo pipefail

SERVER_HOST="sblanken@192.168.50.210"
CLIENT_HOST="sblanken@192.168.50.211"
PROJECT_ROOT="/home/sblanken/Desktop/go/src/github.com/o3willard-AI/SSSonector"

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

# Validate environment and requirements
validate_environment() {
    # Check SSH connectivity
    if ! ssh -o BatchMode=yes -o ConnectTimeout=5 "$SERVER_HOST" "echo 'Connection test'" &>/dev/null; then
        log_error "Cannot connect to server $SERVER_HOST. Please ensure SSH access is configured"
        exit 1
    fi
    if ! ssh -o BatchMode=yes -o ConnectTimeout=5 "$CLIENT_HOST" "echo 'Connection test'" &>/dev/null; then
        log_error "Cannot connect to client $CLIENT_HOST. Please ensure SSH access is configured"
        exit 1
    fi

    # Check if binary exists on both hosts
    for host in "$SERVER_HOST" "$CLIENT_HOST"; do
        if ! ssh "$host" "[ -x ~/sssonector/bin/sssonector ]"; then
            log_error "SSSonector binary not found or not executable on $host"
            exit 1
        fi
    done

    # Check if configuration exists on both hosts
    for host in "$SERVER_HOST" "$CLIENT_HOST"; do
        if ! ssh "$host" "[ -f ~/sssonector/config/config.yaml ]"; then
            log_error "Configuration file not found on $host"
            exit 1
        fi
    done
}

# Generate service file content based on mode
generate_service_content() {
    local mode=$1
    local user=$2
    cat <<EOF
[Unit]
Description=SSSonector Tunnel Service ($mode)
Documentation=https://github.com/o3willard-AI/SSSonector
After=network-online.target
Wants=network-online.target
StartLimitIntervalSec=300
StartLimitBurst=5

[Service]
Type=simple
ExecStart=/home/$user/sssonector/bin/sssonector -config /home/$user/sssonector/config/config.yaml -debug
KillMode=process
KillSignal=SIGTERM
Restart=always
RestartSec=5
User=root
Group=root
TimeoutStartSec=60
TimeoutStopSec=60

# Security settings
ProtectSystem=full
ProtectHome=read-only
PrivateTmp=true
NoNewPrivileges=true
ProtectKernelTunables=true
ProtectControlGroups=true
ProtectKernelModules=true
RestrictAddressFamilies=AF_INET AF_INET6 AF_UNIX AF_NETLINK
RestrictNamespaces=true
RestrictRealtime=true
MemoryDenyWriteExecute=true
SystemCallArchitectures=native

# Resource limits
LimitNOFILE=65535
LimitNPROC=4096
TasksMax=4096

# Runtime directory
RuntimeDirectory=sssonector
RuntimeDirectoryMode=0755

# Logging
StandardOutput=journal
StandardError=journal
SyslogIdentifier=sssonector-$mode

# Environment
Environment=GOMAXPROCS=2
Environment=GOGC=100

# Capabilities
CapabilityBoundingSet=CAP_NET_ADMIN CAP_NET_BIND_SERVICE CAP_NET_RAW
AmbientCapabilities=CAP_NET_ADMIN CAP_NET_BIND_SERVICE CAP_NET_RAW

[Install]
WantedBy=multi-user.target
EOF
}

# Function to setup systemd service on a host with enhanced security
setup_service() {
    local host=$1
    local mode=$2
    local max_retries=3
    local retry_count=0
    local temp_dir
    local user

    # Get remote username
    user=$(ssh "$host" "whoami")
    if [ -z "$user" ]; then
        log_error "Failed to get username on $host"
        return 1
    fi

    log_info "Setting up systemd service ($mode) on $host..."
    
    # Create secure temporary directory on remote host
    temp_dir=$(ssh "$host" "mktemp -d")
    if [ -z "$temp_dir" ]; then
        log_error "Failed to create temporary directory on $host"
        return 1
    fi

    # Cleanup function
    cleanup() {
        ssh "$host" "rm -rf $temp_dir" || log_warn "Failed to cleanup temporary directory on $host"
    }
    trap cleanup EXIT

    # Generate service content
    generate_service_content "$mode" "$user" > "/tmp/sssonector.service"

    while [ $retry_count -lt $max_retries ]; do
        if scp "/tmp/sssonector.service" "${host}:${temp_dir}/sssonector.service" &&
           ssh "$host" "
               # Validate service file syntax
               if ! systemd-analyze verify ${temp_dir}/sssonector.service &>/dev/null; then
                   echo 'Error: Invalid service file syntax'
                   exit 1
               fi

               # Stop service if running
               sudo systemctl stop sssonector.service &>/dev/null || true
               
               # Clean up any existing tun interfaces
               sudo ip tuntap del dev tun0 mode tun &>/dev/null || true
               
               # Install service file
               sudo mv ${temp_dir}/sssonector.service /etc/systemd/system/
               sudo chown root:root /etc/systemd/system/sssonector.service
               sudo chmod 644 /etc/systemd/system/sssonector.service
               
               # Create runtime directory
               sudo mkdir -p /run/sssonector
               sudo chown root:root /run/sssonector
               sudo chmod 755 /run/sssonector
               
               # Reload systemd and enable service
               sudo systemctl daemon-reload
               sudo systemctl enable sssonector.service
               
               # Verify service installation
               if ! systemctl cat sssonector.service &>/dev/null; then
                   echo 'Error: Service file not properly installed'
                   exit 1
               fi
           "; then
            log_info "Successfully configured systemd service on $host"
            rm -f "/tmp/sssonector.service"
            return 0
        fi
        
        retry_count=$((retry_count + 1))
        if [ $retry_count -lt $max_retries ]; then
            log_warn "Attempt $retry_count failed. Retrying in 5 seconds..."
            sleep 5
        fi
    done

    rm -f "/tmp/sssonector.service"
    log_error "Failed to setup service on $host after $max_retries attempts"
    return 1
}

# Main execution
main() {
    log_info "Starting systemd service setup..."
    
    # Validate environment first
    validate_environment
    
    # Setup services
    setup_service "$SERVER_HOST" "server" || {
        log_error "Failed to setup service on server"
        exit 1
    }
    
    setup_service "$CLIENT_HOST" "client" || {
        log_error "Failed to setup service on client"
        exit 1
    }
    
    log_info "Systemd service setup complete and verified"
}

# Execute main function
main
