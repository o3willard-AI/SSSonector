#!/bin/bash

# security_validator.sh
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

# TPM verification
validate_tpm() {
    local failed=0

    log_info "Validating TPM state..."

    # Check if TPM is present
    if ! command -v tpm2_getcap &>/dev/null; then
        log_error "TPM2 tools not installed"
        return 1
    fi

    # Check TPM presence and activation
    if ! tpm2_getcap properties-fixed &>/dev/null; then
        log_error "TPM not present or accessible"
        failed=1
    fi

    # Check TPM ownership
    if ! tpm2_getcap properties-variable | grep -q "ownerAuthSet: set"; then
        log_warn "TPM ownership not set"
        failed=1
    fi

    # Check PCR banks
    if ! tpm2_getcap pcrs &>/dev/null; then
        log_error "Unable to read PCR banks"
        failed=1
    fi

    return ${failed}
}

# Secure boot validation
validate_secure_boot() {
    local failed=0

    log_info "Validating Secure Boot state..."

    # Check if system supports Secure Boot
    if [[ ! -d "/sys/firmware/efi" ]]; then
        log_warn "System not booted in EFI mode"
        return 1
    fi

    # Check Secure Boot state
    if [[ -f "/sys/firmware/efi/efivars/SecureBoot-8be4df61-93ca-11d2-aa0d-00e098032b8c" ]]; then
        local secure_boot_state
        secure_boot_state=$(od -An -t u1 /sys/firmware/efi/efivars/SecureBoot-8be4df61-93ca-11d2-aa0d-00e098032b8c | awk 'NR==1{print $NF}')
        if [[ "${secure_boot_state}" != "1" ]]; then
            log_error "Secure Boot not enabled"
            failed=1
        fi
    else
        log_error "Unable to determine Secure Boot state"
        failed=1
    fi

    return ${failed}
}

# Runtime integrity checking
validate_runtime_integrity() {
    local base_dir=$1
    local failed=0

    log_info "Validating runtime integrity..."

    # Check binary integrity
    local binary="${base_dir}/bin/sssonector"
    if [[ ! -f "${binary}.sha256" ]]; then
        log_error "Binary checksum file not found"
        failed=1
    else
        if ! (cd "${base_dir}/bin" && sha256sum -c sssonector.sha256); then
            log_error "Binary integrity check failed"
            failed=1
        fi
    fi

    # Check library integrity
    while IFS= read -r lib; do
        if [[ -f "${lib}.sha256" ]]; then
            if ! (cd "$(dirname "${lib}")" && sha256sum -c "$(basename "${lib}").sha256"); then
                log_error "Library integrity check failed: ${lib}"
                failed=1
            fi
        fi
    done < <(ldd "${binary}" | grep "=>" | awk '{print $3}')

    # Check kernel module integrity
    if command -v kmod &>/dev/null; then
        while IFS= read -r module; do
            if ! modprobe --dry-run "${module}" &>/dev/null; then
                log_error "Kernel module integrity check failed: ${module}"
                failed=1
            fi
        done < <(lsmod | awk 'NR>1{print $1}')
    fi

    return ${failed}
}

# Security policy enforcement
validate_security_policy() {
    local base_dir=$1
    local failed=0

    log_info "Validating security policy enforcement..."

    # Check SELinux/AppArmor status
    if command -v getenforce &>/dev/null; then
        local selinux_state
        selinux_state=$(getenforce)
        if [[ "${selinux_state}" != "Enforcing" ]]; then
            log_error "SELinux not in enforcing mode"
            failed=1
        fi
    elif command -v aa-status &>/dev/null; then
        if ! aa-status --enabled &>/dev/null; then
            log_error "AppArmor not enabled"
            failed=1
        fi
    else
        log_warn "No mandatory access control system detected"
        failed=1
    fi

    # Check process hardening
    local pid
    pid=$(pgrep -f "${base_dir}/bin/sssonector" || echo "")
    if [[ -n "${pid}" ]]; then
        # Check ASLR
        local aslr_state
        aslr_state=$(cat /proc/sys/kernel/randomize_va_space)
        if [[ "${aslr_state}" != "2" ]]; then
            log_error "ASLR not fully enabled"
            failed=1
        fi

        # Check stack protection
        if ! readelf -s "${base_dir}/bin/sssonector" | grep -q "__stack_chk_fail"; then
            log_error "Stack protection not enabled"
            failed=1
        fi

        # Check process capabilities
        if command -v getcap &>/dev/null; then
            if getcap "${base_dir}/bin/sssonector" | grep -q "=ep"; then
                log_error "Binary has excessive capabilities"
                failed=1
            fi
        fi
    else
        log_error "Process not running"
        failed=1
    fi

    return ${failed}
}

# Compliance verification
validate_compliance() {
    local base_dir=$1
    local failed=0

    log_info "Validating security compliance..."

    # Check file permissions
    while IFS= read -r file; do
        local perms
        perms=$(stat -c "%a" "${file}")
        
        # Check executable permissions
        if [[ "${file}" == *.sh || "${file}" == "${base_dir}/bin/"* ]]; then
            if [[ "${perms}" != "755" ]]; then
                log_error "Invalid permissions on executable: ${file} (${perms})"
                failed=1
            fi
        # Check configuration file permissions
        elif [[ "${file}" == *.yaml || "${file}" == *.conf ]]; then
            if [[ "${perms}" != "644" ]]; then
                log_error "Invalid permissions on config file: ${file} (${perms})"
                failed=1
            fi
        # Check private key permissions
        elif [[ "${file}" == *.key || "${file}" == *.pem ]]; then
            if [[ "${perms}" != "600" ]]; then
                log_error "Invalid permissions on private key: ${file} (${perms})"
                failed=1
            fi
        fi
    done < <(find "${base_dir}" -type f)

    # Check network security
    if command -v ss &>/dev/null; then
        # Check for unencrypted connections
        if ss -tuln | grep -q ":80\|:23\|:21"; then
            log_error "Unencrypted services detected"
            failed=1
        fi
    fi

    return ${failed}
}

# Main security validation function
validate_security() {
    local base_dir=$1
    local failed=0

    # Validate TPM state
    validate_tpm || failed=1

    # Validate Secure Boot
    validate_secure_boot || failed=1

    # Validate runtime integrity
    validate_runtime_integrity "${base_dir}" || failed=1

    # Validate security policy
    validate_security_policy "${base_dir}" || failed=1

    # Validate compliance
    validate_compliance "${base_dir}" || failed=1

    return ${failed}
}

# If script is run directly, show usage
if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then
    echo "This script is meant to be sourced by qa_validator.sh"
    exit 1
fi
