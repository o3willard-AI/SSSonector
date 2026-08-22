#!/bin/bash
#
# SSSonector Upgrade Script
# Upgrades SSSonector to a newer version
#
# Usage:
#   curl -fsSL https://raw.githubusercontent.com/o3willard-AI/SSSonector/main/upgrade.sh | sudo bash
#
# Options:
#   --version VERSION    Upgrade to specific version (default: latest)
#   --no-restart         Don't restart services after upgrade
#   -y                   Don't ask for confirmation
#

set -e

REPO="o3willard-AI/SSSonector"
INSTALL_DIR="/usr/local/bin"
INSTANCE_DIR="/etc/sssonector/instances"
BINARY="${INSTALL_DIR}/sssonector"

RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

TARGET_VERSION=""
NO_RESTART=false
NO_CONFIRM=false
RESTART_INSTANCES=()

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
            --version) TARGET_VERSION="$2"; shift 2 ;;
            --no-restart) NO_RESTART=true; shift ;;
            -y|--yes) NO_CONFIRM=true; shift ;;
            *) log_error "Unknown option: $1"; exit 1 ;;
        esac
    done
}

detect_os() {
    case "$(uname -s)" in
        Linux*)  echo "linux" ;;
        Darwin*) echo "darwin" ;;
        *)       log_error "Unsupported OS: $(uname -s)"; exit 1 ;;
    esac
}

detect_arch() {
    local arch=$(uname -m)
    case "$arch" in
        x86_64|amd64) echo "amd64" ;;
        aarch64|arm64) echo "arm64" ;;
        armv7l)        echo "arm" ;;
        *)             log_error "Unsupported architecture: $arch"; exit 1 ;;
    esac
}

get_current_version() {
    if [ -x "$BINARY" ]; then
        $BINARY -version 2>/dev/null || echo "unknown"
    else
        echo "not installed"
    fi
}

get_latest_version() {
    local version
    version=$(curl -fsSL "https://api.github.com/repos/${REPO}/releases/latest" 2>/dev/null | grep '"tag_name":' | sed -E 's/.*"([^"]+)".*/\1/')
    if [ -z "$version" ]; then
        log_error "Failed to fetch latest version from GitHub"
        exit 1
    fi
    echo "$version"
}

list_running_instances() {
    local instances=()
    if [ -d "$INSTANCE_DIR" ]; then
        for dir in "$INSTANCE_DIR"/*/; do
            if [ -d "$dir" ] && [ -f "$dir/config.yaml" ]; then
                local name=$(basename "$dir")
                if systemctl is-active "sssonector@${name}" &>/dev/null; then
                    instances+=("$name")
                fi
            fi
        done
    fi
    echo "${instances[@]}"
}

stop_instances() {
    local instances="$1"
    for instance in $instances; do
        log_info "Stopping sssonector@${instance}..."
        systemctl stop "sssonector@${instance}" || true
    done
}

start_instances() {
    local instances="$1"
    for instance in $instances; do
        log_info "Starting sssonector@${instance}..."
        systemctl start "sssonector@${instance}" || log_warn "Failed to start sssonector@${instance}"
    done
}

download_binary() {
    local os=$1
    local arch=$2
    local version=$3

    local binary_name="sssonector-${os}-${arch}"
    local base_url="https://github.com/${REPO}/releases/download/${version}"
    local download_url="${base_url}/${binary_name}"
    local sums_url="${base_url}/SHA256SUMS"
    local tmp_file="/tmp/sssonector-$$"
    local tmp_sums="/tmp/sssonector-SHA256SUMS-$$"

    log_step "Downloading SSSonector ${version} for ${os}/${arch}..."

    if ! curl -fsSL "$sums_url" -o "$tmp_sums"; then
        log_error "Failed to download SHA256SUMS from $sums_url"
        exit 1
    fi

    if ! curl -fsSL "$download_url" -o "$tmp_file"; then
        log_error "Failed to download binary from $download_url"
        exit 1
    fi

    log_step "Verifying binary checksum..."
    local expected
    expected=$(grep " ${binary_name}\$" "$tmp_sums" | awk '{print $1}')
    if [ -z "$expected" ]; then
        log_error "No checksum found for ${binary_name} in SHA256SUMS"
        rm -f "$tmp_file" "$tmp_sums"
        exit 1
    fi

    local actual
    if command -v sha256sum >/dev/null 2>&1; then
        actual=$(sha256sum "$tmp_file" | awk '{print $1}')
    elif command -v shasum >/dev/null 2>&1; then
        actual=$(shasum -a 256 "$tmp_file" | awk '{print $1}')
    else
        log_error "No sha256sum or shasum available for verification"
        rm -f "$tmp_file" "$tmp_sums"
        exit 1
    fi

    if [ "$actual" != "$expected" ]; then
        log_error "Checksum mismatch: expected ${expected}, got ${actual}. Upgrade aborted."
        rm -f "$tmp_file" "$tmp_sums"
        exit 1
    fi
    rm -f "$tmp_sums"

    chmod +x "$tmp_file"
    echo "$tmp_file"
}

upgrade_binary() {
    local tmp_file=$1
    
    log_step "Upgrading binary..."
    
    mv "$tmp_file" "${BINARY}.new"
    mv "${BINARY}.new" "$BINARY"
    
    log_info "Binary upgraded to ${INSTALL_DIR}/sssonector"
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
    
    log_step "SSSonector Upgrade"
    echo
    
    if [ ! -x "$BINARY" ]; then
        log_error "SSSonector is not installed. Run install.sh first."
        exit 1
    fi
    
    local current_version
    current_version=$(get_current_version)
    
    if [ -z "$TARGET_VERSION" ]; then
        TARGET_VERSION=$(get_latest_version)
    fi
    
    log_info "Current version: $current_version"
    log_info "Target version:  $TARGET_VERSION"
    echo
    
    if [ "$current_version" = "$TARGET_VERSION" ]; then
        log_info "Already running the target version. Nothing to do."
        exit 0
    fi
    
    local running_instances
    running_instances=$(list_running_instances)
    
    if [ -n "$running_instances" ]; then
        log_info "Running instances: $running_instances"
        
        if [ "$NO_RESTART" = false ]; then
            log_warn "Services will be restarted after upgrade"
        fi
    fi
    
    if ! confirm "Proceed with upgrade?"; then
        log_info "Cancelled"
        exit 0
    fi
    
    if [ -n "$running_instances" ]; then
        log_step "Stopping running instances..."
        stop_instances "$running_instances"
    fi
    
    local os arch
    os=$(detect_os)
    arch=$(detect_arch)
    
    local tmp_binary
    tmp_binary=$(download_binary "$os" "$arch" "$TARGET_VERSION")
    
    upgrade_binary "$tmp_binary"
    
    if [ -n "$running_instances" ] && [ "$NO_RESTART" = false ]; then
        log_step "Restarting instances..."
        start_instances "$running_instances"
    fi
    
    echo
    log_info "Upgrade complete!"
    log_info "Version: $($BINARY -version 2>/dev/null || echo "unknown")"
    
    if [ "$NO_RESTART" = true ] && [ -n "$running_instances" ]; then
        echo
        echo "Services were not restarted. Restart with:"
        echo "  systemctl restart sssonector@<instance>"
    fi
}

main "$@"
