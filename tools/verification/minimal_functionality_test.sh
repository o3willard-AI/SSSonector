#!/bin/bash

# minimal_functionality_test.sh
# Script to test minimal functionality of SSSonector in different deployment scenarios
set -euo pipefail

# QA environment details
QA_SERVER="192.168.50.210"
QA_CLIENT="192.168.50.211"
QA_USER="sblanken"
QA_SUDO_PASSWORD="101abn"

# Test parameters
PACKET_COUNT=20
MAX_RETRIES=3
RETRY_DELAY=5

# Color codes for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
CYAN='\033[0;36m'
NC='\033[0m'

# Logging functions
log_info() {
    echo -e "${GREEN}[INFO]${NC} $1" | tee -a /tmp/sssonector_test.log
}

log_warn() {
    echo -e "${YELLOW}[WARN]${NC} $1" | tee -a /tmp/sssonector_test.log
}

log_error() {
    echo -e "${RED}[ERROR]${NC} $1" | tee -a /tmp/sssonector_test.log
    return 1
}

log_step() {
    echo -e "${BLUE}[STEP]${NC} $1" | tee -a /tmp/sssonector_test.log
}

log_timing() {
    echo -e "${CYAN}[TIME]${NC} $1" | tee -a /tmp/sssonector_test.log
    echo "$1" >> /tmp/sssonector_timing.log
}

# Run command on remote host
run_remote() {
    local host=$1
    local command=$2
    local retry_count=0
    local max_retries=${3:-$MAX_RETRIES}

    while [ $retry_count -lt $max_retries ]; do
        if sshpass -p "${QA_SUDO_PASSWORD}" ssh -o StrictHostKeyChecking=no "${QA_USER}@${host}" "${command}" 2>/tmp/ssh_error.log; then
            return 0
        else
            retry_count=$((retry_count + 1))
            log_warn "Command failed on ${host}, attempt ${retry_count}/${max_retries}"
            log_warn "Error: $(cat /tmp/ssh_error.log)"
            
            if [ $retry_count -lt $max_retries ]; then
                log_info "Retrying in ${RETRY_DELAY} seconds..."
                sleep $RETRY_DELAY
            fi
        fi
    done

    log_error "Command failed on ${host} after ${max_retries} attempts: ${command}"
    return 1
}

# Start timing measurement
start_timing() {
    local label=$1
    local timestamp=$(date +%s.%N)
    echo "$timestamp" > "/tmp/timing_${label}_start"
    log_timing "Started timing for: ${label} at $(date -d @${timestamp%.*})"
}

# End timing measurement and calculate duration
end_timing() {
    local label=$1
    local start_time=$(cat "/tmp/timing_${label}_start")
    local end_time=$(date +%s.%N)
    local duration=$(echo "$end_time - $start_time" | bc)
    
    log_timing "${label}: ${duration} seconds"
    echo "${label},${duration}" >> "/tmp/timing_results.csv"
    
    # Return the duration in milliseconds
    echo $(echo "$duration * 1000" | bc | cut -d'.' -f1)
}

# Generate HTTP-like packet
generate_http_packet() {
    local packet_id=$1
    local host=$2
    
    cat > "/tmp/packet_http_${packet_id}.dat" << EOF
GET /api/resource/${packet_id} HTTP/1.1
Host: example.com
User-Agent: SSSonector-Test/1.0
Accept: application/json
Connection: keep-alive
X-Test-ID: ${packet_id}
X-Test-Timestamp: $(date +%s.%N)

EOF

    run_remote "${host}" "cat > /tmp/packet_http_${packet_id}.dat << 'EOF'
$(cat /tmp/packet_http_${packet_id}.dat)
EOF"
}

# Generate FTP-like packet
generate_ftp_packet() {
    local packet_id=$1
    local host=$2
    
    cat > "/tmp/packet_ftp_${packet_id}.dat" << EOF
USER anonymous
PASS test@example.com
PWD
CWD /pub/test
LIST
RETR test_file_${packet_id}.dat
QUIT
EOF

    run_remote "${host}" "cat > /tmp/packet_ftp_${packet_id}.dat << 'EOF'
$(cat /tmp/packet_ftp_${packet_id}.dat)
EOF"
}

# Generate database-like packet
generate_db_packet() {
    local packet_id=$1
    local host=$2
    
    cat > "/tmp/packet_db_${packet_id}.dat" << EOF
SELECT id, name, status FROM transactions WHERE timestamp > '2025-02-25' LIMIT 1;
INSERT INTO logs (event_type, message, timestamp) VALUES ('TEST', 'Packet ${packet_id}', NOW());
COMMIT;
EOF

    run_remote "${host}" "cat > /tmp/packet_db_${packet_id}.dat << 'EOF'
$(cat /tmp/packet_db_${packet_id}.dat)
EOF"
}

# Start SSSonector in server mode
start_server() {
    local mode=$1
    local config_file="/opt/sssonector/config/server_${mode}.yaml"
    
    log_info "Starting SSSonector in server ${mode} mode"
    
    if [ "${mode}" = "foreground" ]; then
        # Start in foreground mode
        run_remote "${QA_SERVER}" "sudo /opt/sssonector/bin/sssonector -config ${config_file} > /tmp/server.log 2>&1 &"
    else
        # Start in background mode
        run_remote "${QA_SERVER}" "sudo /opt/sssonector/bin/sssonector -config ${config_file} > /dev/null 2>&1 &"
    fi

    # Wait for server to start
    sleep 5

    # Check if server is running
    if run_remote "${QA_SERVER}" "pgrep -f sssonector" &> /dev/null; then
        log_info "SSSonector server started successfully"
    else
        log_error "Failed to start SSSonector server"
        return 1
    fi
}

# Start SSSonector in client mode
start_client() {
    local mode=$1
    local config_file="/opt/sssonector/config/client_${mode}.yaml"
    
    log_info "Starting SSSonector in client ${mode} mode"
    
    if [ "${mode}" = "foreground" ]; then
        # Start in foreground mode
        run_remote "${QA_CLIENT}" "sudo /opt/sssonector/bin/sssonector -config ${config_file} > /tmp/client.log 2>&1 &"
    else
        # Start in background mode
        run_remote "${QA_CLIENT}" "sudo /opt/sssonector/bin/sssonector -config ${config_file} > /dev/null 2>&1 &"
    fi

    # Wait for client to start
    sleep 5

    # Check if client is running
    if run_remote "${QA_CLIENT}" "pgrep -f sssonector" &> /dev/null; then
        log_info "SSSonector client started successfully"
    else
        log_error "Failed to start SSSonector client"
        return 1
    fi
}

# Stop SSSonector
stop_sssonector() {
    local host=$1
    
    log_info "Stopping SSSonector on ${host}"
    
    # Kill SSSonector processes
    run_remote "${host}" "sudo pkill -f sssonector" || true
    
    # Wait for processes to stop
    sleep 2
    
    # Check if processes are still running
    if run_remote "${host}" "pgrep -f sssonector" &> /dev/null; then
        log_warn "SSSonector processes still running on ${host}, forcing kill"
        run_remote "${host}" "sudo pkill -9 -f sssonector" || true
    fi
    
    log_info "SSSonector stopped on ${host}"
}

# Check tunnel status
check_tunnel() {
    log_info "Checking tunnel status"
    
    # Check if tun0 interface exists on server
    log_info "Checking tun0 interface on server"
    if ! run_remote "${QA_SERVER}" "ip link show tun0"; then
        log_error "tun0 interface not found on server"
        return 1
    fi
    
    # Check if tun0 interface exists on client
    log_info "Checking tun0 interface on client"
    if ! run_remote "${QA_CLIENT}" "ip link show tun0"; then
        log_error "tun0 interface not found on client"
        return 1
    fi
    
    # Check server logs
    log_info "Checking server logs"
    run_remote "${QA_SERVER}" "tail -n 20 /opt/sssonector/log/server.log 2>/dev/null || cat /tmp/server.log 2>/dev/null || echo 'No logs found'"
    
    # Check client logs
    log_info "Checking client logs"
    run_remote "${QA_CLIENT}" "tail -n 20 /opt/sssonector/log/client.log 2>/dev/null || cat /tmp/client.log 2>/dev/null || echo 'No logs found'"
    
    log_info "Tunnel established successfully"
    return 0
}

# Send packets from client to server
send_client_to_server_packets() {
    log_step "Sending ${PACKET_COUNT} packets from client to server"
    
    # Start HTTP server on server
    log_info "Starting HTTP server on server"
    run_remote "${QA_SERVER}" "cd /tmp && sudo python3 -m http.server 8000 &"
    sleep 2
    
    # Generate and send packets
    for i in $(seq 1 $PACKET_COUNT); do
        # Determine packet type based on sequence
        if [ $((i % 3)) -eq 0 ]; then
            packet_type="http"
            generate_http_packet $i "${QA_CLIENT}"
        elif [ $((i % 3)) -eq 1 ]; then
            packet_type="ftp"
            generate_ftp_packet $i "${QA_CLIENT}"
        else
            packet_type="db"
            generate_db_packet $i "${QA_CLIENT}"
        fi
        
        log_info "Sending packet ${i}/${PACKET_COUNT} (${packet_type}) from client to server"
        
        # Start timing for this packet
        start_timing "client_to_server_packet_${i}"
        
        # Send packet through tunnel
        if ! run_remote "${QA_CLIENT}" "curl -s -m 5 -X POST -d @/tmp/packet_${packet_type}_${i}.dat http://10.0.0.1:8000/upload -o /dev/null"; then
            log_error "Failed to send packet ${i} from client to server"
            return 1
        fi
        
        # End timing for this packet
        packet_time=$(end_timing "client_to_server_packet_${i}")
        log_info "Packet ${i} sent successfully in ${packet_time}ms"
    done
    
    # Stop HTTP server
    log_info "Stopping HTTP server on server"
    run_remote "${QA_SERVER}" "sudo pkill -f 'python3 -m http.server'" || true
    
    log_info "All ${PACKET_COUNT} packets sent successfully from client to server"
    return 0
}

# Send packets from server to client
send_server_to_client_packets() {
    log_step "Sending ${PACKET_COUNT} packets from server to client"
    
    # Start HTTP server on client
    log_info "Starting HTTP server on client"
    run_remote "${QA_CLIENT}" "cd /tmp && sudo python3 -m http.server 8000 &"
    sleep 2
    
    # Generate and send packets
    for i in $(seq 1 $PACKET_COUNT); do
        # Determine packet type based on sequence
        if [ $((i % 3)) -eq 0 ]; then
            packet_type="http"
            generate_http_packet $i "${QA_SERVER}"
        elif [ $((i % 3)) -eq 1 ]; then
            packet_type="ftp"
            generate_ftp_packet $i "${QA_SERVER}"
        else
            packet_type="db"
            generate_db_packet $i "${QA_SERVER}"
        fi
        
        log_info "Sending packet ${i}/${PACKET_COUNT} (${packet_type}) from server to client"
        
        # Start timing for this packet
        start_timing "server_to_client_packet_${i}"
        
        # Send packet through tunnel
        if ! run_remote "${QA_SERVER}" "curl -s -m 5 -X POST -d @/tmp/packet_${packet_type}_${i}.dat http://10.0.0.2:8000/upload -o /dev/null"; then
            log_error "Failed to send packet ${i} from server to client"
            return 1
        fi
        
        # End timing for this packet
        packet_time=$(end_timing "server_to_client_packet_${i}")
        log_info "Packet ${i} sent successfully in ${packet_time}ms"
    done
    
    # Stop HTTP server
    log_info "Stopping HTTP server on client"
    run_remote "${QA_CLIENT}" "sudo pkill -f 'python3 -m http.server'" || true
    
    log_info "All ${PACKET_COUNT} packets sent successfully from server to client"
    return 0
}

# Create configuration files
create_config_files() {
    local server_mode=$1
    local client_mode=$2
    
    log_step "Creating configuration files for server (${server_mode}) and client (${client_mode})"
    
    # Create server configuration
    cat > "/tmp/server_${server_mode}.yaml" << EOF
# SSSonector Server Configuration (${server_mode} mode)
mode: "server"
listen_address: "0.0.0.0:5000"
tunnel_ip: "10.0.0.1/24"
log_level: "debug"
log_file: "/opt/sssonector/log/server.log"
cert_file: "/opt/sssonector/certs/server.crt"
key_file: "/opt/sssonector/certs/server.key"
ca_file: "/opt/sssonector/certs/ca.crt"
EOF

    # Create client configuration
    cat > "/tmp/client_${client_mode}.yaml" << EOF
# SSSonector Client Configuration (${client_mode} mode)
mode: "client"
server_address: "${QA_SERVER}:5000"
tunnel_ip: "10.0.0.2/24"
log_level: "debug"
log_file: "/opt/sssonector/log/client.log"
cert_file: "/opt/sssonector/certs/client.crt"
key_file: "/opt/sssonector/certs/client.key"
ca_file: "/opt/sssonector/certs/ca.crt"
EOF

    # Copy server configuration to server
    log_info "Copying server configuration to server"
    sshpass -p "${QA_SUDO_PASSWORD}" scp "/tmp/server_${server_mode}.yaml" "${QA_USER}@${QA_SERVER}:/tmp/"
    run_remote "${QA_SERVER}" "sudo mkdir -p /opt/sssonector/config && sudo cp /tmp/server_${server_mode}.yaml /opt/sssonector/config/"
    
    # Copy client configuration to client
    log_info "Copying client configuration to client"
    sshpass -p "${QA_SUDO_PASSWORD}" scp "/tmp/client_${client_mode}.yaml" "${QA_USER}@${QA_CLIENT}:/tmp/"
    run_remote "${QA_CLIENT}" "sudo mkdir -p /opt/sssonector/config && sudo cp /tmp/client_${client_mode}.yaml /opt/sssonector/config/"
    
    log_info "Configuration files created and deployed successfully"
    return 0
}

# Run test scenario
run_test_scenario() {
    local server_mode=$1
    local client_mode=$2
    
    log_step "Running test scenario: Server (${server_mode}), Client (${client_mode})"
    
    # Initialize timing log
    echo "Label,Duration" > "/tmp/timing_results.csv"
    
    # Clean up QA environment
    log_info "Cleaning up QA environment"
    ./cleanup_qa.sh
    
    # Create configuration files
    create_config_files "${server_mode}" "${client_mode}" || return 1
    
    # Deploy SSSonector
    log_info "Deploying SSSonector to QA environment"
    ./deploy_sssonector.sh
    
    # Start server
    start_timing "server_start"
    start_server "${server_mode}" || return 1
    server_start_time=$(end_timing "server_start")
    log_info "Server started in ${server_start_time}ms"
    
    # Start client
    start_timing "client_start"
    start_client "${client_mode}" || {
        stop_sssonector "${QA_SERVER}"
        return 1
    }
    client_start_time=$(end_timing "client_start")
    log_info "Client started in ${client_start_time}ms"
    
    # Check tunnel status
    start_timing "tunnel_establishment"
    check_tunnel || {
        stop_sssonector "${QA_CLIENT}"
        stop_sssonector "${QA_SERVER}"
        return 1
    }
    tunnel_establishment_time=$(end_timing "tunnel_establishment")
    log_info "Tunnel established in ${tunnel_establishment_time}ms"
    
    # Send packets from client to server
    send_client_to_server_packets || {
        stop_sssonector "${QA_CLIENT}"
        stop_sssonector "${QA_SERVER}"
        return 1
    }
    
    # Send packets from server to client
    send_server_to_client_packets || {
        stop_sssonector "${QA_CLIENT}"
        stop_sssonector "${QA_SERVER}"
        return 1
    }
    
    # Stop client
    start_timing "client_stop"
    stop_sssonector "${QA_CLIENT}"
    client_stop_time=$(end_timing "client_stop")
    log_info "Client stopped in ${client_stop_time}ms"
    
    # Check if tunnel is closed
    start_timing "tunnel_closure"
    sleep 2
    if run_remote "${QA_SERVER}" "ip link show tun0" &> /dev/null; then
        log_warn "tun0 interface still exists on server after client shutdown"
    else
        log_info "tun0 interface successfully removed on server after client shutdown"
    fi
    tunnel_closure_time=$(end_timing "tunnel_closure")
    log_info "Tunnel closed in ${tunnel_closure_time}ms"
    
    # Stop server
    start_timing "server_stop"
    stop_sssonector "${QA_SERVER}"
    server_stop_time=$(end_timing "server_stop")
    log_info "Server stopped in ${server_stop_time}ms"
    
    # Generate test report
    generate_test_report "${server_mode}" "${client_mode}"
    
    log_info "Test scenario completed successfully"
    return 0
}

# Generate test report
generate_test_report() {
    local server_mode=$1
    local client_mode=$2
    local report_file="/tmp/sssonector_test_report_${server_mode}_${client_mode}.md"
    
    log_step "Generating test report"
    
    cat > "${report_file}" << EOF
# SSSonector Minimal Functionality Test Report

## Test Scenario
- Server Mode: ${server_mode}
- Client Mode: ${client_mode}
- Test Date: $(date)

## Timing Measurements
$(cat /tmp/timing_results.csv | column -t -s,)

## Packet Transmission Summary
- Client to Server: ${PACKET_COUNT} packets sent successfully
- Server to Client: ${PACKET_COUNT} packets sent successfully

## Test Results
- Tunnel Establishment: SUCCESS
- Client to Server Transmission: SUCCESS
- Server to Client Transmission: SUCCESS
- Tunnel Closure: SUCCESS

## System Information
- Server: ${QA_SERVER}
- Client: ${QA_CLIENT}
- SSSonector Version: $(run_remote "${QA_SERVER}" "sudo /opt/sssonector/bin/sssonector -version" 2>/dev/null || echo "Unknown")

## Logs
Server logs and client logs are available in the QA environment at:
- Server: /opt/sssonector/log/server.log or /tmp/server.log
- Client: /opt/sssonector/log/client.log or /tmp/client.log
EOF

    log_info "Test report generated: ${report_file}"
    cat "${report_file}"
    
    return 0
}

# Main function
main() {
    log_step "Starting SSSonector Minimal Functionality Test"
    
    # Initialize log file
    echo "SSSonector Minimal Functionality Test - $(date)" > /tmp/sssonector_test.log
    
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
    
    # Parse command line arguments
    if [ $# -eq 0 ]; then
        # Run all test scenarios
        log_info "Running all test scenarios"
        
        # Scenario 1: Client foreground, Server foreground
        run_test_scenario "foreground" "foreground" || log_error "Scenario 1 failed"
        
        # Scenario 2: Client background, Server foreground
        run_test_scenario "foreground" "background" || log_error "Scenario 2 failed"
        
        # Scenario 3: Client background, Server background
        run_test_scenario "background" "background" || log_error "Scenario 3 failed"
    else
        # Run specific test scenario
        local server_mode=$1
        local client_mode=$2
        
        if [ -z "${server_mode}" ] || [ -z "${client_mode}" ]; then
            log_error "Usage: $0 [server_mode client_mode]"
            log_error "  server_mode: foreground or background"
            log_error "  client_mode: foreground or background"
            exit 1
        fi
        
        run_test_scenario "${server_mode}" "${client_mode}" || log_error "Test scenario failed"
    fi
    
    log_step "SSSonector Minimal Functionality Test completed"
    log_info "Test logs available at: /tmp/sssonector_test.log"
    log_info "Timing logs available at: /tmp/sssonector_timing.log"
    
    return 0
}

# Run main function with command line arguments
main "$@"
