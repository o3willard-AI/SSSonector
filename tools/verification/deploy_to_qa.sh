#!/bin/bash

# deploy_to_qa.sh
# Script to deploy the verification system to QA environment
set -euo pipefail

# QA environment details
QA_SERVER="192.168.50.210"
QA_CLIENT="192.168.50.211"
QA_USER="sblanken"
QA_SUDO_PASSWORD="101abn"

# Color codes for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m'

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

# Main function
main() {
    log_info "Deploying verification system to QA environment"
    
    # Check if deploy.sh exists and is executable
    if [[ ! -x "./deploy.sh" ]]; then
        log_error "deploy.sh not found or not executable"
        exit 1
    fi
    
    # Check if sshpass is installed
    if ! command -v sshpass &> /dev/null; then
        log_info "Installing sshpass..."
        sudo apt-get update && sudo apt-get install -y sshpass
    fi
    
    # Test SSH connection to QA servers
    log_info "Testing SSH connection to QA servers..."
    
    if ! sshpass -p "${QA_SUDO_PASSWORD}" ssh -o StrictHostKeyChecking=no "${QA_USER}@${QA_SERVER}" "echo 'SSH connection test successful'"; then
        log_error "Cannot SSH to server ${QA_SERVER}"
        exit 1
    fi
    
    if ! sshpass -p "${QA_SUDO_PASSWORD}" ssh -o StrictHostKeyChecking=no "${QA_USER}@${QA_CLIENT}" "echo 'SSH connection test successful'"; then
        log_error "Cannot SSH to client ${QA_CLIENT}"
        exit 1
    fi
    
    # Deploy verification system to QA environment
    log_info "Deploying verification system to QA servers..."
    
    # Use the deploy.sh script with QA environment details
    ./deploy.sh --server-ip "${QA_SERVER}" --client-ip "${QA_CLIENT}" --user "${QA_USER}"
    
    log_info "Verification system deployed successfully to QA environment"
    
    # Run initial verification on QA servers
    log_info "Running initial verification on QA server..."
    sshpass -p "${QA_SUDO_PASSWORD}" ssh -o StrictHostKeyChecking=no "${QA_USER}@${QA_SERVER}" "sudo verify-environment --modules system,network"
    
    log_info "Running initial verification on QA client..."
    sshpass -p "${QA_SUDO_PASSWORD}" ssh -o StrictHostKeyChecking=no "${QA_USER}@${QA_CLIENT}" "sudo verify-environment --modules system,network"
    
    log_info "Deployment and verification completed successfully"
    
    # Display usage instructions
    cat << EOF

Verification system is now available on both QA hosts:
- Server (${QA_SERVER}): verify-environment
- Client (${QA_CLIENT}): verify-environment

Usage examples:
  verify-environment                    # Run all verifications
  verify-environment --modules system   # Run only system verification
  verify-environment --skip performance # Skip performance verification
  verify-environment --debug           # Enable debug output

Reports are stored in:
  /opt/sssonector/tools/verification/reports/
EOF
}

# Run main function
main "$@"
