#!/bin/bash

# deploy.sh
# Deploys the verification system to QA servers
set -euo pipefail

# Color codes for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m'

# Default values
DEFAULT_SERVER_IP="192.168.50.210"
DEFAULT_CLIENT_IP="192.168.50.211"
DEFAULT_USER="sblanken"
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REMOTE_BASE_DIR="/opt/sssonector"

# Logging functions
log_info() {
    echo -e "${GREEN}[INFO]${NC} $1"
}

log_warn() {
    echo -e "${YELLOW}[WARN]${NC} $1"
}

log_error() {
    echo -e "${RED}[ERROR]${NC} $1"
    return 1
}

# Print usage information
usage() {
    cat << EOF
Usage: $0 [options]

Options:
  -s, --server-ip     QA server IP address (default: ${DEFAULT_SERVER_IP})
  -c, --client-ip     QA client IP address (default: ${DEFAULT_CLIENT_IP})
  -u, --user          SSH user (default: ${DEFAULT_USER})
  -h, --help          Show this help message

Example:
  $0 --server-ip 192.168.50.210 --client-ip 192.168.50.211
EOF
}

# Parse command line arguments
server_ip="${DEFAULT_SERVER_IP}"
client_ip="${DEFAULT_CLIENT_IP}"
ssh_user="${DEFAULT_USER}"

while [[ $# -gt 0 ]]; do
    case $1 in
        -s|--server-ip)
            server_ip="$2"
            shift 2
            ;;
        -c|--client-ip)
            client_ip="$2"
            shift 2
            ;;
        -u|--user)
            ssh_user="$2"
            shift 2
            ;;
        -h|--help)
            usage
            exit 0
            ;;
        *)
            echo "Unknown option: $1"
            usage
            exit 1
            ;;
    esac
done

# Deploy to a single host
deploy_to_host() {
    local host=$1
    local type=$2
    local failed=0

    log_info "Deploying verification system to ${type} (${host})"

    # Create remote directories
    ssh "${ssh_user}@${host}" "sudo mkdir -p ${REMOTE_BASE_DIR}/tools/verification/{config,lib,modules,reports}"

    # Copy verification system
    rsync -az --rsync-path="sudo rsync" \
        "${SCRIPT_DIR}/unified_verifier.sh" \
        "${SCRIPT_DIR}/lib/" \
        "${SCRIPT_DIR}/modules/" \
        "${SCRIPT_DIR}/config/" \
        "${ssh_user}@${host}:${REMOTE_BASE_DIR}/tools/verification/"

    # Set permissions
    ssh "${ssh_user}@${host}" "sudo chown -R root:root ${REMOTE_BASE_DIR}/tools/verification/ && \
        sudo chmod -R 755 ${REMOTE_BASE_DIR}/tools/verification/ && \
        sudo chmod 644 ${REMOTE_BASE_DIR}/tools/verification/config/*.yaml"

    # Create symlink in /usr/local/bin
    ssh "${ssh_user}@${host}" "sudo ln -sf ${REMOTE_BASE_DIR}/tools/verification/unified_verifier.sh /usr/local/bin/verify-environment"

    # Run initial verification
    log_info "Running initial verification on ${type}"
    if ! ssh "${ssh_user}@${host}" "sudo verify-environment --debug"; then
        log_error "Initial verification failed on ${type}"
        failed=1
    fi

    return ${failed}
}

# Main deployment function
main() {
    local failed=0

    # Verify SSH access
    for host in "${server_ip}" "${client_ip}"; do
        if ! ssh -q "${ssh_user}@${host}" exit; then
            log_error "Cannot SSH to ${host}"
            exit 1
        fi
    done

    # Deploy to server
    if ! deploy_to_host "${server_ip}" "server"; then
        log_error "Failed to deploy to server"
        failed=1
    fi

    # Deploy to client
    if ! deploy_to_host "${client_ip}" "client"; then
        log_error "Failed to deploy to client"
        failed=1
    fi

    if [[ ${failed} -eq 0 ]]; then
        log_info "Verification system deployed successfully"
        cat << EOF

Verification system is now available on both hosts:
- Server (${server_ip}): verify-environment
- Client (${client_ip}): verify-environment

Usage examples:
  verify-environment                    # Run all verifications
  verify-environment --modules system   # Run only system verification
  verify-environment --skip performance # Skip performance verification
  verify-environment --debug           # Enable debug output

Reports are stored in:
  ${REMOTE_BASE_DIR}/tools/verification/reports/
EOF
    else
        log_error "Deployment failed"
        exit 1
    fi
}

# Run main function
main "$@"
