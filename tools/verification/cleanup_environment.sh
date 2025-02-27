#!/bin/bash

# cleanup_environment.sh
# Script to clean up the environment before running SSSonector
set -euo pipefail

# Color codes for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

# Logging functions
log_info() {
    echo -e "${GREEN}[INFO]${NC} $1"
}

log_warn() {
    echo -e "${YELLOW}[WARN]${NC} $1"
}

log_error() {
    echo -e "${RED}[ERROR]${NC} $1"
    return 1
}

log_step() {
    echo -e "${BLUE}[STEP]${NC} $1"
}

# QA server details
QA_SERVER="192.168.50.210"
QA_CLIENT="192.168.50.211"
QA_USER="sblanken"
QA_SUDO_PASSWORD="password"

# Test SSH connection to QA servers
log_info "Testing SSH connection to QA servers"
ssh -o ConnectTimeout=5 -o BatchMode=yes -o StrictHostKeyChecking=no ${QA_USER}@${QA_SERVER} echo "SSH connection to server successful" || log_error "Failed to connect to server"
ssh -o ConnectTimeout=5 -o BatchMode=yes -o StrictHostKeyChecking=no ${QA_USER}@${QA_CLIENT} echo "SSH connection to client successful" || log_error "Failed to connect to client"

# Function to stop SSSonector
stop_sssonector() {
    log_step "Stopping SSSonector"
    
    # Stop SSSonector on server
    log_info "Stopping SSSonector on server"
    ssh ${QA_USER}@${QA_SERVER} "sudo systemctl stop sssonector || true"
    ssh ${QA_USER}@${QA_SERVER} "sudo killall -9 sssonector || true"
    
    # Stop SSSonector on client
    log_info "Stopping SSSonector on client"
    ssh ${QA_USER}@${QA_CLIENT} "sudo systemctl stop sssonector || true"
    ssh ${QA_USER}@${QA_CLIENT} "sudo killall -9 sssonector || true"
    
    # Wait for processes to stop
    sleep 2
}

# Function to remove tunnel interfaces
remove_tunnel_interfaces() {
    log_step "Removing tunnel interfaces"
    
    # Remove tunnel interface on server
    log_info "Removing tunnel interface on server"
    ssh ${QA_USER}@${QA_SERVER} "sudo ip link delete tun0 || true"
    
    # Remove tunnel interface on client
    log_info "Removing tunnel interface on client"
    ssh ${QA_USER}@${QA_CLIENT} "sudo ip link delete tun0 || true"
}

# Function to clean up firewall rules
cleanup_firewall_rules() {
    log_step "Cleaning up firewall rules"
    
    # Clean up firewall rules on server
    log_info "Cleaning up firewall rules on server"
    ssh ${QA_USER}@${QA_SERVER} "sudo iptables -F INPUT || true"
    ssh ${QA_USER}@${QA_SERVER} "sudo iptables -F FORWARD || true"
    ssh ${QA_USER}@${QA_SERVER} "sudo iptables -t nat -F POSTROUTING || true"
    
    # Clean up firewall rules on client
    log_info "Cleaning up firewall rules on client"
    ssh ${QA_USER}@${QA_CLIENT} "sudo iptables -F INPUT || true"
    ssh ${QA_USER}@${QA_CLIENT} "sudo iptables -F FORWARD || true"
    ssh ${QA_USER}@${QA_CLIENT} "sudo iptables -t nat -F POSTROUTING || true"
}

# Function to clean up routes
cleanup_routes() {
    log_step "Cleaning up routes"
    
    # Clean up routes on server
    log_info "Cleaning up routes on server"
    ssh ${QA_USER}@${QA_SERVER} "sudo ip route del 10.0.0.2/32 || true"
    
    # Clean up routes on client
    log_info "Cleaning up routes on client"
    ssh ${QA_USER}@${QA_CLIENT} "sudo ip route del 10.0.0.1/32 || true"
}

# Function to reset kernel parameters
reset_kernel_parameters() {
    log_step "Resetting kernel parameters"
    
    # Reset kernel parameters on server
    log_info "Resetting kernel parameters on server"
    ssh ${QA_USER}@${QA_SERVER} "sudo sysctl -w net.ipv4.conf.all.rp_filter=1 || true"
    ssh ${QA_USER}@${QA_SERVER} "sudo sysctl -w net.ipv4.conf.default.rp_filter=1 || true"
    ssh ${QA_USER}@${QA_SERVER} "sudo sysctl -w net.ipv4.icmp_echo_ignore_broadcasts=1 || true"
    
    # Reset kernel parameters on client
    log_info "Resetting kernel parameters on client"
    ssh ${QA_USER}@${QA_CLIENT} "sudo sysctl -w net.ipv4.conf.all.rp_filter=1 || true"
    ssh ${QA_USER}@${QA_CLIENT} "sudo sysctl -w net.ipv4.conf.default.rp_filter=1 || true"
    ssh ${QA_USER}@${QA_CLIENT} "sudo sysctl -w net.ipv4.icmp_echo_ignore_broadcasts=1 || true"
}

# Function to clean up temporary files
cleanup_temp_files() {
    log_step "Cleaning up temporary files"
    
    # Clean up temporary files on server
    log_info "Cleaning up temporary files on server"
    ssh ${QA_USER}@${QA_SERVER} "sudo rm -f /tmp/sssonector* || true"
    
    # Clean up temporary files on client
    log_info "Cleaning up temporary files on client"
    ssh ${QA_USER}@${QA_CLIENT} "sudo rm -f /tmp/sssonector* || true"
}

# Function to restart network services
restart_network_services() {
    log_step "Restarting network services"
    
    # Restart network services on server
    log_info "Restarting network services on server"
    ssh ${QA_USER}@${QA_SERVER} "sudo systemctl restart networking || true"
    
    # Restart network services on client
    log_info "Restarting network services on client"
    ssh ${QA_USER}@${QA_CLIENT} "sudo systemctl restart networking || true"
    
    # Wait for network services to restart
    sleep 5
}

# Function to check system status
check_system_status() {
    log_step "Checking system status"
    
    # Check system status on server
    log_info "Checking system status on server"
    ssh ${QA_USER}@${QA_SERVER} "ip addr show"
    ssh ${QA_USER}@${QA_SERVER} "ip route show"
    ssh ${QA_USER}@${QA_SERVER} "sudo iptables -L -n"
    
    # Check system status on client
    log_info "Checking system status on client"
    ssh ${QA_USER}@${QA_CLIENT} "ip addr show"
    ssh ${QA_USER}@${QA_CLIENT} "ip route show"
    ssh ${QA_USER}@${QA_CLIENT} "sudo iptables -L -n"
}

# Main function
main() {
    log_info "Starting environment cleanup"
    
    # Stop SSSonector
    stop_sssonector
    
    # Remove tunnel interfaces
    remove_tunnel_interfaces
    
    # Clean up firewall rules
    cleanup_firewall_rules
    
    # Clean up routes
    cleanup_routes
    
    # Reset kernel parameters
    reset_kernel_parameters
    
    # Clean up temporary files
    cleanup_temp_files
    
    # Restart network services
    restart_network_services
    
    # Check system status
    check_system_status
    
    log_info "Environment cleanup completed"
}

# Run main function
main "$@"
