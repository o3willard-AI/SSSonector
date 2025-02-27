#!/bin/bash

# network_safety.sh
# Network safety script for SSSonector local development testing
set -euo pipefail

# Color codes for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m'

# Global variables
SNAPSHOT_DIR="network_snapshots"
CURRENT_SNAPSHOT="${SNAPSHOT_DIR}/$(date +%Y%m%d_%H%M%S)"
TIMEOUT=300  # 5 minutes
WATCHDOG_INTERVAL=10
SSH_PORT=22
MAIN_INTERFACE=$(ip route | awk '/default/ {print $5}' | head -n1)

# Create snapshot directory structure
mkdir -p "${CURRENT_SNAPSHOT}"

# Logging functions
log_info() {
    echo -e "${GREEN}[INFO]${NC} $1" | tee -a "${CURRENT_SNAPSHOT}/safety.log"
}

log_warn() {
    echo -e "${YELLOW}[WARN]${NC} $1" | tee -a "${CURRENT_SNAPSHOT}/safety.log"
}

log_error() {
    echo -e "${RED}[ERROR]${NC} $1" | tee -a "${CURRENT_SNAPSHOT}/safety.log"
    return 1
}

# Create network snapshot
create_snapshot() {
    local snapshot_dir=$1

    log_info "Creating network snapshot in ${snapshot_dir}"

    # Save routing table
    ip route show > "${snapshot_dir}/routes.txt"

    # Save interface configurations
    ip addr show > "${snapshot_dir}/interfaces.txt"

    # Save active connections
    ss -tuln > "${snapshot_dir}/connections.txt"

    # Save DNS settings
    cp /etc/resolv.conf "${snapshot_dir}/resolv.conf"

    # Save iptables rules
    iptables-save > "${snapshot_dir}/iptables.txt"

    # Save network statistics
    ip -s link > "${snapshot_dir}/link_statistics.txt"

    log_info "Network snapshot created successfully"
}

# Verify network connectivity
verify_connectivity() {
    local failed=0

    # Check default route
    if ! ip route show | grep -q '^default'; then
        log_error "Default route missing"
        failed=1
    fi

    # Check main interface
    if ! ip link show "${MAIN_INTERFACE}" | grep -q 'UP'; then
        log_error "Main interface ${MAIN_INTERFACE} is down"
        failed=1
    fi

    # Check DNS resolution
    if ! dig +short +timeout=2 +tries=1 google.com >/dev/null; then
        log_warn "DNS resolution failing"
        failed=1
    fi

    # Check internet connectivity
    if ! ping -c 1 -W 2 8.8.8.8 >/dev/null 2>&1; then
        log_error "Internet connectivity lost"
        failed=1
    fi

    return ${failed}
}

# Restore network configuration
restore_snapshot() {
    local snapshot_dir=$1
    log_info "Restoring network configuration from ${snapshot_dir}"

    # Restore iptables rules
    iptables-restore < "${snapshot_dir}/iptables.txt"

    # Restore routes (excluding default route to prevent lockout)
    while read -r route; do
        if [[ ! "${route}" =~ ^default ]]; then
            ip route add ${route} 2>/dev/null || true
        fi
    done < "${snapshot_dir}/routes.txt"

    # Restore default route last
    grep '^default' "${snapshot_dir}/routes.txt" | while read -r route; do
        ip route add ${route} 2>/dev/null || true
    done

    # Restore DNS settings
    cp "${snapshot_dir}/resolv.conf" /etc/resolv.conf

    log_info "Network configuration restored"
}

# Network watchdog
start_watchdog() {
    local pid_file="/tmp/network_watchdog.pid"
    local snapshot_dir=$1

    # Start watchdog process
    (
        echo $$ > "${pid_file}"
        local start_time=$(date +%s)
        local fails=0

        while true; do
            # Check if we've exceeded timeout
            local current_time=$(date +%s)
            local elapsed=$((current_time - start_time))
            
            if [ ${elapsed} -ge ${TIMEOUT} ]; then
                log_info "Watchdog timeout reached, exiting normally"
                exit 0
            fi

            # Verify network connectivity
            if ! verify_connectivity; then
                ((fails++))
                log_warn "Network verification failed (${fails}/3)"
                
                if [ ${fails} -ge 3 ]; then
                    log_error "Network verification failed 3 times, initiating rollback"
                    restore_snapshot "${snapshot_dir}"
                    exit 1
                fi
            else
                fails=0
            fi

            sleep ${WATCHDOG_INTERVAL}
        done
    ) &

    log_info "Network watchdog started with PID $(cat ${pid_file})"
}

# Stop watchdog
stop_watchdog() {
    local pid_file="/tmp/network_watchdog.pid"
    if [ -f "${pid_file}" ]; then
        kill $(cat "${pid_file}") 2>/dev/null || true
        rm -f "${pid_file}"
        log_info "Network watchdog stopped"
    fi
}

# Cleanup function
cleanup() {
    log_info "Cleaning up..."
    stop_watchdog
}

# Main function
main() {
    # Register cleanup handler
    trap cleanup EXIT

    # Create current snapshot
    create_snapshot "${CURRENT_SNAPSHOT}"

    # Start network watchdog
    start_watchdog "${CURRENT_SNAPSHOT}"

    # Verify initial connectivity
    if ! verify_connectivity; then
        log_error "Initial network verification failed"
        exit 1
    fi

    log_info "Network safety measures initialized"
    log_info "Snapshot location: ${CURRENT_SNAPSHOT}"
    log_info "Watchdog timeout: ${TIMEOUT} seconds"
    log_info "Main interface: ${MAIN_INTERFACE}"
}

# Run main function if script is executed directly
if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then
    main "$@"
fi
