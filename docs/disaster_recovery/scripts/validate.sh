#!/bin/bash

# SSSonector Validation Script
# This script validates the environment and backup/restore functionality

# Exit on any error
set -e

# Configuration
PROJECT_ROOT="/home/sblanken/Desktop/go/src/github.com/o3willard-AI/SSSonector"
SCRIPTS_DIR="${PROJECT_ROOT}/docs/disaster_recovery/scripts"
TEST_DIR="/tmp/sssonector_test"

# Color output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m'

# Logging functions
log() {
    echo -e "${GREEN}[$(date +'%Y-%m-%d %H:%M:%S')] $1${NC}"
}

error() {
    echo -e "${RED}[ERROR] $1${NC}" >&2
}

warn() {
    echo -e "${YELLOW}[WARNING] $1${NC}"
}

# Validate environment
validate_environment() {
    log "Validating environment..."
    
    # Check Go version
    GO_VERSION=$(go version | awk '{print $3}' | sed 's/go//')
    if [[ "${GO_VERSION}" < "1.21" ]]; then
        error "Go version must be 1.21 or higher (found ${GO_VERSION})"
        return 1
    fi
    
    # Check system packages
    REQUIRED_PACKAGES="rsync virtualbox snmpd ntopng"
    MISSING_PACKAGES=""
    for pkg in $REQUIRED_PACKAGES; do
        if ! command -v $pkg &> /dev/null; then
            MISSING_PACKAGES="${MISSING_PACKAGES} ${pkg}"
        fi
    done
    if [ ! -z "$MISSING_PACKAGES" ]; then
        warn "Missing packages:${MISSING_PACKAGES}"
    fi
    
    # Check TUN module
    if ! lsmod | grep -q "^tun"; then
        warn "TUN module not loaded"
    fi
    
    # Check project structure
    for dir in cmd internal test docs configs; do
        if [ ! -d "${PROJECT_ROOT}/${dir}" ]; then
            error "Required directory '${dir}' not found"
            return 1
        fi
    done
    
    log "Environment validation completed"
}

# Validate backup functionality
validate_backup() {
    log "Validating backup functionality..."
    
    # Run backup script
    if [ ! -x "${SCRIPTS_DIR}/backup.sh" ]; then
        error "Backup script not executable"
        return 1
    fi
    
    "${SCRIPTS_DIR}/backup.sh"
    if [ $? -ne 0 ]; then
        error "Backup script failed"
        return 1
    fi
    
    # Check backup contents
    LATEST_BACKUP=$(ls -td ${PROJECT_ROOT}/docs/disaster_recovery/backups/*/ | head -1)
    if [ ! -d "${LATEST_BACKUP}" ]; then
        error "Backup directory not found"
        return 1
    fi
    
    # Verify backup structure
    for dir in source configs tests qa_data; do
        if [ ! -d "${LATEST_BACKUP}/${dir}" ]; then
            error "Backup directory '${dir}' not found"
            return 1
        fi
    done
    
    log "Backup validation completed"
}

# Validate restore functionality
validate_restore() {
    log "Validating restore functionality..."
    
    # Create test directory
    rm -rf "${TEST_DIR}"
    mkdir -p "${TEST_DIR}"
    
    # Run restore script to test directory
    if [ ! -x "${SCRIPTS_DIR}/restore.sh" ]; then
        error "Restore script not executable"
        return 1
    fi
    
    # Temporarily modify PROJECT_ROOT in restore script
    RESTORE_SCRIPT_TEMP="${TEST_DIR}/restore.sh"
    cp "${SCRIPTS_DIR}/restore.sh" "${RESTORE_SCRIPT_TEMP}"
    sed -i "s|PROJECT_ROOT=.*|PROJECT_ROOT=\"${TEST_DIR}\"|" "${RESTORE_SCRIPT_TEMP}"
    
    # Run restore
    "${RESTORE_SCRIPT_TEMP}"
    if [ $? -ne 0 ]; then
        error "Restore script failed"
        return 1
    fi
    
    # Verify restored contents
    for dir in cmd internal test docs configs; do
        if [ ! -d "${TEST_DIR}/${dir}" ]; then
            error "Restored directory '${dir}' not found"
            return 1
        fi
    done
    
    # Clean up test directory
    rm -rf "${TEST_DIR}"
    
    log "Restore validation completed"
}

# Validate project build
validate_build() {
    log "Validating project build..."
    
    cd "${PROJECT_ROOT}"
    
    # Check dependencies
    go mod download
    if [ $? -ne 0 ]; then
        error "Failed to download dependencies"
        return 1
    fi
    
    # Build project
    make build
    if [ $? -ne 0 ]; then
        error "Project build failed"
        return 1
    }
    
    # Run basic tests
    if [ -f "test/run_cert_tests.sh" ]; then
        ./test/run_cert_tests.sh
        if [ $? -ne 0 ]; then
            warn "Certificate tests failed"
        fi
    fi
    
    log "Build validation completed"
}

# Validate QA environment
validate_qa() {
    log "Validating QA environment..."
    
    # Check VM connectivity
    for vm in "192.168.50.210" "192.168.50.211" "192.168.50.212"; do
        if ! ping -c 1 $vm &> /dev/null; then
            warn "VM $vm not reachable"
        fi
    done
    
    # Check SNMP service
    if nc -zv 192.168.50.212 161 2>/dev/null; then
        log "SNMP service is running"
    else
        warn "SNMP service not available"
    fi
    
    # Check ntopng service
    if nc -zv 192.168.50.212 3000 2>/dev/null; then
        log "ntopng service is running"
    else
        warn "ntopng service not available"
    fi
    
    log "QA environment validation completed"
}

# Show validation summary
show_summary() {
    log "Validation Summary"
    echo "----------------------------------------"
    echo "Environment: $(if validate_environment &>/dev/null; then echo "✓"; else echo "✗"; fi)"
    echo "Backup: $(if validate_backup &>/dev/null; then echo "✓"; else echo "✗"; fi)"
    echo "Restore: $(if validate_restore &>/dev/null; then echo "✓"; else echo "✗"; fi)"
    echo "Build: $(if validate_build &>/dev/null; then echo "✓"; else echo "✗"; fi)"
    echo "QA Environment: $(if validate_qa &>/dev/null; then echo "✓"; else echo "✗"; fi)"
    echo "----------------------------------------"
}

# Main validation process
main() {
    log "Starting SSSonector validation process..."
    
    validate_environment
    validate_backup
    validate_restore
    validate_build
    validate_qa
    show_summary
    
    log "Validation completed successfully"
}

# Run main process
main

# Exit with success
exit 0
