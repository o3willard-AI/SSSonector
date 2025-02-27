#!/bin/bash

# setup_qa_environment.sh
# Script to set up the QA environment for SSSonector testing
set -euo pipefail

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

# Function to make scripts executable
make_scripts_executable() {
    log_step "Making scripts executable"
    
    # List of scripts to make executable
    local scripts=(
        "enhanced_qa_testing.sh"
        "fix_transfer_logic.sh"
    )
    
    # Make each script executable
    for script in "${scripts[@]}"; do
        if [ -f "${script}" ]; then
            log_info "Making ${script} executable"
            chmod +x "${script}"
        else
            log_warn "Script ${script} not found"
        fi
    done
    
    log_info "Scripts made executable successfully"
    return 0
}

# Function to create QA environment configuration
create_qa_environment_conf() {
    log_step "Creating QA environment configuration"
    
    # Check if configuration file already exists
    if [ -f "qa_environment.conf" ]; then
        log_info "QA environment configuration already exists"
        return 0
    fi
    
    # Create configuration file
    log_info "Creating QA environment configuration file"
    cat > "qa_environment.conf" << EOF
# SSSonector QA Environment Configuration
# Edit this file to match your environment

# QA environment details
QA_SERVER="192.168.50.210"
QA_CLIENT="192.168.50.211"
QA_USER="sblanken"
QA_SUDO_PASSWORD=""

# Test settings
TEST_TIMEOUT=300  # Timeout in seconds
PACKET_COUNT=20   # Number of packets to send in each direction
RETRY_COUNT=3     # Number of retries for failed tests
EOF
    
    log_info "QA environment configuration created successfully"
    log_warn "Please edit qa_environment.conf to set QA_SUDO_PASSWORD"
    return 0
}

# Function to check if Go is installed
check_go() {
    log_step "Checking if Go is installed"
    
    # Check if go command is available
    if ! command -v go &> /dev/null; then
        log_error "Go is not installed"
        log_info "Please install Go from https://golang.org/dl/"
        return 1
    fi
    
    # Check Go version
    local go_version
    go_version=$(go version | awk '{print $3}')
    log_info "Go version: ${go_version}"
    
    log_info "Go is installed"
    return 0
}

# Function to check if sshpass is installed
check_sshpass() {
    log_step "Checking if sshpass is installed"
    
    # Check if sshpass command is available
    if ! command -v sshpass &> /dev/null; then
        log_warn "sshpass is not installed"
        
        # Try to install sshpass
        log_info "Attempting to install sshpass"
        if command -v apt-get &> /dev/null; then
            sudo apt-get update && sudo apt-get install -y sshpass
        elif command -v yum &> /dev/null; then
            sudo yum install -y sshpass
        elif command -v dnf &> /dev/null; then
            sudo dnf install -y sshpass
        elif command -v brew &> /dev/null; then
            brew install sshpass
        else
            log_error "Could not install sshpass, please install it manually"
            return 1
        fi
        
        # Check if installation was successful
        if ! command -v sshpass &> /dev/null; then
            log_error "Failed to install sshpass"
            return 1
        fi
    fi
    
    log_info "sshpass is installed"
    return 0
}

# Function to check if openssl is installed
check_openssl() {
    log_step "Checking if openssl is installed"
    
    # Check if openssl command is available
    if ! command -v openssl &> /dev/null; then
        log_error "openssl is not installed"
        log_info "Please install openssl"
        return 1
    fi
    
    # Check openssl version
    local openssl_version
    openssl_version=$(openssl version | awk '{print $2}')
    log_info "OpenSSL version: ${openssl_version}"
    
    log_info "openssl is installed"
    return 0
}

# Function to check if SSSonector binary exists
check_sssonector_binary() {
    log_step "Checking if SSSonector binary exists"
    
    # Check if SSSonector binary exists
    if [ ! -f "../../../sssonector" ]; then
        log_warn "SSSonector binary not found"
        
        # Ask if user wants to build SSSonector
        read -p "Do you want to build SSSonector? (y/n) " -n 1 -r
        echo
        if [[ $REPLY =~ ^[Yy]$ ]]; then
            log_info "Building SSSonector"
            (cd ../../.. && go build -o sssonector cmd/sssonector/main.go)
            
            # Check if build was successful
            if [ ! -f "../../../sssonector" ]; then
                log_error "Failed to build SSSonector"
                return 1
            fi
        else
            log_error "SSSonector binary not found and not built"
            return 1
        fi
    fi
    
    # Check if SSSonector binary is executable
    if [ ! -x "../../../sssonector" ]; then
        log_warn "SSSonector binary is not executable"
        
        # Make SSSonector binary executable
        log_info "Making SSSonector binary executable"
        chmod +x "../../../sssonector"
        
        # Check if chmod was successful
        if [ ! -x "../../../sssonector" ]; then
            log_error "Failed to make SSSonector binary executable"
            return 1
        fi
    fi
    
    log_info "SSSonector binary exists and is executable"
    return 0
}

# Main function
main() {
    log_step "Setting up QA environment for SSSonector testing"
    
    # Make scripts executable
    make_scripts_executable || {
        log_error "Failed to make scripts executable"
        exit 1
    }
    
    # Create QA environment configuration
    create_qa_environment_conf || {
        log_error "Failed to create QA environment configuration"
        exit 1
    }
    
    # Check if Go is installed
    check_go || {
        log_error "Go is not installed"
        exit 1
    }
    
    # Check if sshpass is installed
    check_sshpass || {
        log_error "sshpass is not installed"
        exit 1
    }
    
    # Check if openssl is installed
    check_openssl || {
        log_error "openssl is not installed"
        exit 1
    }
    
    # Check if SSSonector binary exists
    check_sssonector_binary || {
        log_error "SSSonector binary not found or not executable"
        exit 1
    }
    
    log_step "QA environment setup completed successfully"
    log_info "Please edit qa_environment.conf to set QA_SUDO_PASSWORD"
    log_info "Then run ./fix_transfer_logic.sh to fix the transfer logic"
    log_info "Finally, run ./enhanced_qa_testing.sh to run the QA tests"
    
    exit 0
}

# Run main function
main "$@"
