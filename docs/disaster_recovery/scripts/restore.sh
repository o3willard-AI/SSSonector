#!/bin/bash

# SSSonector Restore Script
# This script restores a SSSonector backup to a working state

# Exit on any error
set -e

# Configuration
BACKUP_ROOT="/home/sblanken/Desktop/go/src/github.com/o3willard-AI/SSSonector/docs/disaster_recovery/backups"
PROJECT_ROOT="/home/sblanken/Desktop/go/src/github.com/o3willard-AI/SSSonector"

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

# Check prerequisites
check_prerequisites() {
    log "Checking prerequisites..."
    
    # Check Go installation
    if ! command -v go &> /dev/null; then
        error "Go is not installed"
        exit 1
    fi
    
    # Check required packages
    for pkg in rsync virtualbox snmpd ntopng; do
        if ! command -v $pkg &> /dev/null; then
            warn "$pkg is not installed"
        fi
    done
    
    # Check TUN module
    if ! lsmod | grep -q "^tun"; then
        warn "TUN module is not loaded"
    fi
}

# Get latest backup or specific backup by timestamp
get_backup_path() {
    local timestamp=$1
    local backup_path
    
    if [ -z "$timestamp" ]; then
        backup_path=$(ls -td ${BACKUP_ROOT}/*/ | head -1)
    else
        backup_path="${BACKUP_ROOT}/${timestamp}"
    fi
    
    if [ ! -d "$backup_path" ]; then
        error "Backup not found: $backup_path"
        exit 1
    fi
    
    echo "$backup_path"
}

# Verify backup integrity
verify_backup() {
    local backup_path=$1
    log "Verifying backup integrity at $backup_path..."
    
    # Check manifest
    if [ ! -f "${backup_path}/manifest.txt" ]; then
        error "Backup manifest not found"
        exit 1
    fi
    
    # Verify directory structure
    for dir in source configs tests qa_data; do
        if [ ! -d "${backup_path}/${dir}" ]; then
            error "Backup directory '${dir}' not found"
            exit 1
        fi
    done
    
    # Verify source code
    if [ ! -f "${backup_path}/source/go.mod" ]; then
        error "Source code verification failed"
        exit 1
    }
}

# Restore source code
restore_source() {
    local backup_path=$1
    log "Restoring source code..."
    
    # Create project directory if it doesn't exist
    mkdir -p "${PROJECT_ROOT}"
    
    # Sync source code
    rsync -av --delete "${backup_path}/source/" "${PROJECT_ROOT}/"
    
    # Rebuild project
    cd "${PROJECT_ROOT}"
    go mod download
    go mod tidy
    make build
}

# Restore configurations
restore_configs() {
    local backup_path=$1
    log "Restoring configurations..."
    
    # Project configs
    cp -r "${backup_path}/configs/"* "${PROJECT_ROOT}/configs/"
    
    # SNMP configs
    if [ -f "${backup_path}/configs/metrics_range.conf" ]; then
        sudo cp "${backup_path}/configs/metrics_range.conf" "/etc/snmp/"
    fi
}

# Restore test data
restore_tests() {
    local backup_path=$1
    log "Restoring test data..."
    
    # Test certificates
    if [ -d "${backup_path}/tests/certs" ]; then
        mkdir -p "/home/test/certs"
        cp -r "${backup_path}/tests/certs/"* "/home/test/certs/"
    fi
    
    # Test scripts
    cp -r "${backup_path}/tests/"* "${PROJECT_ROOT}/test/"
}

# Restore QA environment
restore_qa() {
    local backup_path=$1
    log "Restoring QA environment..."
    
    # Create data directory
    mkdir -p "/home/test/data"
    
    # Restore test file
    if [ -f "${backup_path}/qa_data/DryFire_v4_10.zip" ]; then
        cp "${backup_path}/qa_data/DryFire_v4_10.zip" "/home/test/data/"
    fi
    
    # Restore environment status
    if [ -f "${backup_path}/qa_data/qa_environment.json" ]; then
        cp "${backup_path}/qa_data/qa_environment.json" "${PROJECT_ROOT}/"
    fi
}

# Verify restoration
verify_restoration() {
    log "Verifying restoration..."
    
    # Check project build
    cd "${PROJECT_ROOT}"
    if ! make build; then
        error "Project build verification failed"
        return 1
    fi
    
    # Run basic tests
    if [ -f "test/run_cert_tests.sh" ]; then
        if ! ./test/run_cert_tests.sh; then
            warn "Certificate tests failed"
        fi
    fi
    
    log "Restoration verification completed"
}

# Display restoration summary
show_summary() {
    local backup_path=$1
    log "Restoration Summary"
    echo "----------------------------------------"
    echo "Restored from: ${backup_path}"
    echo "Project location: ${PROJECT_ROOT}"
    echo "Build status: $(if make -n build &>/dev/null; then echo "Success"; else echo "Failed"; fi)"
    echo "Test status: $(if [ -f "test/run_cert_tests.sh" ]; then echo "Available"; else echo "Missing"; fi)"
    echo "----------------------------------------"
}

# Main restore process
main() {
    local timestamp=$1
    local backup_path
    
    log "Starting SSSonector restore process..."
    
    check_prerequisites
    backup_path=$(get_backup_path "$timestamp")
    verify_backup "$backup_path"
    
    # Perform restoration
    restore_source "$backup_path"
    restore_configs "$backup_path"
    restore_tests "$backup_path"
    restore_qa "$backup_path"
    
    verify_restoration
    show_summary "$backup_path"
    
    log "Restore completed successfully"
}

# Parse command line arguments
if [ $# -eq 1 ]; then
    main "$1"
else
    main
fi

# Exit with success
exit 0
