#!/bin/bash

# analyze_packet_capture.sh
# Script to capture and analyze packets
set -euo pipefail

# QA environment details
QA_SERVER="192.168.50.210"
QA_CLIENT="192.168.50.211"
QA_USER="sblanken"
QA_SUDO_PASSWORD="101abn"

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

# Run command on remote host
run_remote() {
    local host=$1
    local command=$2
    
    sshpass -p "${QA_SUDO_PASSWORD}" ssh -o StrictHostKeyChecking=no "${QA_USER}@${host}" "${command}"
}

# Start packet capture
start_packet_capture() {
    local host=$1
    local host_type=$2
    local interface=$3
    local filter=$4
    local output_file=$5
    
    log_info "Starting packet capture on ${host_type} (${host}) interface ${interface}"
    
    # Start tcpdump in background
    run_remote "${host}" "sudo tcpdump -i ${interface} ${filter} -w /tmp/${output_file} -c 100 &"
    sleep 2
}

# Stop packet capture
stop_packet_capture() {
    local host=$1
    local host_type=$2
    
    log_info "Stopping packet capture on ${host_type} (${host})"
    
    # Stop tcpdump
    run_remote "${host}" "sudo pkill -f tcpdump || true"
    sleep 2
}

# Analyze packet capture
analyze_packet_capture() {
    local host=$1
    local host_type=$2
    local output_file=$3
    
    log_info "Analyzing packet capture on ${host_type} (${host})"
    
    # Analyze packet capture
    run_remote "${host}" "sudo tcpdump -r /tmp/${output_file} -n -v"
}

# Start SSSonector
start_sssonector() {
    log_step "Starting SSSonector"
    
    # Start SSSonector on server
    log_info "Starting SSSonector on server"
    run_remote "${QA_SERVER}" "sudo /opt/sssonector/bin/sssonector -config /opt/sssonector/config/server_foreground.yaml > /tmp/server.log 2>&1 &"
    sleep 5
    
    # Start SSSonector on client
    log_info "Starting SSSonector on client"
    run_remote "${QA_CLIENT}" "sudo /opt/sssonector/bin/sssonector -config /opt/sssonector/config/client_foreground.yaml > /tmp/client.log 2>&1 &"
    sleep 5
}

# Stop SSSonector
stop_sssonector() {
    log_info "Stopping SSSonector"
    run_remote "${QA_CLIENT}" "sudo pkill -f sssonector || true"
    run_remote "${QA_SERVER}" "sudo pkill -f sssonector || true"
    sleep 2
}

# Generate test traffic
generate_test_traffic() {
    log_step "Generating test traffic"
    
    # Test ping from client to server
    log_info "Testing ping from client to server"
    run_remote "${QA_CLIENT}" "ping -c 20 10.0.0.1" || log_warn "Ping from client to server failed"
    
    # Test HTTP traffic
    log_info "Testing HTTP traffic"
    run_remote "${QA_SERVER}" "echo 'This is a test file for SSSonector tunnel transfer' | sudo tee /tmp/test_file.txt"
    run_remote "${QA_SERVER}" "cd /tmp && sudo python3 -m http.server 8000 &"
    sleep 2
    run_remote "${QA_CLIENT}" "curl -s http://10.0.0.1:8000/test_file.txt -o /tmp/downloaded_file.txt || echo 'HTTP transfer failed'"
    run_remote "${QA_SERVER}" "sudo pkill -f 'python3 -m http.server' || true"
}

# Main function
main() {
    log_info "Starting packet capture analysis"
    
    # Check if sshpass is installed
    if ! command -v sshpass &> /dev/null; then
        log_info "Installing sshpass..."
        sudo apt-get update && sudo apt-get install -y sshpass
    fi
    
    # Test SSH connection to QA servers
    log_info "Testing SSH connection to QA servers"
    
    if ! sshpass -p "${QA_SUDO_PASSWORD}" ssh -o StrictHostKeyChecking=no "${QA_USER}@${QA_SERVER}" "echo 'SSH connection test successful'" &> /dev/null; then
        log_error "Cannot SSH to server ${QA_SERVER}"
        exit 1
    fi
    
    if ! sshpass -p "${QA_SUDO_PASSWORD}" ssh -o StrictHostKeyChecking=no "${QA_USER}@${QA_CLIENT}" "echo 'SSH connection test successful'" &> /dev/null; then
        log_error "Cannot SSH to client ${QA_CLIENT}"
        exit 1
    fi
    
    # Check if tcpdump is installed
    log_info "Checking if tcpdump is installed"
    run_remote "${QA_SERVER}" "which tcpdump || sudo apt-get update && sudo apt-get install -y tcpdump"
    run_remote "${QA_CLIENT}" "which tcpdump || sudo apt-get update && sudo apt-get install -y tcpdump"
    
    # Start SSSonector
    start_sssonector
    
    # Start packet capture on tun0 interfaces
    start_packet_capture "${QA_SERVER}" "server" "tun0" "icmp or tcp" "server_tun0.pcap"
    start_packet_capture "${QA_CLIENT}" "client" "tun0" "icmp or tcp" "client_tun0.pcap"
    
    # Start packet capture on physical interfaces
    start_packet_capture "${QA_SERVER}" "server" "enp0s3" "host 192.168.50.211" "server_enp0s3.pcap"
    start_packet_capture "${QA_CLIENT}" "client" "enp0s3" "host 192.168.50.210" "client_enp0s3.pcap"
    
    # Generate test traffic
    generate_test_traffic
    
    # Stop packet capture
    stop_packet_capture "${QA_SERVER}" "server"
    stop_packet_capture "${QA_CLIENT}" "client"
    
    # Stop SSSonector
    stop_sssonector
    
    # Analyze packet capture
    log_step "Analyzing packet captures"
    analyze_packet_capture "${QA_SERVER}" "server" "server_tun0.pcap"
    analyze_packet_capture "${QA_CLIENT}" "client" "client_tun0.pcap"
    analyze_packet_capture "${QA_SERVER}" "server" "server_enp0s3.pcap"
    analyze_packet_capture "${QA_CLIENT}" "client" "client_enp0s3.pcap"
    
    log_info "Packet capture analysis completed"
}

# Run main function
main "$@"
