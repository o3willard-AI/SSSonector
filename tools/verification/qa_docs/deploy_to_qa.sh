#!/bin/bash
# Script to deploy the enhanced minimal functionality test script to the QA environment

# Set strict error handling
set -e
set -o pipefail

# Configuration
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
ENHANCED_SCRIPT="$SCRIPT_DIR/enhanced_minimal_functionality_test.sh"
LOG_FILE="/tmp/deploy_to_qa.log"

# Extract QA server and client from enhanced script
QA_SERVER=$(grep -E "^QA_SERVER=" "$ENHANCED_SCRIPT" | cut -d'"' -f2)
QA_CLIENT=$(grep -E "^QA_CLIENT=" "$ENHANCED_SCRIPT" | cut -d'"' -f2)
QA_USER=$(grep -E "^QA_USER=" "$ENHANCED_SCRIPT" | cut -d'"' -f2)

# Function to log messages
log() {
    echo "[$(date '+%Y-%m-%d %H:%M:%S')] $1" | tee -a "$LOG_FILE"
}

# Function to check if a command exists
command_exists() {
    command -v "$1" >/dev/null 2>&1
}

# Function to check if a file exists
file_exists() {
    [ -f "$1" ]
}

# Function to check if a script is executable
is_executable() {
    [ -x "$1" ]
}

# Function to check if required tools are installed
check_required_tools() {
    log "Checking if required tools are installed..."
    
    local required_tools=("ssh" "scp")
    local missing_tools=()
    
    for tool in "${required_tools[@]}"; do
        if command_exists "$tool"; then
            log "Tool $tool is installed."
        else
            log "ERROR: Tool $tool is not installed."
            missing_tools+=("$tool")
        fi
    done
    
    if [ ${#missing_tools[@]} -gt 0 ]; then
        log "ERROR: The following tools are missing: ${missing_tools[*]}"
        return 1
    fi
    
    return 0
}

# Function to check if QA environment is accessible
check_qa_environment() {
    log "Checking if QA environment is accessible..."
    
    log "QA server: $QA_SERVER"
    log "QA client: $QA_CLIENT"
    log "QA user: $QA_USER"
    
    # Check if QA server is accessible
    log "Checking if QA server is accessible..."
    if ping -c 1 "$QA_SERVER" >/dev/null 2>&1; then
        log "QA server is accessible."
    else
        log "ERROR: QA server is not accessible."
        return 1
    fi
    
    # Check if QA client is accessible
    log "Checking if QA client is accessible..."
    if ping -c 1 "$QA_CLIENT" >/dev/null 2>&1; then
        log "QA client is accessible."
    else
        log "ERROR: QA client is not accessible."
        return 1
    fi
    
    # Check if SSH to QA server works
    log "Checking if SSH to QA server works..."
    if ssh -o BatchMode=yes -o ConnectTimeout=5 "$QA_USER@$QA_SERVER" "echo 'SSH to QA server works'" >/dev/null 2>&1; then
        log "SSH to QA server works."
    else
        log "ERROR: SSH to QA server does not work."
        return 1
    fi
    
    # Check if SSH to QA client works
    log "Checking if SSH to QA client works..."
    if ssh -o BatchMode=yes -o ConnectTimeout=5 "$QA_USER@$QA_CLIENT" "echo 'SSH to QA client works'" >/dev/null 2>&1; then
        log "SSH to QA client works."
    else
        log "ERROR: SSH to QA client does not work."
        return 1
    fi
    
    return 0
}

# Function to check if enhanced_minimal_functionality_test.sh is valid
check_enhanced_script() {
    log "Checking if enhanced_minimal_functionality_test.sh exists..."
    if file_exists "$ENHANCED_SCRIPT"; then
        log "enhanced_minimal_functionality_test.sh exists."
        
        log "Checking if enhanced_minimal_functionality_test.sh is executable..."
        if is_executable "$ENHANCED_SCRIPT"; then
            log "enhanced_minimal_functionality_test.sh is executable."
        else
            log "ERROR: enhanced_minimal_functionality_test.sh is not executable."
            return 1
        fi
    else
        log "ERROR: enhanced_minimal_functionality_test.sh does not exist."
        return 1
    fi
    
    return 0
}

# Function to deploy enhanced_minimal_functionality_test.sh to QA server
deploy_to_qa_server() {
    log "Deploying enhanced_minimal_functionality_test.sh to QA server..."
    
    # Create directory on QA server
    log "Creating directory on QA server..."
    ssh "$QA_USER@$QA_SERVER" "mkdir -p /opt/sssonector/tools/verification/qa_docs"
    
    # Copy enhanced script to QA server
    log "Copying enhanced script to QA server..."
    scp "$ENHANCED_SCRIPT" "$QA_USER@$QA_SERVER:/opt/sssonector/tools/verification/qa_docs/"
    
    # Make enhanced script executable on QA server
    log "Making enhanced script executable on QA server..."
    ssh "$QA_USER@$QA_SERVER" "chmod +x /opt/sssonector/tools/verification/qa_docs/enhanced_minimal_functionality_test.sh"
    
    log "Enhanced script deployed to QA server."
    
    return 0
}

# Function to deploy enhanced_minimal_functionality_test.sh to QA client
deploy_to_qa_client() {
    log "Deploying enhanced_minimal_functionality_test.sh to QA client..."
    
    # Create directory on QA client
    log "Creating directory on QA client..."
    ssh "$QA_USER@$QA_CLIENT" "mkdir -p /opt/sssonector/tools/verification/qa_docs"
    
    # Copy enhanced script to QA client
    log "Copying enhanced script to QA client..."
    scp "$ENHANCED_SCRIPT" "$QA_USER@$QA_CLIENT:/opt/sssonector/tools/verification/qa_docs/"
    
    # Make enhanced script executable on QA client
    log "Making enhanced script executable on QA client..."
    ssh "$QA_USER@$QA_CLIENT" "chmod +x /opt/sssonector/tools/verification/qa_docs/enhanced_minimal_functionality_test.sh"
    
    log "Enhanced script deployed to QA client."
    
    return 0
}

# Function to verify deployment
verify_deployment() {
    log "Verifying deployment..."
    
    # Verify deployment to QA server
    log "Verifying deployment to QA server..."
    if ssh "$QA_USER@$QA_SERVER" "[ -x /opt/sssonector/tools/verification/qa_docs/enhanced_minimal_functionality_test.sh ] && echo 'Deployment to QA server verified'"; then
        log "Deployment to QA server verified."
    else
        log "ERROR: Deployment to QA server could not be verified."
        return 1
    fi
    
    # Verify deployment to QA client
    log "Verifying deployment to QA client..."
    if ssh "$QA_USER@$QA_CLIENT" "[ -x /opt/sssonector/tools/verification/qa_docs/enhanced_minimal_functionality_test.sh ] && echo 'Deployment to QA client verified'"; then
        log "Deployment to QA client verified."
    else
        log "ERROR: Deployment to QA client could not be verified."
        return 1
    fi
    
    log "Deployment verified."
    
    return 0
}

# Main function
main() {
    log "Starting deployment of enhanced minimal functionality test script to QA environment..."
    
    # Check if required tools are installed
    if ! check_required_tools; then
        log "ERROR: Required tools are missing. Please install them before continuing."
        exit 1
    fi
    
    # Check if enhanced_minimal_functionality_test.sh is valid
    if ! check_enhanced_script; then
        log "ERROR: enhanced_minimal_functionality_test.sh is not valid. Please fix it before continuing."
        exit 1
    fi
    
    # Check if QA environment is accessible
    if ! check_qa_environment; then
        log "ERROR: QA environment is not accessible. Please fix the issues before continuing."
        exit 1
    fi
    
    # Deploy enhanced_minimal_functionality_test.sh to QA server
    if ! deploy_to_qa_server; then
        log "ERROR: Failed to deploy enhanced_minimal_functionality_test.sh to QA server."
        exit 1
    fi
    
    # Deploy enhanced_minimal_functionality_test.sh to QA client
    if ! deploy_to_qa_client; then
        log "ERROR: Failed to deploy enhanced_minimal_functionality_test.sh to QA client."
        exit 1
    fi
    
    # Verify deployment
    if ! verify_deployment; then
        log "ERROR: Failed to verify deployment."
        exit 1
    fi
    
    log "Deployment of enhanced minimal functionality test script to QA environment completed successfully."
    log "To run the enhanced minimal functionality test script on the QA server, use:"
    log "  ssh $QA_USER@$QA_SERVER 'cd /opt/sssonector/tools/verification/qa_docs && ./enhanced_minimal_functionality_test.sh'"
    
    return 0
}

# Run the main function
main
