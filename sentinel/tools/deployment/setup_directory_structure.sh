#!/bin/bash

# setup_directory_structure.sh
# Part of Project SENTINEL - Directory Structure Setup Tool
# Version: 1.0.0

set -euo pipefail

# Color codes for output
GREEN='\033[0;32m'
RED='\033[0;31m'
YELLOW='\033[1;33m'
NC='\033[0m'

# Logging functions
log_info() {
    echo -e "${GREEN}[INFO]${NC} $1"
}

log_error() {
    echo -e "${RED}[ERROR]${NC} $1"
    return 1
}

log_warn() {
    echo -e "${YELLOW}[WARN]${NC} $1"
}

# Default paths
DEFAULT_BASE_DIR="/opt/sssonector"
SSSONECTOR_DIRS=(
    "bin"      # Binary executables
    "config"   # Configuration files
    "certs"    # Certificates and keys
    "log"      # Log files
    "state"    # Runtime state
    "tools"    # Utility scripts
)

# Permission settings
DIR_PERMS=755
CONFIG_PERMS=644
CERT_PERMS=644
KEY_PERMS=600
BIN_PERMS=755
LOG_PERMS=644

# Create directory structure
create_directories() {
    local base_dir=$1
    log_info "Creating directory structure in ${base_dir}..."

    for dir in "${SSSONECTOR_DIRS[@]}"; do
        if [[ ! -d "${base_dir}/${dir}" ]]; then
            sudo mkdir -p "${base_dir}/${dir}"
            log_info "Created directory: ${base_dir}/${dir}"
        else
            log_warn "Directory already exists: ${base_dir}/${dir}"
        fi
    done
}

# Set directory permissions
set_permissions() {
    local base_dir=$1
    log_info "Setting directory permissions..."

    # Set base directory permissions
    sudo chmod ${DIR_PERMS} "${base_dir}"

    # Set subdirectory permissions
    for dir in "${SSSONECTOR_DIRS[@]}"; do
        sudo chmod ${DIR_PERMS} "${base_dir}/${dir}"
    done

    # Set specific file permissions if files exist
    if [[ -d "${base_dir}/config" ]]; then
        sudo find "${base_dir}/config" -type f -name "*.yaml" -exec chmod ${CONFIG_PERMS} {} \;
    fi

    if [[ -d "${base_dir}/certs" ]]; then
        sudo find "${base_dir}/certs" -type f -name "*.crt" -exec chmod ${CERT_PERMS} {} \;
        sudo find "${base_dir}/certs" -type f -name "*.key" -exec chmod ${KEY_PERMS} {} \;
    fi

    if [[ -d "${base_dir}/bin" ]]; then
        sudo find "${base_dir}/bin" -type f -exec chmod ${BIN_PERMS} {} \;
    fi

    if [[ -d "${base_dir}/log" ]]; then
        sudo find "${base_dir}/log" -type f -exec chmod ${LOG_PERMS} {} \;
    fi
}

# Set ownership
set_ownership() {
    local base_dir=$1
    local owner=$2
    log_info "Setting ownership to ${owner}..."

    sudo chown -R "${owner}:${owner}" "${base_dir}"
}

# Verify directory structure
verify_structure() {
    local base_dir=$1
    local failed=0
    log_info "Verifying directory structure..."

    # Check directories exist
    for dir in "${SSSONECTOR_DIRS[@]}"; do
        if [[ ! -d "${base_dir}/${dir}" ]]; then
            log_error "Missing directory: ${base_dir}/${dir}"
            failed=1
        fi
    done

    # Check permissions
    local dir_perms
    for dir in "${SSSONECTOR_DIRS[@]}"; do
        dir_perms=$(stat -c "%a" "${base_dir}/${dir}")
        if [[ "${dir_perms}" != "${DIR_PERMS}" ]]; then
            log_error "Incorrect permissions on ${base_dir}/${dir}: ${dir_perms} (expected ${DIR_PERMS})"
            failed=1
        fi
    done

    if [[ ${failed} -eq 1 ]]; then
        return 1
    fi

    log_info "Directory structure verified successfully"
}

# Print usage
usage() {
    echo "Usage: $0 [-d base_directory] [-o owner]"
    echo
    echo "Options:"
    echo "  -d    Base directory (default: ${DEFAULT_BASE_DIR})"
    echo "  -o    Owner (user:group) for the directories"
    echo "  -h    Show this help message"
    exit 1
}

# Main function
main() {
    local base_dir="${DEFAULT_BASE_DIR}"
    local owner=""

    # Parse command line arguments
    while getopts "d:o:h" opt; do
        case ${opt} in
            d)
                base_dir="${OPTARG}"
                ;;
            o)
                owner="${OPTARG}"
                ;;
            h)
                usage
                ;;
            \?)
                usage
                ;;
        esac
    done

    # Validate owner is provided
    if [[ -z "${owner}" ]]; then
        log_error "Owner (-o) must be specified"
        usage
    fi

    log_info "Starting directory structure setup..."
    log_info "Base directory: ${base_dir}"
    log_info "Owner: ${owner}"

    create_directories "${base_dir}" || exit 1
    set_permissions "${base_dir}" || exit 1
    set_ownership "${base_dir}" "${owner}" || exit 1
    verify_structure "${base_dir}" || exit 1

    log_info "Directory structure setup completed successfully"
}

# Execute main function
main "$@"
