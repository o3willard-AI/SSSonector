#!/bin/bash

# SSSonector Backup Script
# This script creates a complete backup of the SSSonector project state

# Exit on any error
set -e

# Configuration
BACKUP_ROOT="/home/sblanken/Desktop/go/src/github.com/o3willard-AI/SSSonector/docs/disaster_recovery/backups"
SOURCE_ROOT="/home/sblanken/Desktop/go/src/github.com/o3willard-AI/SSSonector"
TIMESTAMP=$(date +%Y%m%d_%H%M%S)
BACKUP_DIR="${BACKUP_ROOT}/${TIMESTAMP}"

# Color output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m'

# Logging function
log() {
    echo -e "${GREEN}[$(date +'%Y-%m-%d %H:%M:%S')] $1${NC}"
}

error() {
    echo -e "${RED}[ERROR] $1${NC}" >&2
}

warn() {
    echo -e "${YELLOW}[WARNING] $1${NC}"
}

# Create backup directories
create_dirs() {
    log "Creating backup directories..."
    mkdir -p "${BACKUP_DIR}/"{source,configs,tests,qa_data}
}

# Backup source code
backup_source() {
    log "Backing up source code..."
    rsync -av --exclude '.git' \
              --exclude 'docs/disaster_recovery/backups' \
              "${SOURCE_ROOT}/" "${BACKUP_DIR}/source/"
}

# Backup configurations
backup_configs() {
    log "Backing up configurations..."
    
    # Project configs
    cp -r "${SOURCE_ROOT}/configs" "${BACKUP_DIR}/configs/"
    
    # SNMP configs
    if [ -f "/etc/snmp/metrics_range.conf" ]; then
        cp "/etc/snmp/metrics_range.conf" "${BACKUP_DIR}/configs/"
    else
        warn "SNMP metrics configuration not found"
    fi
}

# Backup test data
backup_tests() {
    log "Backing up test data..."
    
    # Test certificates
    if [ -d "/home/test/certs" ]; then
        cp -r "/home/test/certs" "${BACKUP_DIR}/tests/"
    else
        warn "Test certificates directory not found"
    fi
    
    # Test scripts
    cp -r "${SOURCE_ROOT}/test" "${BACKUP_DIR}/tests/"
}

# Backup QA environment data
backup_qa() {
    log "Backing up QA environment data..."
    
    # Test file
    if [ -f "/home/test/data/DryFire_v4_10.zip" ]; then
        cp "/home/test/data/DryFire_v4_10.zip" "${BACKUP_DIR}/qa_data/"
    else
        warn "Test file DryFire_v4_10.zip not found"
    fi
    
    # Environment status
    if [ -f "${SOURCE_ROOT}/qa_environment.json" ]; then
        cp "${SOURCE_ROOT}/qa_environment.json" "${BACKUP_DIR}/qa_data/"
    fi
}

# Create backup manifest
create_manifest() {
    log "Creating backup manifest..."
    
    cat > "${BACKUP_DIR}/manifest.txt" << EOF
SSSonector Backup Manifest
Created: $(date)

Source Code Hash: $(cd "${BACKUP_DIR}/source" && find . -type f -exec sha256sum {} \; | sort | sha256sum)
Configuration Hash: $(cd "${BACKUP_DIR}/configs" && find . -type f -exec sha256sum {} \; | sort | sha256sum)
Test Data Hash: $(cd "${BACKUP_DIR}/tests" && find . -type f -exec sha256sum {} \; | sort | sha256sum)
QA Data Hash: $(cd "${BACKUP_DIR}/qa_data" && find . -type f -exec sha256sum {} \; | sort | sha256sum)

Environment Information:
Go Version: $(go version)
OS: $(uname -a)
Timestamp: ${TIMESTAMP}
EOF
}

# Verify backup
verify_backup() {
    log "Verifying backup..."
    
    # Check source code
    if [ ! -f "${BACKUP_DIR}/source/go.mod" ]; then
        error "Source code backup verification failed"
        return 1
    fi
    
    # Check configs
    if [ ! -d "${BACKUP_DIR}/configs" ]; then
        error "Configuration backup verification failed"
        return 1
    }
    
    # Check test data
    if [ ! -d "${BACKUP_DIR}/tests" ]; then
        error "Test data backup verification failed"
        return 1
    }
    
    log "Backup verification completed successfully"
}

# Cleanup old backups (keep last 5)
cleanup_old_backups() {
    log "Cleaning up old backups..."
    cd "${BACKUP_ROOT}"
    ls -t | tail -n +6 | xargs -r rm -rf
}

# Main backup process
main() {
    log "Starting SSSonector backup process..."
    
    create_dirs
    backup_source
    backup_configs
    backup_tests
    backup_qa
    create_manifest
    verify_backup
    cleanup_old_backups
    
    log "Backup completed successfully at ${BACKUP_DIR}"
}

# Run main process
main

# Exit with success
exit 0
