#!/bin/bash

# run_qa_tests.sh
# Script to automate the entire QA testing process for SSSonector
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
BLUE='\033[0;34m'
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

log_step() {
    echo -e "${BLUE}[STEP]${NC} $1"
}

# Run command on remote host
run_remote() {
    local host=$1
    local command=$2
    
    sshpass -p "${QA_SUDO_PASSWORD}" ssh -o StrictHostKeyChecking=no "${QA_USER}@${host}" "${command}"
}

# Main function
main() {
    log_step "Starting SSSonector QA Testing Process"
    
    # Check if sshpass is installed
    if ! command -v sshpass &> /dev/null; then
        log_info "Installing sshpass..."
        sudo apt-get update && sudo apt-get install -y sshpass
    fi
    
    # Test SSH connection to QA servers
    log_info "Testing SSH connection to QA servers"
    
    if ! sshpass -p "${QA_SUDO_PASSWORD}" ssh -o StrictHostKeyChecking=no "${QA_USER}@${QA_SERVER}" "echo 'SSH connection test successful'" &> /dev/null; then
        log_error "Cannot SSH to server ${QA_SERVER}"
        exit 1
    fi
    
    if ! sshpass -p "${QA_SUDO_PASSWORD}" ssh -o StrictHostKeyChecking=no "${QA_USER}@${QA_CLIENT}" "echo 'SSH connection test successful'" &> /dev/null; then
        log_error "Cannot SSH to client ${QA_CLIENT}"
        exit 1
    fi
    
    # Step 1: Environment Preparation
    log_step "1. Environment Preparation"
    
    # Enable IP forwarding on server
    log_info "Enabling IP forwarding on server"
    run_remote "${QA_SERVER}" "echo 1 | sudo tee /proc/sys/net/ipv4/ip_forward"
    
    # Enable IP forwarding on client
    log_info "Enabling IP forwarding on client"
    run_remote "${QA_CLIENT}" "echo 1 | sudo tee /proc/sys/net/ipv4/ip_forward"
    
    # Step 2: Clean Up the QA Environment
    log_step "2. Cleaning Up QA Environment"
    
    # Run cleanup script
    log_info "Running cleanup script"
    ./cleanup_qa.sh
    
    # Step 3: Deploy SSSonector to the QA Environment
    log_step "3. Deploying SSSonector to QA Environment"
    
    # Run deployment script
    log_info "Running deployment script"
    ./deploy_sssonector.sh
    
    # Step 4: Run Sanity Checks
    log_step "4. Running Sanity Checks"
    
    # Run sanity checks script
    log_info "Running sanity checks script"
    ./run_sanity_checks.sh
    
    log_step "SSSonector QA Testing Process Completed"
    
    # Generate test report
    log_info "Generating test report"
    
    # Get current date and time
    current_date=$(date +"%Y-%m-%d")
    current_time=$(date +"%H:%M:%S")
    
    # Create test report
    cat > qa_test_report_${current_date}.md << EOF
# SSSonector QA Test Report

## Test Information

- **Date**: ${current_date}
- **Time**: ${current_time}
- **Server**: ${QA_SERVER}
- **Client**: ${QA_CLIENT}
- **User**: ${QA_USER}

## Test Results

The QA testing process has been completed. Please review the logs for detailed results.

### Environment Preparation

- IP forwarding enabled on server
- IP forwarding enabled on client

### QA Environment Cleanup

- SSSonector binaries removed
- Certificates cleaned up
- Configuration files cleaned up
- SSSonector processes killed
- TUN interfaces removed
- Required ports freed up

### SSSonector Deployment

- Certificates generated
- Configuration files created
- SSSonector binary deployed
- Permissions set

### Sanity Checks

- Scenario 1: Client Foreground / Server Foreground
- Scenario 2: Client Background / Server Foreground
- Scenario 3: Client Background / Server Background

## Next Steps

1. Review the logs for any errors or warnings
2. Address any issues identified during testing
3. Re-run the tests if necessary

EOF
    
    log_info "Test report generated: qa_test_report_${current_date}.md"
}

# Run main function
main "$@"
