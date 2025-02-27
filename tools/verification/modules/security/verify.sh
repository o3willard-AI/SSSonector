#!/bin/bash

# security/verify.sh
# Security settings verification module
set -euo pipefail

# Source common utilities
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source "${SCRIPT_DIR}/../../lib/common.sh"

# Certificate verification
verify_certificates() {
    local failed=0

    log_info "Verifying certificate configuration"

    # Get environment type and paths
    local env_type
    env_type=$(load_state "ENVIRONMENT")
    
    # Determine certificate directory
    local cert_dir
    if [[ "${env_type}" =~ ^qa ]]; then
        cert_dir="/opt/sssonector/certs"
    else
        cert_dir="./dev/certs"
    fi

    # Required certificate files
    local -a cert_files=("ca.crt" "ca.key" "server.crt" "server.key" "client.crt" "client.key")
    
    # Check certificate files
    for cert in "${cert_files[@]}"; do
        local file="${cert_dir}/${cert}"
        if [[ -f "${file}" ]]; then
            # Check permissions
            local perms
            perms=$(stat -c "%a" "${file}")
            if [[ "${cert}" == *".key" ]]; then
                if [[ "${perms}" == "600" ]]; then
                    track_result "security_cert_perms_${cert}" "PASS" "Key file ${cert} has correct permissions"
                else
                    track_result "security_cert_perms_${cert}" "FAIL" "Key file ${cert} has incorrect permissions: ${perms}"
                    failed=1
                fi
            else
                if [[ "${perms}" == "644" ]]; then
                    track_result "security_cert_perms_${cert}" "PASS" "Certificate file ${cert} has correct permissions"
                else
                    track_result "security_cert_perms_${cert}" "FAIL" "Certificate file ${cert} has incorrect permissions: ${perms}"
                    failed=1
                fi
            fi

            # Verify certificate validity
            if [[ "${cert}" == *".crt" ]]; then
                local end_date
                end_date=$(openssl x509 -enddate -noout -in "${file}" | cut -d= -f2)
                local end_epoch
                end_epoch=$(date -d "${end_date}" +%s)
                local now_epoch
                now_epoch=$(date +%s)
                local days_remaining
                days_remaining=$(( (end_epoch - now_epoch) / 86400 ))

                if [[ ${days_remaining} -gt 30 ]]; then
                    track_result "security_cert_validity_${cert}" "PASS" "Certificate ${cert} valid for ${days_remaining} days"
                elif [[ ${days_remaining} -gt 0 ]]; then
                    track_result "security_cert_validity_${cert}" "WARN" "Certificate ${cert} expires in ${days_remaining} days"
                else
                    track_result "security_cert_validity_${cert}" "FAIL" "Certificate ${cert} has expired"
                    failed=1
                fi

                # Verify certificate chain
                if openssl verify -CAfile "${cert_dir}/ca.crt" "${file}" >/dev/null 2>&1; then
                    track_result "security_cert_chain_${cert}" "PASS" "Certificate ${cert} has valid chain"
                else
                    track_result "security_cert_chain_${cert}" "FAIL" "Certificate ${cert} has invalid chain"
                    failed=1
                fi
            fi
        else
            track_result "security_cert_exists_${cert}" "FAIL" "Certificate file ${cert} not found"
            failed=1
        fi
    done

    return ${failed}
}

# Memory protection verification
verify_memory_protections() {
    local failed=0

    log_info "Verifying memory protection settings"

    # Check ASLR
    local aslr_status
    aslr_status=$(cat /proc/sys/kernel/randomize_va_space)
    if [[ "${aslr_status}" == "2" ]]; then
        track_result "security_aslr" "PASS" "ASLR is fully enabled"
    elif [[ "${aslr_status}" == "1" ]]; then
        track_result "security_aslr" "WARN" "ASLR is partially enabled"
    else
        track_result "security_aslr" "FAIL" "ASLR is disabled"
        failed=1
    fi

    # Check NX bit
    if grep -q nx /proc/cpuinfo; then
        track_result "security_nx" "PASS" "NX bit is supported"
        # Check if NX is enabled in kernel
        if dmesg | grep -q "NX (Execute Disable) protection: active"; then
            track_result "security_nx_active" "PASS" "NX protection is active"
        else
            track_result "security_nx_active" "FAIL" "NX protection is not active"
            failed=1
        fi
    else
        track_result "security_nx" "FAIL" "NX bit is not supported"
        failed=1
    fi

    # Check stack protector
    if gcc -fstack-protector -E -x c /dev/null >/dev/null 2>&1; then
        track_result "security_stack_protector" "PASS" "Stack protector is available"
    else
        track_result "security_stack_protector" "FAIL" "Stack protector is not available"
        failed=1
    fi

    return ${failed}
}

# Namespace verification
verify_namespaces() {
    local failed=0

    log_info "Verifying namespace support"

    # Check namespace support in kernel
    if [[ -f /proc/self/ns ]]; then
        track_result "security_namespace_support" "PASS" "Namespace support available"
        
        # Check specific namespace types
        local -a required_ns=("net" "mnt" "pid")
        for ns in "${required_ns[@]}"; do
            if [[ -e "/proc/self/ns/${ns}" ]]; then
                track_result "security_namespace_${ns}" "PASS" "${ns} namespace supported"
            else
                track_result "security_namespace_${ns}" "FAIL" "${ns} namespace not supported"
                failed=1
            fi
        done
    else
        track_result "security_namespace_support" "FAIL" "Namespace support not available"
        failed=1
    fi

    # Check unshare capability
    if unshare --help >/dev/null 2>&1; then
        track_result "security_unshare" "PASS" "unshare command available"
    else
        track_result "security_unshare" "FAIL" "unshare command not available"
        failed=1
    fi

    return ${failed}
}

# Capability verification
verify_capabilities() {
    local failed=0

    log_info "Verifying capability support"

    # Check capability support
    if command -v capsh >/dev/null 2>&1; then
        track_result "security_capabilities" "PASS" "Capability support available"
        
        # Check specific capabilities
        local -a required_caps=("CAP_NET_ADMIN" "CAP_NET_RAW" "CAP_NET_BIND_SERVICE")
        for cap in "${required_caps[@]}"; do
            if capsh --print | grep -q "${cap}"; then
                track_result "security_capability_${cap}" "PASS" "${cap} capability available"
            else
                track_result "security_capability_${cap}" "FAIL" "${cap} capability not available"
                failed=1
            fi
        done
    else
        track_result "security_capabilities" "WARN" "Capability tools not installed"
    fi

    return ${failed}
}

# Main verification function
main() {
    local failed=0

    # Run verifications
    verify_certificates || failed=1
    verify_memory_protections || failed=1
    verify_namespaces || failed=1
    verify_capabilities || failed=1

    return ${failed}
}

# Run main function
main "$@"
