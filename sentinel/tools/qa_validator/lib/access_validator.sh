#!/bin/bash

# access_validator.sh
# Part of Project SENTINEL - QA Environment Validation Tool
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

# SSH access validation
validate_ssh_access() {
    local base_dir=$1
    local failed=0

    log_info "Validating SSH access..."

    # Check SSH key presence
    local ssh_dir="${HOME}/.ssh"
    if [[ ! -d "${ssh_dir}" ]]; then
        log_error "SSH directory not found"
        failed=1
    else
        # Check for SSH keys
        if ! find "${ssh_dir}" -name "id_*" | grep -q .; then
            log_error "No SSH keys found"
            failed=1
        fi

        # Check SSH key permissions
        while IFS= read -r key_file; do
            local perms
            perms=$(stat -c "%a" "${key_file}")
            if [[ "${key_file}" == *".pub" && "${perms}" != "644" ]]; then
                log_error "Invalid public key permissions: ${perms} (expected 644) for ${key_file}"
                failed=1
            elif [[ "${key_file}" != *".pub" && "${perms}" != "600" ]]; then
                log_error "Invalid private key permissions: ${perms} (expected 600) for ${key_file}"
                failed=1
            fi
        done < <(find "${ssh_dir}" -name "id_*")

        # Check known_hosts file
        if [[ ! -f "${ssh_dir}/known_hosts" ]]; then
            log_warn "known_hosts file not found"
        fi
    fi

    # Check SSH config
    if [[ -f "${ssh_dir}/config" ]]; then
        local config_perms
        config_perms=$(stat -c "%a" "${ssh_dir}/config")
        if [[ "${config_perms}" != "600" ]]; then
            log_error "Invalid SSH config permissions: ${config_perms} (expected 600)"
            failed=1
        fi

        # Check for QA host entries
        if ! grep -q "Host.*qa" "${ssh_dir}/config"; then
            log_warn "No QA host entries found in SSH config"
        fi
    else
        log_warn "SSH config file not found"
    fi

    return ${failed}
}

# FTP access validation
validate_ftp_access() {
    local base_dir=$1
    local failed=0

    log_info "Validating FTP access..."

    # Check FTP client installation
    if ! command -v ftp &>/dev/null; then
        log_error "FTP client not installed"
        failed=1
    fi

    # Check for FTP configuration
    local ftp_config="${HOME}/.netrc"
    if [[ -f "${ftp_config}" ]]; then
        local config_perms
        config_perms=$(stat -c "%a" "${ftp_config}")
        if [[ "${config_perms}" != "600" ]]; then
            log_error "Invalid .netrc permissions: ${config_perms} (expected 600)"
            failed=1
        fi

        # Check for QA host entries
        if ! grep -q "machine.*qa" "${ftp_config}"; then
            log_warn "No QA host entries found in FTP config"
        fi
    else
        log_warn ".netrc file not found"
    fi

    return ${failed}
}

# Node connectivity validation
validate_node_connectivity() {
    local base_dir=$1
    local failed=0

    log_info "Validating node connectivity..."

    # Get QA nodes from config
    local qa_nodes=()
    while IFS= read -r config_file; do
        if [[ -f "${config_file}" ]]; then
            while IFS= read -r node; do
                qa_nodes+=("${node}")
            done < <(yq eval '.qa_nodes[]' "${config_file}" 2>/dev/null || echo "")
        fi
    done < <(find "${base_dir}/config" -name "*.yaml")

    if [[ ${#qa_nodes[@]} -eq 0 ]]; then
        log_warn "No QA nodes found in configuration"
        return 0
    fi

    # Check connectivity to each node
    for node in "${qa_nodes[@]}"; do
        # Check SSH connectivity
        if ! ssh -q -o BatchMode=yes -o ConnectTimeout=5 "${node}" exit &>/dev/null; then
            log_error "SSH connectivity failed for node: ${node}"
            failed=1
        fi

        # Check FTP connectivity
        if ! (echo "quit" | ftp -n "${node}" &>/dev/null); then
            log_error "FTP connectivity failed for node: ${node}"
            failed=1
        fi

        # Check SFTP connectivity
        if ! sftp -q -o BatchMode=yes -o ConnectTimeout=5 "${node}" <<< "quit" &>/dev/null; then
            log_error "SFTP connectivity failed for node: ${node}"
            failed=1
        fi
    done

    return ${failed}
}

# Interactive access validation
validate_interactive_access() {
    local base_dir=$1
    local failed=0

    log_info "Validating interactive access capabilities..."

    # Check terminal capabilities
    if [[ ! -t 0 ]]; then
        log_warn "No interactive terminal available"
    fi

    # Check for required interactive tools
    local interactive_tools=(
        "tmux"      # Terminal multiplexer
        "screen"    # Terminal multiplexer (alternative)
        "expect"    # Automation for interactive applications
    )

    for tool in "${interactive_tools[@]}"; do
        if ! command -v "${tool}" &>/dev/null; then
            log_warn "Interactive tool not found: ${tool}"
        fi
    done

    # Check SSH agent
    if [[ -z "${SSH_AGENT_PID:-}" ]]; then
        log_warn "SSH agent not running"
    fi

    # Check for SSH agent forwarding in config
    if [[ -f "${HOME}/.ssh/config" ]]; then
        if ! grep -q "ForwardAgent.*yes" "${HOME}/.ssh/config"; then
            log_warn "SSH agent forwarding not configured"
        fi
    fi

    return ${failed}
}

# File transfer validation
validate_file_transfer() {
    local base_dir=$1
    local failed=0

    log_info "Validating file transfer capabilities..."

    # Check for file transfer tools
    local transfer_tools=(
        "scp"       # Secure copy
        "rsync"     # Remote sync
        "sftp"      # Secure FTP
        "curl"      # URL transfer tool
        "wget"      # Web download tool
    )

    for tool in "${transfer_tools[@]}"; do
        if ! command -v "${tool}" &>/dev/null; then
            log_error "File transfer tool not found: ${tool}"
            failed=1
        fi
    done

    # Check transfer directories
    local transfer_dirs=(
        "${base_dir}/transfer"
        "${base_dir}/incoming"
        "${base_dir}/outgoing"
    )

    for dir in "${transfer_dirs[@]}"; do
        if [[ ! -d "${dir}" ]]; then
            mkdir -p "${dir}"
        fi
        
        local dir_perms
        dir_perms=$(stat -c "%a" "${dir}")
        if [[ "${dir_perms}" != "755" ]]; then
            log_error "Invalid transfer directory permissions: ${dir_perms} (expected 755) for ${dir}"
            failed=1
        fi
    done

    return ${failed}
}

# Main access validation function
validate_access() {
    local base_dir=$1
    local failed=0

    # Validate SSH access
    validate_ssh_access "${base_dir}" || failed=1

    # Validate FTP access
    validate_ftp_access "${base_dir}" || failed=1

    # Validate node connectivity
    validate_node_connectivity "${base_dir}" || failed=1

    # Validate interactive access
    validate_interactive_access "${base_dir}" || failed=1

    # Validate file transfer capabilities
    validate_file_transfer "${base_dir}" || failed=1

    return ${failed}
}

# If script is run directly, show usage
if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then
    echo "This script is meant to be sourced by qa_validator.sh"
    exit 1
fi
