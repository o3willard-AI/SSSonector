#!/bin/bash

# deploy.sh
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
DEPLOY_LOG="${SCRIPT_DIR}/deploy.log"
BACKUP_DIR="${SCRIPT_DIR}/backups/$(date +%Y%m%d_%H%M%S)"
DEPLOY_LOCK="/var/run/sssonector_deploy.lock"

# Deployment stages
declare -A STAGES=(
    ["pre_deploy"]="Pre-deployment checks"
    ["backup"]="Backup current deployment"
    ["validate"]="Validate deployment package"
    ["stop"]="Stop services"
    ["deploy"]="Deploy new version"
    ["configure"]="Configure deployment"
    ["start"]="Start services"
    ["verify"]="Verify deployment"
    ["cleanup"]="Cleanup deployment artifacts"
)

# Deployment configuration
declare -A CONFIG
CONFIG=(
    ["version"]=""
    ["environment"]=""
    ["package"]=""
    ["target_hosts"]=""
    ["deploy_user"]=""
    ["backup_enabled"]="true"
    ["rollback_on_failure"]="true"
    ["health_check_retries"]="3"
    ["health_check_interval"]="5"
)

# Load deployment configuration
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
    local required_keys=("version" "environment" "package" "target_hosts" "deploy_user")
    for key in "${required_keys[@]}"; do
        if [[ -z "${CONFIG[${key}]}" ]]; then
            log_error "Required configuration missing: ${key}"
            return 1
        fi
    done
}

# Pre-deployment checks
pre_deploy() {
    log_info "Running pre-deployment checks..."

    # Check deployment lock
    if [[ -f "${DEPLOY_LOCK}" ]]; then
        log_error "Another deployment is in progress"
        return 1
    fi
    touch "${DEPLOY_LOCK}"

    # Validate deployment package
    if [[ ! -f "${CONFIG[package]}" ]]; then
        log_error "Deployment package not found: ${CONFIG[package]}"
        return 1
    fi

    # Check target hosts connectivity
    IFS=',' read -ra hosts <<< "${CONFIG[target_hosts]}"
    for host in "${hosts[@]}"; do
        if ! ssh -q -o BatchMode=yes -o ConnectTimeout=5 "${CONFIG[deploy_user]}@${host}" exit; then
            log_error "Cannot connect to host: ${host}"
            return 1
        fi
    done

    # Verify deployment user permissions
    for host in "${hosts[@]}"; do
        if ! ssh "${CONFIG[deploy_user]}@${host}" "test -w ${DEFAULT_BASE_DIR}"; then
            log_error "Insufficient permissions on host ${host}"
            return 1
        fi
    done

    # Run QA environment validation
    if ! "${SCRIPT_DIR}/../qa_validator/qa_validator.sh" -d "${DEFAULT_BASE_DIR}"; then
        log_error "QA environment validation failed"
        return 1
    fi
}

# Backup current deployment
backup() {
    if [[ "${CONFIG[backup_enabled]}" != "true" ]]; then
        log_info "Backup disabled, skipping..."
        return 0
    fi

    log_info "Backing up current deployment..."
    mkdir -p "${BACKUP_DIR}"

    IFS=',' read -ra hosts <<< "${CONFIG[target_hosts]}"
    for host in "${hosts[@]}"; do
        log_info "Creating backup on ${host}..."
        
        # Create backup archive
        ssh "${CONFIG[deploy_user]}@${host}" "cd ${DEFAULT_BASE_DIR} && \
            tar czf - . --exclude='*.log' --exclude='*.pid'" > \
            "${BACKUP_DIR}/${host}_backup.tar.gz"

        # Verify backup
        if ! tar tzf "${BACKUP_DIR}/${host}_backup.tar.gz" &>/dev/null; then
            log_error "Backup verification failed for host: ${host}"
            return 1
        fi
    done
}

# Validate deployment package
validate() {
    log_info "Validating deployment package..."

    # Extract package to temporary directory
    local temp_dir
    temp_dir=$(mktemp -d)
    trap 'rm -rf "${temp_dir}"' EXIT

    tar xzf "${CONFIG[package]}" -C "${temp_dir}"

    # Verify package structure
    local required_dirs=("bin" "config" "certs" "tools")
    for dir in "${required_dirs[@]}"; do
        if [[ ! -d "${temp_dir}/${dir}" ]]; then
            log_error "Required directory missing in package: ${dir}"
            return 1
        fi
    done

    # Verify binary
    if [[ ! -x "${temp_dir}/bin/sssonector" ]]; then
        log_error "Binary not executable: bin/sssonector"
        return 1
    fi

    # Verify configuration templates
    if [[ ! -f "${temp_dir}/config/server_template.yaml" || ! -f "${temp_dir}/config/client_template.yaml" ]]; then
        log_error "Configuration templates missing"
        return 1
    fi

    # Verify version information
    local version_output
    version_output=$("${temp_dir}/bin/sssonector" --version)
    if [[ "${version_output}" != *"${CONFIG[version]}"* ]]; then
        log_error "Version mismatch. Expected: ${CONFIG[version]}, Found: ${version_output}"
        return 1
    fi
}

# Stop services
stop() {
    log_info "Stopping services..."

    IFS=',' read -ra hosts <<< "${CONFIG[target_hosts]}"
    for host in "${hosts[@]}"; do
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
    done
}

# Deploy new version
deploy() {
    log_info "Deploying new version..."

    IFS=',' read -ra hosts <<< "${CONFIG[target_hosts]}"
    for host in "${hosts[@]}"; do
        log_info "Deploying to ${host}..."

        # Transfer package
        if ! scp "${CONFIG[package]}" "${CONFIG[deploy_user]}@${host}:/tmp/"; then
            log_error "Failed to transfer package to host: ${host}"
            return 1
        fi

        # Extract package
        if ! ssh "${CONFIG[deploy_user]}@${host}" "cd ${DEFAULT_BASE_DIR} && \
            tar xzf /tmp/$(basename "${CONFIG[package]}") && \
            rm /tmp/$(basename "${CONFIG[package]}")"; then
            log_error "Failed to extract package on host: ${host}"
            return 1
        fi

        # Set permissions
        if ! ssh "${CONFIG[deploy_user]}@${host}" "chmod 755 ${DEFAULT_BASE_DIR}/bin/* && \
            chmod 644 ${DEFAULT_BASE_DIR}/config/*.yaml && \
            chmod 600 ${DEFAULT_BASE_DIR}/certs/*.key && \
            chmod 644 ${DEFAULT_BASE_DIR}/certs/*.crt"; then
            log_error "Failed to set permissions on host: ${host}"
            return 1
        fi
    done
}

# Configure deployment
configure() {
    log_info "Configuring deployment..."

    IFS=',' read -ra hosts <<< "${CONFIG[target_hosts]}"
    for host in "${hosts[@]}"; do
        log_info "Configuring ${host}..."

        # Apply environment-specific configuration
        if ! ssh "${CONFIG[deploy_user]}@${host}" \
            "${DEFAULT_BASE_DIR}/tools/config_validator/config_validator.sh" \
            -c "${DEFAULT_BASE_DIR}/config/${CONFIG[environment]}.yaml"; then
            log_error "Configuration validation failed on host: ${host}"
            return 1
        fi

        # Update system configuration
        if ! ssh "${CONFIG[deploy_user]}@${host}" "systemctl daemon-reload"; then
            log_error "Failed to reload system configuration on host: ${host}"
            return 1
        fi
    done
}

# Start services
start() {
    log_info "Starting services..."

    IFS=',' read -ra hosts <<< "${CONFIG[target_hosts]}"
    for host in "${hosts[@]}"; do
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
    done
}

# Verify deployment
verify() {
    log_info "Verifying deployment..."

    IFS=',' read -ra hosts <<< "${CONFIG[target_hosts]}"
    for host in "${hosts[@]}"; do
        log_info "Verifying deployment on ${host}..."

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
    done
}

# Cleanup deployment artifacts
cleanup() {
    log_info "Cleaning up deployment artifacts..."

    # Remove deployment lock
    rm -f "${DEPLOY_LOCK}"

    # Cleanup old backups (keep last 5)
    if [[ -d "${SCRIPT_DIR}/backups" ]]; then
        cd "${SCRIPT_DIR}/backups" && ls -t | tail -n +6 | xargs -r rm -rf
    fi

    # Remove temporary files
    find /tmp -name "sssonector_*" -mtime +1 -delete
}

# Rollback deployment
rollback() {
    if [[ "${CONFIG[rollback_on_failure]}" != "true" ]]; then
        log_error "Deployment failed, rollback disabled"
        return 1
    fi

    log_warn "Rolling back deployment..."

    IFS=',' read -ra hosts <<< "${CONFIG[target_hosts]}"
    for host in "${hosts[@]}"; do
        log_info "Rolling back ${host}..."

        # Stop services
        ssh "${CONFIG[deploy_user]}@${host}" "systemctl stop sssonector" || true

        # Restore backup
        if [[ -f "${BACKUP_DIR}/${host}_backup.tar.gz" ]]; then
            ssh "${CONFIG[deploy_user]}@${host}" "cd ${DEFAULT_BASE_DIR} && \
                rm -rf * && \
                tar xzf -" < "${BACKUP_DIR}/${host}_backup.tar.gz"
        fi

        # Start services
        ssh "${CONFIG[deploy_user]}@${host}" "systemctl start sssonector" || true
    done

    log_warn "Rollback complete"
}

# Main deployment function
main() {
    local config_file=""
    local stage_start=""

    # Parse command line arguments
    while getopts "c:s:h" opt; do
        case ${opt} in
            c)
                config_file="${OPTARG}"
                ;;
            s)
                stage_start="${OPTARG}"
                ;;
            h)
                echo "Usage: $0 [-c config_file] [-s start_stage]"
                echo
                echo "Options:"
                echo "  -c    Configuration file"
                echo "  -s    Start from specific stage"
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
    exec 1> >(tee -a "${DEPLOY_LOG}")
    exec 2>&1

    log_info "Starting deployment process..."
    log_info "Configuration file: ${config_file}"
    log_info "Deployment log: ${DEPLOY_LOG}"

    # Load configuration
    if ! load_config "${config_file}"; then
        log_error "Failed to load configuration"
        exit 1
    fi

    # Track if we should run each stage
    local run_stage=false
    if [[ -z "${stage_start}" ]]; then
        run_stage=true
    fi

    # Run deployment stages
    for stage in "${!STAGES[@]}"; do
        if [[ "${stage}" == "${stage_start}" ]]; then
            run_stage=true
        fi

        if [[ "${run_stage}" == "true" ]]; then
            log_info "Stage: ${STAGES[${stage}]}"
            if ! ${stage}; then
                log_error "Stage failed: ${STAGES[${stage}]}"
                rollback
                exit 1
            fi
        fi
    done

    log_info "Deployment completed successfully"
}

# Execute main function
main "$@"
