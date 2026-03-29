#!/bin/bash
#
# SSSonector Instance Generator
# Creates configuration and certificates for a new tunnel instance
#

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
TEMPLATE_DIR="${SCRIPT_DIR}/templates"
INSTANCE_DIR="/etc/sssonector/instances"

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m'

usage() {
    cat <<EOF
Usage: $(basename "$0") <command> [options]

Commands:
  create-server   Create a server instance
  create-client   Create a client instance
  list            List all instances
  remove          Remove an instance
  generate-certs  Generate certificates for an instance

Options:
  -n, --name          Instance name (required)
  -t, --tun           TUN interface name (default: tunN based on instance number)
  -a, --address       TUN interface address (required)
  -p, --port          Listen/Server port (default: 8443)
  -s, --server        Server address (required for client)
  -r, --rate          Rate limit in bytes/sec (default: 1048576 = 1MB/s)
  -b, --burst         Burst limit in bytes/sec (default: 2097152 = 2MB/s)
  --prometheus-port   Prometheus metrics port (default: 9090)

Examples:
  # Create server instance for client-a
  $(basename "$0") create-server -n client-a -a 10.0.1.1/24 -p 8443

  # Create client instance connecting to server
  $(basename "$0") create-client -n client-a -a 10.0.1.2/24 -s 192.168.1.10 -p 8443

  # List all instances
  $(basename "$0") list

  # Remove an instance
  $(basename "$0") remove -n client-a

EOF
    exit 1
}

log_info() {
    echo -e "${GREEN}[INFO]${NC} $1"
}

log_warn() {
    echo -e "${YELLOW}[WARN]${NC} $1"
}

log_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

get_next_instance_number() {
    local max_num=0
    if [ -d "$INSTANCE_DIR" ]; then
        for dir in "$INSTANCE_DIR"/*/; do
            if [ -d "$dir" ]; then
                name=$(basename "$dir")
                if [[ -f "$dir/config.yaml" ]]; then
                    num=$(grep -oP 'tun\d+' "$dir/config.yaml" 2>/dev/null | grep -oP '\d+' | head -1)
                    if [ -n "$num" ] && [ "$num" -gt "$max_num" ]; then
                        max_num=$num
                    fi
                fi
            fi
        done
    fi
    echo $((max_num + 1))
}

get_next_port() {
    local base_port=8443
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

generate_certificates() {
    local instance_name=$1
    local instance_path="$INSTANCE_DIR/$instance_name"
    local cert_dir="$instance_path/certs"
    
    log_info "Generating certificates for instance: $instance_name"
    
    mkdir -p "$cert_dir"
    
    # Generate CA key and certificate
    openssl genrsa -out "$cert_dir/ca.key" 4096 2>/dev/null
    openssl req -new -x509 -days 365 -key "$cert_dir/ca.key" \
        -out "$cert_dir/ca.crt" \
        -subj "/C=US/ST=State/L=City/O=SSSonector/OU=CA/CN=SSSonector-CA" 2>/dev/null
    
    # Generate server key and certificate
    openssl genrsa -out "$cert_dir/server.key" 4096 2>/dev/null
    openssl req -new -key "$cert_dir/server.key" \
        -out "$cert_dir/server.csr" \
        -subj "/C=US/ST=State/L=City/O=SSSonector/OU=Server/CN=server" 2>/dev/null
    openssl x509 -req -days 365 -in "$cert_dir/server.csr" \
        -CA "$cert_dir/ca.crt" -CAkey "$cert_dir/ca.key" \
        -CAcreateserial -out "$cert_dir/server.crt" 2>/dev/null
    
    # Generate client key and certificate
    openssl genrsa -out "$cert_dir/client.key" 4096 2>/dev/null
    openssl req -new -key "$cert_dir/client.key" \
        -out "$cert_dir/client.csr" \
        -subj "/C=US/ST=State/L=City/O=SSSonector/OU=Client/CN=client" 2>/dev/null
    openssl x509 -req -days 365 -in "$cert_dir/client.csr" \
        -CA "$cert_dir/ca.crt" -CAkey "$cert_dir/ca.key" \
        -CAcreateserial -out "$cert_dir/client.crt" 2>/dev/null
    
    # Set permissions
    chmod 600 "$cert_dir"/*.key
    chmod 644 "$cert_dir"/*.crt
    
    # Clean up CSR files
    rm -f "$cert_dir"/*.csr "$cert_dir"/*.srl
    
    log_info "Certificates generated successfully"
}

create_server() {
    local name=""
    local tun=""
    local address=""
    local port=""
    local prom_port=""
    local rate=1048576
    local burst=2097152
    
    while [[ $# -gt 0 ]]; do
        case $1 in
            -n|--name) name="$2"; shift 2 ;;
            -t|--tun) tun="$2"; shift 2 ;;
            -a|--address) address="$2"; shift 2 ;;
            -p|--port) port="$2"; shift 2 ;;
            -r|--rate) rate="$2"; shift 2 ;;
            -b|--burst) burst="$2"; shift 2 ;;
            --prometheus-port) prom_port="$2"; shift 2 ;;
            *) log_error "Unknown option: $1"; usage ;;
        esac
    done
    
    if [ -z "$name" ] || [ -z "$address" ]; then
        log_error "Instance name and address are required"
        usage
    fi
    
    local instance_num
    instance_num=$(get_next_instance_number)
    
    [ -z "$tun" ] && tun="tun$instance_num"
    [ -z "$port" ] && port=$(get_next_port)
    [ -z "$prom_port" ] && prom_port=$(get_next_prometheus_port)
    
    local instance_path="$INSTANCE_DIR/$name"
    
    if [ -d "$instance_path" ]; then
        log_error "Instance '$name' already exists"
        exit 1
    fi
    
    log_info "Creating server instance: $name"
    mkdir -p "$instance_path"
    
    # Generate certificates
    generate_certificates "$name"
    
    # Create config from template
    sed -e "s/{{INSTANCE_NAME}}/$name/g" \
        -e "s/{{TUN_INTERFACE}}/$tun/g" \
        -e "s/{{TUN_ADDRESS}}/$address/g" \
        -e "s/{{LISTEN_PORT}}/$port/g" \
        -e "s/{{PROMETHEUS_PORT}}/$prom_port/g" \
        -e "s/{{RATE_LIMIT}}/$rate/g" \
        -e "s/{{RATE_BURST}}/$burst/g" \
        "$TEMPLATE_DIR/server.yaml.template" > "$instance_path/config.yaml"
    
    log_info "Server instance created: $instance_path"
    log_info "  TUN Interface: $tun"
    log_info "  TUN Address: $address"
    log_info "  Listen Port: $port"
    log_info "  Prometheus Port: $prom_port"
    log_info ""
    log_info "Start with: systemctl start sssonector@$name"
}

create_client() {
    local name=""
    local tun=""
    local address=""
    local server=""
    local server_port=""
    local prom_port=""
    local rate=1048576
    local burst=2097152
    
    while [[ $# -gt 0 ]]; do
        case $1 in
            -n|--name) name="$2"; shift 2 ;;
            -t|--tun) tun="$2"; shift 2 ;;
            -a|--address) address="$2"; shift 2 ;;
            -s|--server) server="$2"; shift 2 ;;
            -p|--port) server_port="$2"; shift 2 ;;
            -r|--rate) rate="$2"; shift 2 ;;
            -b|--burst) burst="$2"; shift 2 ;;
            --prometheus-port) prom_port="$2"; shift 2 ;;
            *) log_error "Unknown option: $1"; usage ;;
        esac
    done
    
    if [ -z "$name" ] || [ -z "$address" ] || [ -z "$server" ]; then
        log_error "Instance name, address, and server address are required"
        usage
    fi
    
    [ -z "$server_port" ] && server_port=8443
    [ -z "$prom_port" ] && prom_port=9090
    
    local instance_num
    instance_num=$(get_next_instance_number)
    [ -z "$tun" ] && tun="tun$instance_num"
    
    local instance_path="$INSTANCE_DIR/$name"
    
    if [ -d "$instance_path" ]; then
        log_error "Instance '$name' already exists"
        exit 1
    fi
    
    log_info "Creating client instance: $name"
    mkdir -p "$instance_path"
    
    # Generate certificates
    generate_certificates "$name"
    
    # Create config from template
    sed -e "s/{{INSTANCE_NAME}}/$name/g" \
        -e "s/{{TUN_INTERFACE}}/$tun/g" \
        -e "s/{{TUN_ADDRESS}}/$address/g" \
        -e "s/{{SERVER_ADDRESS}}/$server/g" \
        -e "s/{{SERVER_PORT}}/$server_port/g" \
        -e "s/{{PROMETHEUS_PORT}}/$prom_port/g" \
        -e "s/{{RATE_LIMIT}}/$rate/g" \
        -e "s/{{RATE_BURST}}/$burst/g" \
        "$TEMPLATE_DIR/client.yaml.template" > "$instance_path/config.yaml"
    
    log_info "Client instance created: $instance_path"
    log_info "  TUN Interface: $tun"
    log_info "  TUN Address: $address"
    log_info "  Server: $server:$server_port"
    log_info "  Prometheus Port: $prom_port"
    log_info ""
    log_info "Start with: systemctl start sssonector@$name"
}

list_instances() {
    if [ ! -d "$INSTANCE_DIR" ]; then
        log_info "No instances found"
        return
    fi
    
    echo "SSSonector Instances:"
    echo "====================="
    
    for dir in "$INSTANCE_DIR"/*/; do
        if [ -d "$dir" ] && [ -f "$dir/config.yaml" ]; then
            local name=$(basename "$dir")
            local mode=$(grep -oP 'mode:\s*\K\w+' "$dir/config.yaml" 2>/dev/null || echo "unknown")
            local tun=$(grep -oP 'name:\s*tun\d+' "$dir/config.yaml" 2>/dev/null | grep -oP 'tun\d+' || echo "unknown")
            local address=$(grep -oP 'address:\s*\K[0-9./]+' "$dir/config.yaml" 2>/dev/null | head -1 || echo "unknown")
            
            echo ""
            echo "  [$mode] $name"
            echo "    Interface: $tun"
            echo "    Address: $address"
            echo "    Config: $dir/config.yaml"
        fi
    done
}

remove_instance() {
    local name=""
    
    while [[ $# -gt 0 ]]; do
        case $1 in
            -n|--name) name="$2"; shift 2 ;;
            *) log_error "Unknown option: $1"; usage ;;
        esac
    done
    
    if [ -z "$name" ]; then
        log_error "Instance name is required"
        usage
    fi
    
    local instance_path="$INSTANCE_DIR/$name"
    
    if [ ! -d "$instance_path" ]; then
        log_error "Instance '$name' not found"
        exit 1
    fi
    
    log_warn "This will remove instance '$name' and all its data"
    read -p "Are you sure? (y/N): " confirm
    
    if [ "$confirm" = "y" ] || [ "$confirm" = "Y" ]; then
        rm -rf "$instance_path"
        log_info "Instance '$name' removed"
    else
        log_info "Cancelled"
    fi
}

# Main
if [ $# -lt 1 ]; then
    usage
fi

command=$1
shift

case $command in
    create-server) create_server "$@" ;;
    create-client) create_client "$@" ;;
    list) list_instances ;;
    remove) remove_instance "$@" ;;
    generate-certs)
        if [ -z "$1" ]; then
            log_error "Instance name required"
            usage
        fi
        generate_certificates "$1"
        ;;
    *) log_error "Unknown command: $command"; usage ;;
esac
