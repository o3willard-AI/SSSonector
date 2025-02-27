#!/bin/bash

# monitor.sh
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
MONITOR_LOG="${SCRIPT_DIR}/monitor.log"
METRICS_DIR="${SCRIPT_DIR}/metrics"
PROMETHEUS_DIR="/var/lib/node_exporter/textfile"

# Monitoring configuration
declare -A CONFIG
CONFIG=(
    ["target_hosts"]=""
    ["deploy_user"]=""
    ["collect_interval"]="30"
    ["retention_days"]="7"
    ["prometheus_enabled"]="true"
    ["grafana_enabled"]="true"
    ["grafana_api_key"]=""
    ["grafana_url"]=""
)

# Load monitoring configuration
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

# Collect system metrics
collect_system_metrics() {
    local host=$1
    local timestamp
    timestamp=$(date +%s)
    local metrics_file="${METRICS_DIR}/${host}_system_${timestamp}.metrics"

    log_info "Collecting system metrics from ${host}..."

    # Create metrics directory if it doesn't exist
    mkdir -p "${METRICS_DIR}"

    # Collect CPU metrics
    ssh "${CONFIG[deploy_user]}@${host}" "top -bn1" | \
        awk '/Cpu/ {printf "cpu_user=%.1f\ncpu_system=%.1f\ncpu_idle=%.1f\n", $2, $4, $8}' \
        > "${metrics_file}"

    # Collect memory metrics
    ssh "${CONFIG[deploy_user]}@${host}" "free -m" | \
        awk '/Mem:/ {printf "memory_total=%d\nmemory_used=%d\nmemory_free=%d\n", $2, $3, $4}' \
        >> "${metrics_file}"

    # Collect disk metrics
    ssh "${CONFIG[deploy_user]}@${host}" "df -m ${DEFAULT_BASE_DIR}" | \
        awk 'NR==2 {printf "disk_total=%d\ndisk_used=%d\ndisk_free=%d\n", $2, $3, $4}' \
        >> "${metrics_file}"

    # Collect network metrics
    ssh "${CONFIG[deploy_user]}@${host}" "cat /proc/net/dev" | \
        awk '/tun0/ {printf "network_rx_bytes=%d\nnetwork_tx_bytes=%d\n", $2, $10}' \
        >> "${metrics_file}"

    # Collect process metrics
    local pid
    pid=$(ssh "${CONFIG[deploy_user]}@${host}" "systemctl show -p MainPID sssonector | cut -d= -f2")
    if [[ -n "${pid}" && "${pid}" != "0" ]]; then
        ssh "${CONFIG[deploy_user]}@${host}" "ps -p ${pid} -o %cpu=,%mem= --no-headers" | \
            awk '{printf "process_cpu=%.1f\nprocess_memory=%.1f\n", $1, $2}' \
            >> "${metrics_file}"
    fi

    # Export metrics to Prometheus if enabled
    if [[ "${CONFIG[prometheus_enabled]}" == "true" && -d "${PROMETHEUS_DIR}" ]]; then
        export_to_prometheus "${metrics_file}" "${host}"
    fi

    # Export metrics to Grafana if enabled
    if [[ "${CONFIG[grafana_enabled]}" == "true" && -n "${CONFIG[grafana_api_key]}" ]]; then
        export_to_grafana "${metrics_file}" "${host}"
    fi
}

# Export metrics to Prometheus
export_to_prometheus() {
    local metrics_file=$1
    local host=$2
    local prom_file="${PROMETHEUS_DIR}/sssonector_${host}.prom"

    log_info "Exporting metrics to Prometheus for ${host}..."

    # Convert metrics to Prometheus format
    {
        echo "# HELP sssonector_cpu_usage CPU usage metrics"
        echo "# TYPE sssonector_cpu_usage gauge"
        grep "^cpu_" "${metrics_file}" | while IFS='=' read -r key value; do
            echo "sssonector_${key}{host=\"${host}\"} ${value}"
        done

        echo "# HELP sssonector_memory_usage Memory usage metrics (MB)"
        echo "# TYPE sssonector_memory_usage gauge"
        grep "^memory_" "${metrics_file}" | while IFS='=' read -r key value; do
            echo "sssonector_${key}{host=\"${host}\"} ${value}"
        done

        echo "# HELP sssonector_disk_usage Disk usage metrics (MB)"
        echo "# TYPE sssonector_disk_usage gauge"
        grep "^disk_" "${metrics_file}" | while IFS='=' read -r key value; do
            echo "sssonector_${key}{host=\"${host}\"} ${value}"
        done

        echo "# HELP sssonector_network_traffic Network traffic metrics (bytes)"
        echo "# TYPE sssonector_network_traffic counter"
        grep "^network_" "${metrics_file}" | while IFS='=' read -r key value; do
            echo "sssonector_${key}{host=\"${host}\"} ${value}"
        done

        echo "# HELP sssonector_process_metrics Process resource usage"
        echo "# TYPE sssonector_process_metrics gauge"
        grep "^process_" "${metrics_file}" | while IFS='=' read -r key value; do
            echo "sssonector_${key}{host=\"${host}\"} ${value}"
        done
    } > "${prom_file}"
}

# Export metrics to Grafana
export_to_grafana() {
    local metrics_file=$1
    local host=$2

    if [[ -z "${CONFIG[grafana_url]}" ]]; then
        return 0
    fi

    log_info "Exporting metrics to Grafana for ${host}..."

    # Prepare metrics payload
    local timestamp
    timestamp=$(date +%s)000  # Grafana expects milliseconds

    # Convert metrics to Grafana format
    local payload
    payload=$(jq -n \
        --arg host "${host}" \
        --arg timestamp "${timestamp}" \
        --arg cpu_user "$(grep '^cpu_user=' "${metrics_file}" | cut -d= -f2)" \
        --arg cpu_system "$(grep '^cpu_system=' "${metrics_file}" | cut -d= -f2)" \
        --arg memory_used "$(grep '^memory_used=' "${metrics_file}" | cut -d= -f2)" \
        --arg disk_used "$(grep '^disk_used=' "${metrics_file}" | cut -d= -f2)" \
        '{
            "host": $host,
            "timestamp": $timestamp,
            "metrics": {
                "cpu": {
                    "user": ($cpu_user | tonumber),
                    "system": ($cpu_system | tonumber)
                },
                "memory": {
                    "used": ($memory_used | tonumber)
                },
                "disk": {
                    "used": ($disk_used | tonumber)
                }
            }
        }'
    )

    # Send metrics to Grafana
    curl -s -X POST \
        -H "Content-Type: application/json" \
        -H "Authorization: Bearer ${CONFIG[grafana_api_key]}" \
        -d "${payload}" \
        "${CONFIG[grafana_url]}/api/datasources/proxy/1/metrics" || true
}

# Cleanup old metrics
cleanup_metrics() {
    log_info "Cleaning up old metrics..."

    # Remove metrics files older than retention period
    find "${METRICS_DIR}" -name "*.metrics" -mtime +"${CONFIG[retention_days]}" -delete

    # Remove old Prometheus files
    if [[ "${CONFIG[prometheus_enabled]}" == "true" && -d "${PROMETHEUS_DIR}" ]]; then
        find "${PROMETHEUS_DIR}" -name "sssonector_*.prom" -mtime +"${CONFIG[retention_days]}" -delete
    fi
}

# Main monitoring function
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
    exec 1> >(tee -a "${MONITOR_LOG}")
    exec 2>&1

    # Load configuration
    if ! load_config "${config_file}"; then
        log_error "Failed to load configuration"
        exit 1
    fi

    # Split target hosts
    IFS=',' read -ra hosts <<< "${CONFIG[target_hosts]}"

    if [[ "${continuous}" == "true" ]]; then
        log_info "Starting continuous monitoring..."
        while true; do
            for host in "${hosts[@]}"; do
                collect_system_metrics "${host}"
            done
            cleanup_metrics
            sleep "${CONFIG[collect_interval]}"
        done
    else
        log_info "Running one-time metrics collection..."
        for host in "${hosts[@]}"; do
            collect_system_metrics "${host}"
        done
        cleanup_metrics
    fi
}

# Execute main function
main "$@"
