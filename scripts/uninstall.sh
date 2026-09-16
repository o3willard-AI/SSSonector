#!/bin/bash
#
# SSSonector Uninstall Script
# Removes SSSonector and everything install.sh creates:
#   - Service (systemd on Linux, launchd on macOS)
#   - Binary (/usr/local/bin/sssonector)
#   - Config directory (/etc/sssonector)
#   - Log directory (/var/log/sssonector)
#
# Usage:
#   curl -fsSL https://raw.githubusercontent.com/o3willard-AI/SSSonector/main/scripts/uninstall.sh | sudo bash
#
# Environment variables:
#   SSSONECTOR_INSTANCE    - Remove only this instance (default: remove all)
#   SSSONECTOR_INSTALL_DIR - Override binary install directory
#   SSSONECTOR_CONFIG_DIR  - Override config directory
#   SSSONECTOR_LOG_DIR     - Override log directory
#   SSSONECTOR_LAUNCHD_DIR - Override launchd daemon directory (macOS)
#   SSSONECTOR_SYSTEMD_DIR - Override systemd unit directory (Linux)
#
set -e

REPO="o3willard-AI/SSSonector"
INSTALL_DIR="${SSSONECTOR_INSTALL_DIR:-/usr/local/bin}"
CONFIG_DIR="${SSSONECTOR_CONFIG_DIR:-/etc/sssonector}"
INSTANCE_DIR="${SSSONECTOR_INSTANCE_DIR:-/etc/sssonector/instances}"
LOG_DIR="${SSSONECTOR_LOG_DIR:-/var/log/sssonector}"

RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

log_info() { echo -e "${GREEN}[INFO]${NC} $1" >&2; }
log_warn() { echo -e "${YELLOW}[WARN]${NC} $1" >&2; }
log_error() { echo -e "${RED}[ERROR]${NC} $1" >&2; }
log_step() { echo -e "${BLUE}==>${NC} $1" >&2; }

check_root() {
    if [ "$EUID" -ne 0 ]; then
        log_error "This script must be run as root"
        exit 1
    fi
}

detect_os() {
    case "$(uname -s)" in
        Linux*)  echo "linux" ;;
        Darwin*) echo "darwin" ;;
        *)       log_error "Unsupported OS: $(uname -s)"; exit 1 ;;
    esac
}

# ---------------------------------------------------------------------------
# Service removal
# ---------------------------------------------------------------------------

remove_systemd_service() {
    local systemd_dir="${SSSONECTOR_SYSTEMD_DIR:-/etc/systemd/system}"
    local service_file="${systemd_dir}/sssonector@.service"
    local instance="$SSSONECTOR_INSTANCE"

    if [ -n "$instance" ]; then
        log_step "Stopping and disabling sssonector@${instance}..."
        systemctl stop "sssonector@${instance}" 2>/dev/null || true
        systemctl disable "sssonector@${instance}" 2>/dev/null || true
    else
        log_step "Stopping and disabling all sssonector instances..."
        systemctl stop 'sssonector@*' 2>/dev/null || true
        systemctl disable 'sssonector@*' 2>/dev/null || true
    fi

    if [ -f "$service_file" ]; then
        log_step "Removing systemd service template..."
        rm -f "$service_file"
        systemctl daemon-reload 2>/dev/null || true
        log_info "Systemd service template removed"
    else
        log_info "No systemd service template found (already clean)"
    fi
}

remove_launchd_service() {
    local launchd_dir="${SSSONECTOR_LAUNCHD_DIR:-/Library/LaunchDaemons}"
    local instance="$SSSONECTOR_INSTANCE"

    if [ -n "$instance" ]; then
        local plist_path="${launchd_dir}/com.o3willard.sssonector.${instance}.plist"
        if [ -f "$plist_path" ]; then
            log_step "Unloading launchd service for instance '${instance}'..."
            launchctl unload "$plist_path" 2>/dev/null || true
            rm -f "$plist_path"
            log_info "Removed plist: ${plist_path}"
        else
            log_info "No plist found for instance '${instance}' (already clean)"
        fi
    else
        log_step "Unloading all sssonector launchd services..."
        if [ -d "$launchd_dir" ]; then
            for plist in "$launchd_dir"/com.o3willard.sssonector.*.plist; do
                [ -f "$plist" ] || continue
                log_info "Unloading: $(basename "$plist")"
                launchctl unload "$plist" 2>/dev/null || true
                rm -f "$plist"
                log_info "Removed plist: $(basename "$plist")"
            done
        else
            log_info "No launchd directory found: ${launchd_dir} (already clean)"
        fi
    fi
}

# ---------------------------------------------------------------------------
# Artifact removal
# ---------------------------------------------------------------------------

remove_binary() {
    local binary="${INSTALL_DIR}/sssonector"
    if [ -f "$binary" ]; then
        log_step "Removing binary..."
        rm -f "$binary"
        log_info "Binary removed: ${binary}"
    else
        log_info "Binary not found: ${binary} (already clean)"
    fi
}

remove_config() {
    local instance="$SSSONECTOR_INSTANCE"
    if [ -n "$instance" ]; then
        local instance_path="${INSTANCE_DIR}/${instance}"
        if [ -d "$instance_path" ]; then
            log_step "Removing instance '${instance}'..."
            rm -rf "$instance_path"
            log_info "Instance removed: ${instance_path}"
        else
            log_info "Instance not found: ${instance_path} (already clean)"
        fi
    else
        if [ -d "$CONFIG_DIR" ]; then
            log_step "Removing configuration directory..."
            rm -rf "$CONFIG_DIR"
            log_info "Configuration directory removed: ${CONFIG_DIR}"
        else
            log_info "Configuration directory not found: ${CONFIG_DIR} (already clean)"
        fi
    fi
}

remove_logs() {
    if [ -d "$LOG_DIR" ]; then
        log_step "Removing log directory..."
        rm -rf "$LOG_DIR"
        log_info "Log directory removed: ${LOG_DIR}"
    else
        log_info "Log directory not found: ${LOG_DIR} (already clean)"
    fi
}

# ---------------------------------------------------------------------------
# Main
# ---------------------------------------------------------------------------

main() {
    check_root

    echo
    log_step "SSSonector Uninstaller"
    echo

    local os
    os=$(detect_os)

    # Remove service manager unit
    if [ "$os" = "darwin" ]; then
        remove_launchd_service
    else
        if command -v systemctl >/dev/null 2>&1; then
            remove_systemd_service
        else
            log_warn "systemctl not found; skipping service removal"
        fi
    fi

    # Remove artifacts
    remove_binary
    remove_config
    remove_logs

    echo
    log_info "Uninstall complete!"
    echo "If you need further assistance, visit: https://github.com/${REPO}/issues"
}

# Only run main when executed directly, not when sourced (e.g. by tests)
if [ "${BASH_SOURCE[0]}" = "${0}" ]; then
    main "$@"
fi
