#!/bin/bash
#
# SSSonector Uninstall Script
# Removes SSSonector from the system
#
# Usage:
#   sudo ./uninstall.sh [options]
#
# Options:
#   --purge    Remove all configuration and data files
#   --all      Remove all instances (stops them first)
#   -y         Don't ask for confirmation
#

set -e

INSTALL_DIR="/usr/local/bin"
CONFIG_DIR="/etc/sssonector"
INSTANCE_DIR="/etc/sssonector/instances"
LOG_DIR="/var/log/sssonector"
SERVICE_FILE="/etc/systemd/system/sssonector@.service"

RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

PURGE=false
REMOVE_ALL=false
NO_CONFIRM=false

log_info() { echo -e "${GREEN}[INFO]${NC} $1"; }
log_warn() { echo -e "${YELLOW}[WARN]${NC} $1"; }
log_error() { echo -e "${RED}[ERROR]${NC} $1"; }
log_step() { echo -e "${BLUE}==>${NC} $1"; }

check_root() {
    if [ "$EUID" -ne 0 ]; then
        log_error "This script must be run as root"
        exit 1
    fi
}

parse_args() {
    while [[ $# -gt 0 ]]; do
        case $1 in
            --purge) PURGE=true ;;
            --all) REMOVE_ALL=true ;;
            -y|--yes) NO_CONFIRM=true ;;
            *) log_error "Unknown option: $1"; exit 1 ;;
        esac
        shift
    done
}

list_instances() {
    local instances=()
    if [ -d "$INSTANCE_DIR" ]; then
        for dir in "$INSTANCE_DIR"/*/; do
            if [ -d "$dir" ] && [ -f "$dir/config.yaml" ]; then
                instances+=("$(basename "$dir")")
            fi
        done
    fi
    echo "${instances[@]}"
}

stop_all_services() {
    log_step "Stopping all SSSonector services..."
    
    local instances
    instances=$(list_instances)
    
    for instance in $instances; do
        if systemctl is-active "sssonector@${instance}" &>/dev/null; then
            log_info "Stopping sssonector@${instance}..."
            systemctl stop "sssonector@${instance}" || true
        fi
        if systemctl is-enabled "sssonector@${instance}" &>/dev/null; then
            systemctl disable "sssonector@${instance}" || true
        fi
    done
}

remove_service_template() {
    if [ -f "$SERVICE_FILE" ]; then
        log_step "Removing systemd service template..."
        rm -f "$SERVICE_FILE"
        systemctl daemon-reload
        log_info "Systemd service template removed"
    fi
}

remove_binary() {
    if [ -f "${INSTALL_DIR}/sssonector" ]; then
        log_step "Removing binary..."
        rm -f "${INSTALL_DIR}/sssonector"
        log_info "Binary removed"
    fi
}

remove_instances() {
    if [ -d "$INSTANCE_DIR" ]; then
        local instances
        instances=$(list_instances)
        
        if [ -n "$instances" ]; then
            log_step "Removing instance configurations..."
            
            if [ "$REMOVE_ALL" = true ]; then
                for instance in $instances; do
                    log_info "Removing instance: $instance"
                    rm -rf "${INSTANCE_DIR}/${instance}"
                done
            else
                log_warn "The following instances exist:"
                for instance in $instances; do
                    echo "  - $instance"
                done
                echo
                echo "Use --all to remove all instances, or --purge to remove everything."
                echo "Or manually remove instances:"
                echo "  rm -rf ${INSTANCE_DIR}/<instance-name>"
            fi
        fi
    fi
}

purge_all() {
    log_step "Purging all SSSonector data..."
    
    if [ -d "$CONFIG_DIR" ]; then
        rm -rf "$CONFIG_DIR"
        log_info "Removed $CONFIG_DIR"
    fi
    
    if [ -d "$LOG_DIR" ]; then
        rm -rf "$LOG_DIR"
        log_info "Removed $LOG_DIR"
    fi
    
    if [ -f "/etc/logrotate.d/sssonector" ]; then
        rm -f "/etc/logrotate.d/sssonector"
        log_info "Removed logrotate config"
    fi
}

confirm() {
    if [ "$NO_CONFIRM" = true ]; then
        return 0
    fi
    
    read -p "$1 [y/N]: " response
    case "$response" in
        [yY][eE][sS]|[yY]) return 0 ;;
        *) return 1 ;;
    esac
}

main() {
    check_root
    parse_args "$@"
    
    log_step "SSSonector Uninstaller"
    echo
    
    if [ "$PURGE" = true ]; then
        log_warn "This will remove ALL SSSonector data including:"
        echo "  - Binary: ${INSTALL_DIR}/sssonector"
        echo "  - Configuration: $CONFIG_DIR"
        echo "  - Logs: $LOG_DIR"
        echo "  - Systemd service"
        echo
        
        if ! confirm "Proceed with complete removal?"; then
            log_info "Cancelled"
            exit 0
        fi
        
        stop_all_services
        remove_service_template
        remove_binary
        purge_all
        
    else
        log_warn "This will remove the SSSonector binary and systemd service."
        echo "Instance configurations will be preserved."
        echo "Use --purge to remove everything."
        echo
        
        if ! confirm "Proceed?"; then
            log_info "Cancelled"
            exit 0
        fi
        
        stop_all_services
        remove_service_template
        remove_binary
        remove_instances
    fi
    
    echo
    log_info "Uninstall complete!"
    
    if [ "$PURGE" = false ]; then
        echo
        echo "To completely remove all data, run:"
        echo "  $0 --purge"
    fi
}

main "$@"
