#!/bin/bash

# rollback.sh
# Part of Project SENTINEL - Deployment Automation
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

# Default paths and settings
DEFAULT_BASE_DIR="/opt/sssonector"
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
ROLLBACK_LOG="${SCRIPT_DIR}/rollback.log"
BACKUP_DIR="${SCRIPT_DIR}/backups"
ROLLBACK_LOCK="/var/run/sssonector_rollback.lock"

# Rollback configuration
declare -A CONFIG
CONFIG=(
    ["target_hosts"]=""
    ["deploy_user"]=""
    ["version"]=""
    ["verify_after_rollback"]="true"
    ["health_check_retries"]="3"
    ["health_check_interval"]="5"
)

# Load rollback configuration
load_config() {
    local config_file=$1
    log_info "Loading configuration from ${config_file}"

    if [[ ! -f "${config_file}" ]]; then
        log_error "Configuration file not found: ${config_file}"
        return 1
    fi

    # Read configuration
    while IFS='=' read -r key value; do
        if [[ -n "${key}" && ! "${key}" =~ ^[[:space:]]*# ]]; then
            CONFIG["${key}"]="${value}"
        fi
    done < "${config_file}"

    # Validate required configuration
    local required_keys=("target_hosts" "deploy_user")
    for key in "${required_keys[@]}"; do
        if [[ -z "${CONFIG[${key}]}" ]]; then
            log_error "Required configuration missing: ${key}"
            return 1
        fi
    done
}

# List available backups
list_backups() {
    local host=$1
    log_info "Available backups for host: ${host}"

    if [[ ! -d "${BACKUP_DIR}" ]]; then
        log_error "No backups directory found"
        return 1
    fi

    # Find all backup files for the host
    local backups=()
    while IFS= read -r backup; do
        local timestamp
        timestamp=$(basename "$(dirname "${backup}")")
        local version
        version=$(tar xzf "${backup}" -O ./version 2>/dev/null || echo "unknown")
        backups+=("${timestamp} (Version: ${version})")
    done < <(find "${BACKUP_DIR}" -name "${host}_backup.tar.gz" -type f)

    if [[ ${#backups[@]} -eq 0 ]]; then
        log_warn "No backups found for host: ${host}"
        return 1
    fi

    # Display backups in reverse chronological order
    printf '%s\n' "${backups[@]}" | sort -r
}

# Verify backup integrity
verify_backup() {
    local backup_file=$1
    log_info "Verifying backup integrity: ${backup_file}"

    # Check if backup exists
    if [[ ! -f "${backup_file}" ]]; then
        log_error "Backup file not found: ${backup_file}"
        return 1
    fi

    # Verify backup can be read
    if ! tar tzf "${backup_file}" &>/dev/null; then
        log_error "Backup file is corrupted: ${backup_file}"
        return 1
    fi

    # Check required files in backup
    local required_files=("bin/sssonector" "config/server_template.yaml" "config/client_template.yaml")
    for file in "${required_files[@]}"; do
        if ! tar tzf "${backup_file}" | grep -q "^\./${file}$"; then
            log_error "Required file missing in backup: ${file}"
            return 1
        fi
    done

    return 0
}

# Stop services
stop_services() {
    local host=$1
    log_info "Stopping services on ${host}..."

    if ! ssh "${CONFIG[deploy_user]}@${host}" "systemctl stop sssonector"; then
        log_error "Failed to stop services on host: ${host}"
        return 1
    fi

    # Wait for services to stop
    local retries=0
    while ssh "${CONFIG[deploy_user]}@${host}" "systemctl is-active sssonector" &>/dev/null; do
        if ((retries >= 5)); then
            log_error "Timeout waiting for services to stop on host: ${host}"
            return 1
        fi
        sleep 2
        ((retries++))
    done
}

# Restore backup
restore_backup() {
    local host=$1
    local backup_file=$2
    log_info "Restoring backup on ${host}..."

    # Stop services first
    stop_services "${host}" || return 1

    # Create temporary backup of current state
    local temp_backup
    temp_backup=$(mktemp)
    trap 'rm -f "${temp_backup}"' EXIT

    ssh "${CONFIG[deploy_user]}@${host}" "cd ${DEFAULT_BASE_DIR} && tar czf - ." > "${temp_backup}"

    # Restore from backup
    if ! ssh "${CONFIG[deploy_user]}@${host}" "cd ${DEFAULT_BASE_DIR} && rm -rf * && tar xzf -" < "${backup_file}"; then
        log_error "Failed to restore backup on host: ${host}"
        # Attempt to restore temporary backup
        log_warn "Attempting to restore previous state..."
        ssh "${CONFIG[deploy_user]}@${host}" "cd ${DEFAULT_BASE_DIR} && rm -rf * && tar xzf -" < "${temp_backup}"
        return 1
    fi

    # Set correct permissions
    ssh "${CONFIG[deploy_user]}@${host}" "chmod 755 ${DEFAULT_BASE_DIR}/bin/* && \
        chmod 644 ${DEFAULT_BASE_DIR}/config/*.yaml && \
        chmod 600 ${DEFAULT_BASE_DIR}/certs/*.key && \
        chmod 644 ${DEFAULT_BASE_DIR}/certs/*.crt" || return 1
}

# Start services
start_services() {
    local host=$1
    log_info "Starting services on ${host}..."

    if ! ssh "${CONFIG[deploy_user]}@${host}" "systemctl start sssonector"; then
        log_error "Failed to start services on host: ${host}"
        return 1
    fi

    # Wait for services to start
    local retries=0
    while ! ssh "${CONFIG[deploy_user]}@${host}" "systemctl is-active sssonector" &>/dev/null; do
        if ((retries >= "${CONFIG[health_check_retries]}")); then
            log_error "Timeout waiting for services to start on host: ${host}"
            return 1
        fi
        sleep "${CONFIG[health_check_interval]}"
        ((retries++))
    done
}

# Verify rollback
verify_rollback() {
    local host=$1
    log_info "Verifying rollback on ${host}..."

    # Check service status
    if ! ssh "${CONFIG[deploy_user]}@${host}" "systemctl status sssonector"; then
        log_error "Service verification failed on host: ${host}"
        return 1
    fi

    # Run health checks
    if ! ssh "${CONFIG[deploy_user]}@${host}" \
        "${DEFAULT_BASE_DIR}/tools/qa_validator/qa_validator.sh" \
        -d "${DEFAULT_BASE_DIR}"; then
        log_error "Health check failed on host: ${host}"
        return 1
    fi

    # Verify version if specified
    if [[ -n "${CONFIG[version]}" ]]; then
        local current_version
        current_version=$(ssh "${CONFIG[deploy_user]}@${host}" "${DEFAULT_BASE_DIR}/bin/sssonector --version")
        if [[ "${current_version}" != *"${CONFIG[version]}"* ]]; then
            log_error "Version mismatch after rollback. Expected: ${CONFIG[version]}, Found: ${current_version}"
            return 1
        fi
    fi
}

# Main rollback function
main() {
    local config_file=""
    local backup_timestamp=""
    local list_only=false

    # Parse command line arguments
    while getopts "c:t:lh" opt; do
        case ${opt} in
            c)
                config_file="${OPTARG}"
                ;;
            t)
                backup_timestamp="${OPTARG}"
                ;;
            l)
                list_only=true
                ;;
            h)
                echo "Usage: $0 [-c config_file] [-t timestamp] [-l]"
                echo
                echo "Options:"
                echo "  -c    Configuration file"
                echo "  -t    Backup timestamp to restore"
                echo "  -l    List available backups"
                echo "  -h    Show this help message"
                exit 0
                ;;
            \?)
                echo "Invalid option: -${OPTARG}"
                exit 1
                ;;
        esac
    done

    if [[ -z "${config_file}" ]]; then
        log_error "Configuration file required"
        exit 1
    fi

    # Initialize logging
    exec 1> >(tee -a "${ROLLBACK_LOG}")
    exec 2>&1

    # Load configuration
    if ! load_config "${config_file}"; then
        log_error "Failed to load configuration"
        exit 1
    fi

    # Check rollback lock
    if [[ -f "${ROLLBACK_LOCK}" ]]; then
        log_error "Another rollback is in progress"
        exit 1
    fi
    touch "${ROLLBACK_LOCK}"
    trap 'rm -f "${ROLLBACK_LOCK}"' EXIT

    # Split target hosts
    IFS=',' read -ra hosts <<< "${CONFIG[target_hosts]}"

    # List backups if requested
    if [[ "${list_only}" == "true" ]]; then
        for host in "${hosts[@]}"; do
            list_backups "${host}"
        done
        exit 0
    fi

    # Verify backup timestamp provided
    if [[ -z "${backup_timestamp}" ]]; then
        log_error "Backup timestamp required"
        exit 1
    fi

    log_info "Starting rollback process..."
    log_info "Configuration file: ${config_file}"
    log_info "Backup timestamp: ${backup_timestamp}"
    log_info "Rollback log: ${ROLLBACK_LOG}"

    # Process each host
    for host in "${hosts[@]}"; do
        log_info "Processing host: ${host}"

        # Find backup file
        local backup_file="${BACKUP_DIR}/${backup_timestamp}/${host}_backup.tar.gz"

        # Verify backup
        if ! verify_backup "${backup_file}"; then
            log_error "Backup verification failed for host: ${host}"
            continue
        fi

        # Restore backup
        if ! restore_backup "${host}" "${backup_file}"; then
            log_error "Backup restoration failed for host: ${host}"
            continue
        fi

        # Start services
        if ! start_services "${host}"; then
            log_error "Failed to start services on host: ${host}"
            continue
        fi

        # Verify rollback if enabled
        if [[ "${CONFIG[verify_after_rollback]}" == "true" ]]; then
            if ! verify_rollback "${host}"; then
                log_error "Rollback verification failed for host: ${host}"
                continue
            fi
        fi

        log_info "Rollback completed successfully for host: ${host}"
    done

    log_info "Rollback process completed"
}

# Execute main function
main "$@"
