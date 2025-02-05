#!/bin/bash

# Configuration
LOG_FILE="temp_cert_test.log"
SERVER_SYSTEM="sblanken@192.168.50.210"
CLIENT_SYSTEM="sblanken@192.168.50.211"
TEST_DATA="test_data.txt"
TEST_PORT=8443

# Logging function
log() {
    echo "[$(date '+%Y-%m-%d %H:%M:%S')] $1" | tee -a "$LOG_FILE"
}

# Ensure sssonector binary is installed
ensure_binary() {
    local system=$1
    log "Building and installing sssonector on $system..."
    
    # Clean up any existing repository
    ssh "$system" "rm -rf /tmp/SSSonector"
    
    # Build and copy binary
    if ! ssh "$system" "cd /tmp && git clone https://github.com/o3willard-AI/SSSonector.git && \
                        cd SSSonector && \
                        git checkout main && \
                        git pull && \
                        make clean && make build && \
                        sudo cp bin/sssonector /usr/local/bin/ && \
                        sudo chmod +x /usr/local/bin/sssonector"; then
        log "Failed to build and install sssonector on $system"
        return 1
    fi
    
    return 0
}

# Check if a process is running
is_running() {
    local system=$1
    ssh "$system" "pgrep -f sssonector" > /dev/null
    return $?
}

# Kill any running sssonector processes
cleanup() {
    local system=$1
    log "Cleaning up on $system..."
    ssh "$system" "sudo pkill -f sssonector; sudo rm -f $TEST_DATA"
}

# Create test configuration file
create_config() {
    local system=$1
    local mode=$2
    local config_file="/tmp/config.yaml"
    
    ssh "$system" "cat > $config_file << EOL
mode: $mode
network:
  interface: tun0
  mtu: 1500
tunnel:
  listen_address: 0.0.0.0
  listen_port: $TEST_PORT
  max_clients: 10
logging:
  level: debug
  file: /tmp/sssonector.log
EOL"
    
    echo "$config_file"
}

# Test basic temporary certificate functionality
test_basic_temp_certs() {
    log "Starting basic temporary certificate test..."
    
    # Clean up any existing processes
    cleanup "$SERVER_SYSTEM"
    cleanup "$CLIENT_SYSTEM"
    
    # Create test configuration files
    local server_config=$(create_config "$SERVER_SYSTEM" "server")
    local client_config=$(create_config "$CLIENT_SYSTEM" "client")
    
    # Start server with temporary certificates
    log "Starting server with temporary certificates..."
    ssh "$SERVER_SYSTEM" "sssonector -mode server -test-without-certs -config $server_config" &
    sleep 5
    
    if ! is_running "$SERVER_SYSTEM"; then
        log "Failed to start server"
        return 1
    fi
    
    # Start client with temporary certificates
    log "Starting client with temporary certificates..."
    ssh "$CLIENT_SYSTEM" "sssonector -mode client -test-without-certs -config $client_config" &
    sleep 5
    
    if ! is_running "$CLIENT_SYSTEM"; then
        log "Failed to start client"
        cleanup "$SERVER_SYSTEM"
        return 1
    fi
    
    # Test data transfer
    log "Testing data transfer..."
    ssh "$SERVER_SYSTEM" "echo 'Test data' > $TEST_DATA"
    ssh "$CLIENT_SYSTEM" "nc -w 5 localhost $TEST_PORT < $TEST_DATA"
    
    # Wait for certificate expiration (15 seconds)
    log "Waiting for certificate expiration..."
    sleep 20
    
    # Verify processes have terminated
    if is_running "$SERVER_SYSTEM" || is_running "$CLIENT_SYSTEM"; then
        log "Error: Processes did not terminate after certificate expiration"
        cleanup "$SERVER_SYSTEM"
        cleanup "$CLIENT_SYSTEM"
        return 1
    fi
    
    log "Basic temporary certificate test completed successfully"
    return 0
}

# Test mixed mode (server temp cert, client real cert)
test_mixed_mode() {
    log "Starting mixed mode test..."
    
    # Clean up any existing processes
    cleanup "$SERVER_SYSTEM"
    cleanup "$CLIENT_SYSTEM"
    
    # Create test configuration files
    local server_config=$(create_config "$SERVER_SYSTEM" "server")
    local client_config=$(create_config "$CLIENT_SYSTEM" "client")
    
    # Start server with temporary certificates
    log "Starting server with temporary certificates..."
    ssh "$SERVER_SYSTEM" "sssonector -mode server -test-without-certs -config $server_config" &
    sleep 5
    
    # Start client with real certificates
    log "Starting client with real certificates..."
    ssh "$CLIENT_SYSTEM" "sssonector -mode client -config $client_config" &
    sleep 5
    
    # Check if connection was properly rejected
    if is_running "$CLIENT_SYSTEM"; then
        log "Error: Client should not connect with real certs to temp cert server"
        cleanup "$SERVER_SYSTEM"
        cleanup "$CLIENT_SYSTEM"
        return 1
    fi
    
    log "Mixed mode test completed successfully"
    return 0
}

# Test concurrent connections
test_concurrent() {
    log "Starting concurrent connections test..."
    
    # Clean up any existing processes
    cleanup "$SERVER_SYSTEM"
    cleanup "$CLIENT_SYSTEM"
    
    # Create test configuration files
    local server_config=$(create_config "$SERVER_SYSTEM" "server")
    local client_config=$(create_config "$CLIENT_SYSTEM" "client")
    
    # Start server with temporary certificates
    log "Starting server with temporary certificates..."
    ssh "$SERVER_SYSTEM" "sssonector -mode server -test-without-certs -config $server_config" &
    sleep 5
    
    # Start multiple clients
    for i in {1..3}; do
        log "Starting client $i..."
        ssh "$CLIENT_SYSTEM" "sssonector -mode client -test-without-certs -config $client_config" &
        sleep 2
    done
    
    # Wait for certificate expiration
    log "Waiting for certificate expiration..."
    sleep 20
    
    # Verify all processes have terminated
    if is_running "$SERVER_SYSTEM" || is_running "$CLIENT_SYSTEM"; then
        log "Error: Not all processes terminated after certificate expiration"
        cleanup "$SERVER_SYSTEM"
        cleanup "$CLIENT_SYSTEM"
        return 1
    fi
    
    log "Concurrent connections test completed successfully"
    return 0
}

# Main execution
main() {
    log "Starting temporary certificate tests..."
    
    # Ensure binary is installed on both systems
    if ! ensure_binary "$SERVER_SYSTEM"; then
        log "Failed to install binary on server"
        exit 1
    fi
    
    if ! ensure_binary "$CLIENT_SYSTEM"; then
        log "Failed to install binary on client"
        exit 1
    fi
    
    # Run basic test
    if ! test_basic_temp_certs; then
        log "Basic temporary certificate test failed"
        exit 1
    fi
    
    # Run mixed mode test
    if ! test_mixed_mode; then
        log "Mixed mode test failed"
        exit 1
    fi
    
    # Run concurrent test
    if ! test_concurrent; then
        log "Concurrent connections test failed"
        exit 1
    fi
    
    log "All temporary certificate tests completed successfully"
    exit 0
}

# Start execution
main
