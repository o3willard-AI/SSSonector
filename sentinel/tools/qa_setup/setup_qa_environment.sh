#!/bin/bash

# setup_qa_environment.sh
# Part of Project SENTINEL - QA Environment Setup Tool
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

# Default paths
DEFAULT_BASE_DIR="/opt/sssonector"
DEFAULT_SERVER_IP="192.168.50.210"
DEFAULT_CLIENT_IP="192.168.50.211"

# Setup directory structure
setup_directories() {
    local base_dir=$1
    log_info "Setting up directory structure in ${base_dir}"

    # Create required directories
    sudo mkdir -p "${base_dir}"/{bin,config,certs,log,state,tools}

    # Set directory permissions
    sudo chmod 755 "${base_dir}"
    sudo chmod 755 "${base_dir}"/{bin,config,certs,log,state,tools}
    sudo chown -R root:root "${base_dir}"

    log_info "Directory structure created successfully"
}

# Configure system requirements
configure_system() {
    log_info "Configuring system requirements"

    # Enable IP forwarding
    echo 1 | sudo tee /proc/sys/net/ipv4/ip_forward
    sudo sysctl -w net.ipv4.ip_forward=1
    
    # Make IP forwarding persistent
    echo "net.ipv4.ip_forward=1" | sudo tee /etc/sysctl.d/99-sssonector.conf

    # Load TUN module
    sudo modprobe tun
    echo "tun" | sudo tee /etc/modules-load.d/sssonector.conf

    log_info "System configuration completed"
}

# Setup monitoring service
setup_monitoring() {
    local base_dir=$1
    log_info "Setting up monitoring service"

    # Create monitoring service file
    cat << EOF | sudo tee /etc/systemd/system/sssonector-monitor.service
[Unit]
Description=SSSonector System Monitoring Service
After=network.target

[Service]
Type=simple
User=sblanken
ExecStart=${base_dir}/tools/monitor.sh -c ${base_dir}/config/monitor.ini -d
Restart=always
RestartSec=10

[Install]
WantedBy=multi-user.target
EOF

    # Create monitoring script
    cat << 'EOF' | sudo tee "${base_dir}/tools/monitor.sh"
#!/bin/bash
set -euo pipefail

METRICS_DIR="${BASE_DIR}/tools/metrics"
mkdir -p "${METRICS_DIR}"

while true; do
    # Collect metrics
    CPU_USER=$(top -bn1 | grep "Cpu(s)" | awk '{print $2}')
    CPU_SYSTEM=$(top -bn1 | grep "Cpu(s)" | awk '{print $4}')
    CPU_IDLE=$(top -bn1 | grep "Cpu(s)" | awk '{print $8}')
    
    MEM_TOTAL=$(free -m | awk '/^Mem:/{print $2}')
    MEM_USED=$(free -m | awk '/^Mem:/{print $3}')
    MEM_FREE=$(free -m | awk '/^Mem:/{print $4}')
    
    DISK_TOTAL=$(df -m / | awk 'NR==2 {print $2}')
    DISK_USED=$(df -m / | awk 'NR==2 {print $3}')
    DISK_FREE=$(df -m / | awk 'NR==2 {print $4}')
    
    RX_BYTES=$(cat /sys/class/net/$(ip route | awk '/default/ {print $5}')/statistics/rx_bytes)
    TX_BYTES=$(cat /sys/class/net/$(ip route | awk '/default/ {print $5}')/statistics/tx_bytes)

    # Write metrics
    cat << METRICS > "${METRICS_DIR}/$(date +%s).metrics"
cpu_user=${CPU_USER}
cpu_system=${CPU_SYSTEM}
cpu_idle=${CPU_IDLE}
memory_total=${MEM_TOTAL}
memory_used=${MEM_USED}
memory_free=${MEM_FREE}
disk_total=${DISK_TOTAL}
disk_used=${DISK_USED}
disk_free=${DISK_FREE}
network_rx_bytes=${RX_BYTES}
network_tx_bytes=${TX_BYTES}
METRICS

    # Cleanup old metrics (keep last 24 hours)
    find "${METRICS_DIR}" -type f -mtime +1 -delete

    sleep 60
done
EOF

    # Set permissions
    sudo chmod 755 "${base_dir}/tools/monitor.sh"
    sudo systemctl daemon-reload
    sudo systemctl enable sssonector-monitor
    sudo systemctl start sssonector-monitor

    log_info "Monitoring service setup completed"
}

# Setup validation scripts
setup_validation() {
    local base_dir=$1
    log_info "Setting up validation scripts"

    # Create validation directory
    sudo mkdir -p "${base_dir}/tools/validation"

    # Create main validation script
    cat << 'EOF' | sudo tee "${base_dir}/tools/validation/validate_environment.sh"
#!/bin/bash
set -euo pipefail

# Source validation modules
source "${BASE_DIR}/tools/qa_validator/lib/process_monitor.sh"
source "${BASE_DIR}/tools/qa_validator/lib/resource_validator.sh"

# Run validations
validate_processes "${BASE_DIR}" || exit 1
validate_resources "${BASE_DIR}" || exit 1

# Verify monitoring service
systemctl is-active sssonector-monitor || exit 1

# Check metrics collection
test -n "$(ls -A ${BASE_DIR}/tools/metrics/)" || exit 1

echo "Environment validation completed successfully"
EOF

    # Set permissions
    sudo chmod 755 "${base_dir}/tools/validation/validate_environment.sh"

    log_info "Validation scripts setup completed"
}

# Main setup function
main() {
    local base_dir="${DEFAULT_BASE_DIR}"
    local server_ip="${DEFAULT_SERVER_IP}"
    local client_ip="${DEFAULT_CLIENT_IP}"

    # Parse command line arguments
    while getopts "d:s:c:h" opt; do
        case ${opt} in
            d)
                base_dir="${OPTARG}"
                ;;
            s)
                server_ip="${OPTARG}"
                ;;
            c)
                client_ip="${OPTARG}"
                ;;
            h)
                echo "Usage: $0 [-d base_directory] [-s server_ip] [-c client_ip]"
                exit 0
                ;;
            \?)
                echo "Invalid option: -${OPTARG}"
                exit 1
                ;;
        esac
    done

    log_info "Starting QA environment setup..."
    log_info "Base directory: ${base_dir}"
    log_info "Server IP: ${server_ip}"
    log_info "Client IP: ${client_ip}"

    # Run setup steps
    setup_directories "${base_dir}"
    configure_system
    setup_monitoring "${base_dir}"
    setup_validation "${base_dir}"

    log_info "QA environment setup completed successfully"
    log_info "Run ${base_dir}/tools/validation/validate_environment.sh to verify the setup"
}

# Execute main function
main "$@"
