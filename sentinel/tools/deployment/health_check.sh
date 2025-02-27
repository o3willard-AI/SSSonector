#!/bin/bash

# health_check.sh
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
HEALTH_LOG="${SCRIPT_DIR}/health_check.log"
METRICS_DIR="${SCRIPT_DIR}/metrics"

# Health check configuration
declare -A CONFIG
CONFIG=(
    ["target_hosts"]=""
    ["deploy_user"]=""
    ["check_interval"]="60"
    ["failure_threshold"]="3"
    ["success_threshold"]="2"
    ["metrics_enabled"]="true"
    ["alert_enabled"]="true"
    ["alert_webhook"]=""
)

# Load health check configuration
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

# Check service status
check_service_status() {
    local host=$1
    log_info "Checking service status on ${host}..."

    # Check if service is running
    if ! ssh "${CONFIG[deploy_user]}@${host}" "systemctl is-active sssonector" &>/dev/null; then
        return 1
    fi

    # Check service health
    if ! ssh "${CONFIG[deploy_user]}@${host}" "systemctl status sssonector | grep -q 'active (running)'"; then
        return 1
    fi

    return 0
}

# Check process resources
check_process_resources() {
    local host=$1
    log_info "Checking process resources on ${host}..."

    # Get process statistics
    local pid
    pid=$(ssh "${CONFIG[deploy_user]}@${host}" "systemctl show -p MainPID sssonector | cut -d= -f2")

    if [[ -z "${pid}" || "${pid}" == "0" ]]; then
        log_error "Process not found on ${host}"
        return 1
    fi

    # Check CPU usage
    local cpu_usage
    cpu_usage=$(ssh "${CONFIG[deploy_user]}@${host}" "ps -p ${pid} -o %cpu= | tr -d ' '")
    if (( $(echo "${cpu_usage} > 80" | bc -l) )); then
        log_error "High CPU usage on ${host}: ${cpu_usage}%"
        return 1
    fi

    # Check memory usage
    local mem_usage
    mem_usage=$(ssh "${CONFIG[deploy_user]}@${host}" "ps -p ${pid} -o %mem= | tr -d ' '")
    if (( $(echo "${mem_usage} > 80" | bc -l) )); then
        log_error "High memory usage on ${host}: ${mem_usage}%"
        return 1
    fi

    return 0
}

# Check network connectivity
check_network_connectivity() {
    local host=$1
    log_info "Checking network connectivity on ${host}..."

    # Check network interfaces
    if ! ssh "${CONFIG[deploy_user]}@${host}" "ip link show tun0" &>/dev/null; then
        log_error "TUN interface not found on ${host}"
        return 1
    fi

    # Check network connectivity
    if ! ssh "${CONFIG[deploy_user]}@${host}" "ping -c 1 -W 2 8.8.8.8" &>/dev/null; then
        log_error "Network connectivity check failed on ${host}"
        return 1
    fi

    return 0
}

# Check logs for errors
check_logs() {
    local host=$1
    log_info "Checking logs on ${host}..."

    # Check for errors in service logs
    if ssh "${CONFIG[deploy_user]}@${host}" "journalctl -u sssonector -n 100 --no-pager" | \
        grep -i "error\|failed\|fatal" &>/dev/null; then
        log_error "Found errors in service logs on ${host}"
        return 1
    fi

    return 0
}

# Record metrics
record_metrics() {
    local host=$1
    local status=$2
    local timestamp
    timestamp=$(date +%s)

    if [[ "${CONFIG[metrics_enabled]}" != "true" ]]; then
        return 0
    fi

    # Create metrics directory if it doesn't exist
    mkdir -p "${METRICS_DIR}"

    # Record health check result
    echo "${timestamp} ${status}" >> "${METRICS_DIR}/${host}_health.log"

    # Cleanup old metrics (keep last 24 hours)
    find "${METRICS_DIR}" -name "*_health.log" -mtime +1 -delete

    # If Prometheus node exporter is available, update its metrics
    if [[ -d "/var/lib/node_exporter/textfile" ]]; then
        echo "# HELP sssonector_health_status Service health status (1 = healthy, 0 = unhealthy)" > \
            "/var/lib/node_exporter/textfile/sssonector.prom"
        echo "# TYPE sssonector_health_status gauge" >> \
            "/var/lib/node_exporter/textfile/sssonector.prom"
        echo "sssonector_health_status{host=\"${host}\"} ${status}" >> \
            "/var/lib/node_exporter/textfile/sssonector.prom"
    fi
}

# Send alerts
send_alert() {
    local host=$1
    local message=$2

    if [[ "${CONFIG[alert_enabled]}" != "true" || -z "${CONFIG[alert_webhook]}" ]]; then
        return 0
    fi

    # Prepare alert payload
    local payload
    payload=$(cat <<EOF
{
    "host": "${host}",
    "status": "unhealthy",
    "message": "${message}",
    "timestamp": "$(date -u +"%Y-%m-%dT%H:%M:%SZ")"
}
EOF
)

    # Send alert
    curl -s -X POST -H "Content-Type: application/json" \
        -d "${payload}" "${CONFIG[alert_webhook]}" || true
}

# Perform health check
perform_health_check() {
    local host=$1
    local failed=0

    # Check service status
    if ! check_service_status "${host}"; then
        log_error "Service status check failed on ${host}"
        failed=1
    fi

    # Check process resources
    if ! check_process_resources "${host}"; then
        log_error "Process resource check failed on ${host}"
        failed=1
    fi

    # Check network connectivity
    if ! check_network_connectivity "${host}"; then
        log_error "Network connectivity check failed on ${host}"
        failed=1
    fi

    # Check logs
    if ! check_logs "${host}"; then
        log_error "Log check failed on ${host}"
        failed=1
    fi

    # Record metrics and send alerts if needed
    if [[ ${failed} -eq 0 ]]; then
        record_metrics "${host}" 1
    else
        record_metrics "${host}" 0
        send_alert "${host}" "Health check failed"
    fi

    return ${failed}
}

# Main health check function
main() {
    local config_file=""
    local continuous=false

    # Parse command line arguments
    while getopts "c:dh" opt; do
        case ${opt} in
            c)
                config_file="${OPTARG}"
                ;;
            d)
                continuous=true
                ;;
            h)
                echo "Usage: $0 [-c config_file] [-d]"
                echo
                echo "Options:"
                echo "  -c    Configuration file"
                echo "  -d    Run continuously as a daemon"
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
    exec 1> >(tee -a "${HEALTH_LOG}")
    exec 2>&1

    # Load configuration
    if ! load_config "${config_file}"; then
        log_error "Failed to load configuration"
        exit 1
    fi

    # Split target hosts
    IFS=',' read -ra hosts <<< "${CONFIG[target_hosts]}"

    if [[ "${continuous}" == "true" ]]; then
        log_info "Starting continuous health checks..."
        while true; do
            for host in "${hosts[@]}"; do
                perform_health_check "${host}"
            done
            sleep "${CONFIG[check_interval]}"
        done
    else
        log_info "Running one-time health check..."
        local failed=0
        for host in "${hosts[@]}"; do
            perform_health_check "${host}" || failed=1
        done
        exit ${failed}
    fi
}

# Execute main function
main "$@"
