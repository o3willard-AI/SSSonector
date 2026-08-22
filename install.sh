#!/bin/bash
#
# SSSonector Install Script
# Downloads and installs SSSonector from GitHub releases
#
# Usage:
#   curl -fsSL https://raw.githubusercontent.com/o3willard-AI/SSSonector/main/install.sh | sudo bash
#
# Environment variables:
#   SSSONECTOR_VERSION    - Version to install (default: latest)
#   SSSONECTOR_MODE       - "server" or "client" (interactive if not set)
#   SSSONECTOR_INSTANCE   - Instance name (interactive if not set)
#   SSSONECTOR_ADDRESS    - TUN interface address (interactive if not set)
#   SSSONECTOR_SERVER     - Server address for client mode (interactive if not set)
#   SSSONECTOR_PORT       - Listen/connect port (default: 8443)
#   SSSONECTOR_NO_SERVICE - If set, don't install/enabled systemd service
#

set -e

REPO="o3willard-AI/SSSonector"
INSTALL_DIR="/usr/local/bin"
CONFIG_DIR="/etc/sssonector"
INSTANCE_DIR="/etc/sssonector/instances"
LOG_DIR="/var/log/sssonector"
TEMPLATE_URL="https://raw.githubusercontent.com/${REPO}/main/templates"

RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

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

get_latest_version() {
    local version
    version=$(curl -fsSL "https://api.github.com/repos/${REPO}/releases/latest" 2>/dev/null | grep '"tag_name":' | sed -E 's/.*"([^"]+)".*/\1/')
    if [ -z "$version" ]; then
        log_error "Failed to fetch latest version from GitHub"
        exit 1
    fi
    echo "$version"
}

download_binary() {
    local os=$1
    local arch=$2
    local version=$3

    local binary_name="sssonector-${os}-${arch}"
    local base_url="https://github.com/${REPO}/releases/download/${version}"
    local download_url="${base_url}/${binary_name}"
    local sums_url="${base_url}/SHA256SUMS"
    local tmp_file="/tmp/sssonector"
    local tmp_sums="/tmp/sssonector-SHA256SUMS"

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
        log_error "Checksum mismatch: expected ${expected}, got ${actual}. Binary NOT installed."
        rm -f "$tmp_file" "$tmp_sums"
        exit 1
    fi
    rm -f "$tmp_sums"

    chmod +x "$tmp_file"
    echo "$tmp_file"
}

install_binary() {
    local tmp_file=$1
    
    log_step "Installing binary to ${INSTALL_DIR}..."
    
    mkdir -p "$INSTALL_DIR"
    install -m 755 "$tmp_file" "${INSTALL_DIR}/sssonector"
    rm -f "$tmp_file"
    
    log_info "Binary installed: ${INSTALL_DIR}/sssonector"
}

create_directories() {
    log_step "Creating directories..."
    
    mkdir -p "$CONFIG_DIR"
    mkdir -p "$INSTANCE_DIR"
    mkdir -p "$LOG_DIR"
    
    log_info "Directories created"
}

download_templates() {
    log_step "Downloading configuration templates..."
    
    local tmp_dir="/tmp/sssonector-templates"
    mkdir -p "$tmp_dir"
    
    if ! curl -fsSL "${TEMPLATE_URL}/server.yaml.template" -o "${tmp_dir}/server.yaml.template"; then
        log_error "Failed to download server template"
        exit 1
    fi
    
    if ! curl -fsSL "${TEMPLATE_URL}/client.yaml.template" -o "${tmp_dir}/client.yaml.template"; then
        log_error "Failed to download client template"
        exit 1
    fi
    
    echo "$tmp_dir"
}

generate_certificates() {
    local instance_name=$1
    local cert_dir="$INSTANCE_DIR/$instance_name/certs"
    
    log_step "Generating certificates..."
    
    mkdir -p "$cert_dir"
    
    openssl genrsa -out "$cert_dir/ca.key" 4096 2>/dev/null
    openssl req -new -x509 -days 365 -key "$cert_dir/ca.key" \
        -out "$cert_dir/ca.crt" \
        -subj "/C=US/ST=State/L=City/O=SSSonector/OU=CA/CN=SSSonector-CA" 2>/dev/null
    
    openssl genrsa -out "$cert_dir/server.key" 4096 2>/dev/null
    openssl req -new -key "$cert_dir/server.key" \
        -out "$cert_dir/server.csr" \
        -subj "/C=US/ST=State/L=City/O=SSSonector/OU=Server/CN=server" 2>/dev/null
    openssl x509 -req -days 365 -in "$cert_dir/server.csr" \
        -CA "$cert_dir/ca.crt" -CAkey "$cert_dir/ca.key" \
        -CAcreateserial -out "$cert_dir/server.crt" 2>/dev/null
    
    openssl genrsa -out "$cert_dir/client.key" 4096 2>/dev/null
    openssl req -new -key "$cert_dir/client.key" \
        -out "$cert_dir/client.csr" \
        -subj "/C=US/ST=State/L=City/O=SSSonector/OU=Client/CN=client" 2>/dev/null
    openssl x509 -req -days 365 -in "$cert_dir/client.csr" \
        -CA "$cert_dir/ca.crt" -CAkey "$cert_dir/ca.key" \
        -CAcreateserial -out "$cert_dir/client.crt" 2>/dev/null
    
    chmod 600 "$cert_dir"/*.key
    chmod 644 "$cert_dir"/*.crt
    rm -f "$cert_dir"/*.csr "$cert_dir"/*.srl
    
    log_info "Certificates generated in $cert_dir"
}

get_next_port() {
    local base_port=${1:-8443}
    local max_port=$base_port
    
    if [ -d "$INSTANCE_DIR" ]; then
        for dir in "$INSTANCE_DIR"/*/; do
            if [ -f "$dir/config.yaml" ]; then
                port=$(grep -oP 'listen_port:\s*\K\d+' "$dir/config.yaml" 2>/dev/null)
                if [ -n "$port" ] && [ "$port" -ge "$max_port" ]; then
                    max_port=$((port + 1))
                fi
            fi
        done
    fi
    echo $max_port
}

get_next_prometheus_port() {
    local base_port=9090
    local max_port=$base_port
    
    if [ -d "$INSTANCE_DIR" ]; then
        for dir in "$INSTANCE_DIR"/*/; do
            if [ -f "$dir/config.yaml" ]; then
                port=$(grep -oP 'port:\s*\K\d+' "$dir/config.yaml" 2>/dev/null | tail -1)
                if [ -n "$port" ] && [ "$port" -ge "$max_port" ]; then
                    max_port=$((port + 1))
                fi
            fi
        done
    fi
    echo $max_port
}

get_next_instance_number() {
    local max_num=0
    if [ -d "$INSTANCE_DIR" ]; then
        for dir in "$INSTANCE_DIR"/*/; do
            if [ -d "$dir" ] && [ -f "$dir/config.yaml" ]; then
                num=$(grep -oP 'tun\d+' "$dir/config.yaml" 2>/dev/null | grep -oP '\d+' | head -1)
                if [ -n "$num" ] && [ "$num" -gt "$max_num" ]; then
                    max_num=$num
                fi
            fi
        done
    fi
    echo $((max_num + 1))
}

create_server_config() {
    local name=$1
    local tun=$2
    local address=$3
    local port=$4
    local prom_port=$5
    local rate=${6:-1048576}
    local burst=${7:-2097152}
    local template_dir=$8
    
    local instance_path="$INSTANCE_DIR/$name"
    
    if [ -d "$instance_path" ]; then
        log_error "Instance '$name' already exists"
        exit 1
    fi
    
    mkdir -p "$instance_path"
    
    generate_certificates "$name"
    
    sed -e "s/{{INSTANCE_NAME}}/$name/g" \
        -e "s/{{TUN_INTERFACE}}/$tun/g" \
        -e "s/{{TUN_ADDRESS}}/$address/g" \
        -e "s/{{LISTEN_PORT}}/$port/g" \
        -e "s/{{PROMETHEUS_PORT}}/$prom_port/g" \
        -e "s/{{RATE_LIMIT}}/$rate/g" \
        -e "s/{{RATE_BURST}}/$burst/g" \
        "${template_dir}/server.yaml.template" > "$instance_path/config.yaml"
    
    log_info "Server instance created: $instance_path"
}

create_client_config() {
    local name=$1
    local tun=$2
    local address=$3
    local server=$4
    local server_port=$5
    local prom_port=$6
    local rate=${7:-1048576}
    local burst=${8:-2097152}
    local template_dir=$9
    
    local instance_path="$INSTANCE_DIR/$name"
    
    if [ -d "$instance_path" ]; then
        log_error "Instance '$name' already exists"
        exit 1
    fi
    
    mkdir -p "$instance_path"
    
    generate_certificates "$name"
    
    sed -e "s/{{INSTANCE_NAME}}/$name/g" \
        -e "s/{{TUN_INTERFACE}}/$tun/g" \
        -e "s/{{TUN_ADDRESS}}/$address/g" \
        -e "s/{{SERVER_ADDRESS}}/$server/g" \
        -e "s/{{SERVER_PORT}}/$server_port/g" \
        -e "s/{{PROMETHEUS_PORT}}/$prom_port/g" \
        -e "s/{{RATE_LIMIT}}/$rate/g" \
        -e "s/{{RATE_BURST}}/$burst/g" \
        "${template_dir}/client.yaml.template" > "$instance_path/config.yaml"
    
    log_info "Client instance created: $instance_path"
}

install_systemd_service() {
    local service_file="/etc/systemd/system/sssonector@.service"
    
    log_step "Installing systemd service template..."
    
    cat > "$service_file" << 'EOF'
[Unit]
Description=SSSonector Tunnel Instance %i
Documentation=https://github.com/o3willard-AI/SSSonector
After=network.target
Wants=network-online.target

[Service]
Type=simple
User=root
Group=root
ExecStart=/usr/local/bin/sssonector -config /etc/sssonector/instances/%i/config.yaml
ExecReload=/bin/kill -HUP $MAINPID
Restart=on-failure
RestartSec=5
TimeoutStartSec=30
TimeoutStopSec=30

NoNewPrivileges=true
PrivateTmp=true
ProtectSystem=strict
ProtectHome=true
ReadWritePaths=/var/log/sssonector /var/run

LimitNOFILE=65535
LimitNPROC=4096

StandardOutput=journal
StandardError=journal
SyslogIdentifier=sssonector-%i

[Install]
WantedBy=multi-user.target
EOF
    
    systemctl daemon-reload
    log_info "Systemd service template installed"
}

prompt_input() {
    local prompt=$1
    local default=$2
    local var
    
    if [ -n "$default" ]; then
        prompt="$prompt [$default]"
    fi
    
    read -p "$prompt: " var
    echo "${var:-$default}"
}

interactive_setup() {
    local template_dir=$1
    
    echo
    log_step "SSSonector Setup"
    echo
    
    if [ -z "$SSSONECTOR_MODE" ]; then
        echo "Select mode:"
        echo "  1) Server - Listen for incoming tunnel connections"
        echo "  2) Client - Connect to a remote server"
        echo
        read -p "Mode [1/2]: " mode_choice
        
        case "$mode_choice" in
            1) SSSONECTOR_MODE="server" ;;
            2) SSSONECTOR_MODE="client" ;;
            *) log_error "Invalid choice"; exit 1 ;;
        esac
    fi
    
    local instance_name
    if [ -n "$SSSONECTOR_INSTANCE" ]; then
        instance_name="$SSSONECTOR_INSTANCE"
    else
        instance_name=$(prompt_input "Instance name" "tunnel-$(date +%s)")
    fi
    
    local tun_num
    tun_num=$(get_next_instance_number)
    
    local tun_address
    if [ -n "$SSSONECTOR_ADDRESS" ]; then
        tun_address="$SSSONECTOR_ADDRESS"
    else
        if [ "$SSSONECTOR_MODE" = "server" ]; then
            tun_address=$(prompt_input "TUN interface address (CIDR)" "10.0.${tun_num}.1/24")
        else
            tun_address=$(prompt_input "TUN interface address (CIDR)" "10.0.${tun_num}.2/24")
        fi
    fi
    
    local tun_interface="tun${tun_num}"
    local port
    if [ -n "$SSSONECTOR_PORT" ]; then
        port="$SSSONECTOR_PORT"
    else
        port=$(get_next_port)
    fi
    
    local prom_port
    prom_port=$(get_next_prometheus_port)
    
    local server_addr=""
    if [ "$SSSONECTOR_MODE" = "client" ]; then
        if [ -n "$SSSONECTOR_SERVER" ]; then
            server_addr="$SSSONECTOR_SERVER"
        else
            server_addr=$(prompt_input "Server address" "")
            if [ -z "$server_addr" ]; then
                log_error "Server address is required for client mode"
                exit 1
            fi
        fi
    fi
    
    echo
    echo "Configuration:"
    echo "  Mode:          $SSSONECTOR_MODE"
    echo "  Instance:      $instance_name"
    echo "  TUN Interface: $tun_interface"
    echo "  TUN Address:   $tun_address"
    echo "  Port:          $port"
    echo "  Prometheus:    $prom_port"
    if [ "$SSSONECTOR_MODE" = "client" ]; then
        echo "  Server:        $server_addr:$port"
    fi
    echo
    
    read -p "Proceed with installation? [Y/n]: " confirm
    if [ "$confirm" = "n" ] || [ "$confirm" = "N" ]; then
        log_info "Installation cancelled"
        exit 0
    fi
    
    if [ "$SSSONECTOR_MODE" = "server" ]; then
        create_server_config "$instance_name" "$tun_interface" "$tun_address" "$port" "$prom_port" 1048576 2097152 "$template_dir"
    else
        create_client_config "$instance_name" "$tun_interface" "$tun_address" "$server_addr" "$port" "$prom_port" 1048576 2097152 "$template_dir"
    fi
    
    echo "$instance_name"
}

print_success() {
    local instance_name=$1
    
    echo
    log_info "Installation complete!"
    echo
    echo "Quick start:"
    echo "  Start service:   systemctl start sssonector@${instance_name}"
    echo "  Check status:    systemctl status sssonector@${instance_name}"
    echo "  View logs:       journalctl -u sssonector@${instance_name} -f"
    echo
    echo "Config file:    $INSTANCE_DIR/${instance_name}/config.yaml"
    echo "Certificates:   $INSTANCE_DIR/${instance_name}/certs/"
    echo "Logs:           $LOG_DIR/"
    echo
    
    if [ "$SSSONECTOR_MODE" = "server" ]; then
        echo "Remember to:"
        echo "  1. Copy the CA certificate to clients"
        echo "  2. Open firewall port $(grep -oP 'listen_port:\s*\K\d+' "$INSTANCE_DIR/${instance_name}/config.yaml")"
        echo
    else
        echo "Remember to:"
        echo "  1. Copy the server's CA certificate to this client"
        echo "  2. Ensure the server is reachable at the configured address"
        echo
    fi
}

main() {
    check_root
    
    local os arch version
    
    os=$(detect_os)
    arch=$(detect_arch)
    
    if [ "$os" = "darwin" ]; then
        log_error "This install script is for Linux. For macOS, please use install_macos.sh"
        exit 1
    fi
    
    if [ -n "$SSSONECTOR_VERSION" ]; then
        version="$SSSONECTOR_VERSION"
    else
        version=$(get_latest_version)
    fi
    
    log_step "SSSonector ${version} Installer for Linux"
    echo
    
    create_directories
    
    local tmp_binary
    tmp_binary=$(download_binary "$os" "$arch" "$version")
    install_binary "$tmp_binary"
    
    local template_dir
    template_dir=$(download_templates)
    
    local instance_name
    instance_name=$(interactive_setup "$template_dir")
    
    rm -rf "$template_dir"
    
    if [ -z "$SSSONECTOR_NO_SERVICE" ]; then
        install_systemd_service
        
        read -p "Enable and start the service now? [Y/n]: " start_now
        if [ "$start_now" != "n" ] && [ "$start_now" != "N" ]; then
            systemctl enable "sssonector@${instance_name}"
            systemctl start "sssonector@${instance_name}"
            log_info "Service started"
        fi
    fi
    
    print_success "$instance_name"
}

main "$@"
