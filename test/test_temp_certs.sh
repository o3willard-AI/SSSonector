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

# Kill any running sssonector processes and clean up temp directories
cleanup() {
    local system=$1
    log "Cleaning up on $system..."
    ssh "$system" "sudo pkill -9 -f sssonector; sudo rm -f $TEST_DATA; sudo rm -rf /tmp/tmp.*"
}

# Check if a process is running
is_running() {
    local system=$1
    ssh "$system" "pgrep -f sssonector" > /dev/null
    return $?
}

# Ensure sssonector binary is installed
ensure_binary() {
    local system=$1
    log "Building and installing sssonector on $system..."
    
    # Build locally
    cd /home/sblanken/Desktop/go/src/github.com/o3willard-AI/SSSonector && make clean && make build
    if [ $? -ne 0 ]; then
        log "Failed to build binary locally"
        return 1
    fi
    
    # Calculate local binary hash
    local_hash=$(cd /home/sblanken/Desktop/go/src/github.com/o3willard-AI/SSSonector && sha256sum build/sssonector | cut -d' ' -f1)
    log "Local binary hash: $local_hash"

    # Copy and install binary
    scp /home/sblanken/Desktop/go/src/github.com/o3willard-AI/SSSonector/build/sssonector "$system:/tmp/sssonector"
    if [ $? -ne 0 ]; then
        log "Failed to copy binary"
        return 1
    fi

    # Verify binary hash
    remote_hash=$(ssh "$system" "sha256sum /tmp/sssonector | cut -d' ' -f1")
    if [ "$local_hash" != "$remote_hash" ]; then
        log "Binary hash mismatch on $system"
        log "Expected: $local_hash"
        log "Got: $remote_hash"
        return 1
    fi
    log "Binary hash verified on $system"
    
    # Install binary and setup permissions
    ssh "$system" "sudo cp /tmp/sssonector /usr/local/bin/ && \
                   sudo chmod +x /usr/local/bin/sssonector && \
                   sudo groupadd -f tun && \
                   sudo usermod -aG tun \$USER && \
                   sudo modprobe tun && \
                   sudo mkdir -p /dev/net && \
                   if [ ! -e /dev/net/tun ]; then sudo mknod /dev/net/tun c 10 200; fi && \
                   sudo chmod 666 /dev/net/tun && \
                   sudo apt-get update && \
                   sudo apt-get install -y iproute2"
    if [ $? -ne 0 ]; then
        log "Failed to install binary and setup permissions"
        return 1
    fi
    
    # Need to reconnect for group changes to take effect
    ssh "$system" "exit"
    
    return 0
}

# Create test configuration file
create_config() {
    local system=$1
    local mode=$2
    local cert_dir=$3
    local config_file="/tmp/config.yaml"
    local address="10.0.0.1/24"
    if [ "$mode" = "client" ]; then
        address="10.0.0.2/24"
    fi
    
    ssh "$system" "cat > $config_file << EOL
mode: \"$mode\"

network:
  interface: \"tun0\"
  address: \"$address\"
  mtu: 1500

tunnel:
  cert_file: \"$cert_dir/$mode.crt\"
  key_file: \"$cert_dir/$mode.key\"
  ca_file: \"$cert_dir/ca.crt\"
  listen_address: \"0.0.0.0\"
  listen_port: $TEST_PORT
  server_address: \"${SERVER_SYSTEM#*@}\"
  server_port: $TEST_PORT
  max_clients: 10
  upload_kbps: 10240
  download_kbps: 10240

monitor:
  log_file: \"/tmp/sssonector.log\"
  snmp_enabled: false
  snmp_port: 161
  snmp_community: \"public\"
EOL"
    
    echo "$config_file"
}

# Verify certificate files match between systems
verify_certs() {
    local server_dir=$1
    local client_dir=$2
    local files=("server.crt" "server.key" "client.crt" "client.key" "ca.crt" "ca.key")
    
    for file in "${files[@]}"; do
        log "Verifying $file..."
        server_hash=$(ssh "$SERVER_SYSTEM" "sha256sum $server_dir/$file | cut -d' ' -f1")
        client_hash=$(ssh "$CLIENT_SYSTEM" "sha256sum $client_dir/$file | cut -d' ' -f1")
        if [ "$server_hash" != "$client_hash" ]; then
            log "Certificate hash mismatch for $file"
            log "Server hash: $server_hash"
            log "Client hash: $client_hash"
            return 1
        fi
    done
    log "All certificate files verified"
    return 0
}

# Test basic temporary certificate functionality
test_basic_temp_certs() {
    log "Starting basic temporary certificate test..."
    
    # Clean up any existing processes
    cleanup "$SERVER_SYSTEM"
    cleanup "$CLIENT_SYSTEM"
    
    # Create temporary directories
    local server_temp_dir=$(ssh "$SERVER_SYSTEM" "mktemp -d")
    local client_temp_dir=$(ssh "$CLIENT_SYSTEM" "mktemp -d")
    
    # Create test configuration files
    local server_config=$(create_config "$SERVER_SYSTEM" "server" "$server_temp_dir")
    local client_config=$(create_config "$CLIENT_SYSTEM" "client" "$client_temp_dir")
    
    # Generate certificates directly in server_temp_dir
    log "Generating temporary certificates..."
    ssh "$SERVER_SYSTEM" "sudo -E sssonector -mode server -test-without-certs -config $server_config -keyfile $server_temp_dir -generate-certs-only > /tmp/gen.log 2>&1 && \
                         sudo chown -R sblanken:sblanken $server_temp_dir && \
                         sudo chmod 700 $server_temp_dir && \
                         sudo find $server_temp_dir -name '*.crt' -exec chmod 644 {} \; && \
                         sudo find $server_temp_dir -name '*.key' -exec chmod 600 {} \; && \
                         sudo sync"
    if [ $? -ne 0 ]; then
        log "Failed to generate temporary certificates"
        log "Generation log:"
        ssh "$SERVER_SYSTEM" "cat /tmp/gen.log"
        return 1
    fi
    
    # Wait a moment for file system to sync and verify files
    sleep 2
    ssh "$SERVER_SYSTEM" "sudo ls -la $server_temp_dir/"
    
    # Check if certificates were generated
    log "Checking certificate files:"
    if ! ssh "$SERVER_SYSTEM" "ls -l $server_temp_dir/{server,client,ca}.{crt,key} 2>/dev/null"; then
        log "No certificate files found"
        log "Generation log:"
        ssh "$SERVER_SYSTEM" "cat /tmp/gen.log"
        log "Directory contents:"
        ssh "$SERVER_SYSTEM" "ls -la $server_temp_dir/"
        log "File permissions:"
        ssh "$SERVER_SYSTEM" "stat $server_temp_dir/"
        return 1
    fi
    
    # Copy certificates to client with proper permissions
    log "Copying certificates to client..."
    for cert in ca.crt ca.key server.crt server.key client.crt client.key; do
        scp "$SERVER_SYSTEM:$server_temp_dir/$cert" "/tmp/$cert"
        scp "/tmp/$cert" "$CLIENT_SYSTEM:$client_temp_dir/$cert"
        if [[ $cert == *.key ]]; then
            ssh "$CLIENT_SYSTEM" "chmod 600 $client_temp_dir/$cert"
        else
            ssh "$CLIENT_SYSTEM" "chmod 644 $client_temp_dir/$cert"
        fi
        rm -f "/tmp/$cert"
    done
    
    # Verify certificate files match
    if ! verify_certs "$server_temp_dir" "$client_temp_dir"; then
        log "Certificate verification failed"
        return 1
    fi
    
    # Start server with temporary certificates
    log "Starting server with temporary certificates..."
    ssh "$SERVER_SYSTEM" "sudo -E env TEMP_DIR=$server_temp_dir sssonector -mode server -test-without-certs -config $server_config -keyfile $server_temp_dir > /tmp/server.log 2>&1" &
    
    # Wait for server to start and verify it's listening
    for i in {1..10}; do
        if ssh "$SERVER_SYSTEM" "sudo netstat -tulpn | grep $TEST_PORT"; then
            log "Server is listening on port $TEST_PORT"
            break
        fi
        if [ $i -eq 10 ]; then
            log "Server failed to start and listen on port $TEST_PORT"
            log "Server logs:"
            ssh "$SERVER_SYSTEM" "cat /tmp/server.log"
            return 1
        fi
        sleep 1
    done
    
    # Check server logs
    log "Server logs:"
    ssh "$SERVER_SYSTEM" "cat /tmp/server.log || echo 'No server log file found'"
    
    # Check server process and environment
    log "Server process info:"
    ssh "$SERVER_SYSTEM" "ps aux | grep sssonector || echo 'No sssonector process found'"
    
    # Check server environment variables
    log "Server environment variables:"
    ssh "$SERVER_SYSTEM" "sudo cat /proc/\$(pgrep sssonector)/environ 2>/dev/null | tr '\0' '\n' | grep TEMP_DIR || echo 'No environment variables found'"
    
    # Check if server is listening
    log "Checking server port..."
    ssh "$SERVER_SYSTEM" "sudo netstat -tulpn | grep $TEST_PORT || echo 'Server not listening on port $TEST_PORT'"
    
    if ! is_running "$SERVER_SYSTEM"; then
        log "Failed to start server"
        return 1
    fi
    
    # Start client with temporary certificates
    log "Starting client with temporary certificates..."
    ssh "$CLIENT_SYSTEM" "sudo -E env TEMP_DIR=$client_temp_dir sssonector -mode client -test-without-certs -config $client_config -keyfile $client_temp_dir > /tmp/client.log 2>&1" &
    sleep 5
    
    if ! is_running "$CLIENT_SYSTEM"; then
        log "Failed to start client"
        log "Client logs:"
        ssh "$CLIENT_SYSTEM" "cat /tmp/client.log"
        cleanup "$SERVER_SYSTEM"
        return 1
    fi
    
    # Test data transfer
    log "Testing data transfer..."
    ssh "$SERVER_SYSTEM" "echo 'Test data' > /tmp/$TEST_DATA"
    scp "$SERVER_SYSTEM:/tmp/$TEST_DATA" "/tmp/$TEST_DATA"
    scp "/tmp/$TEST_DATA" "$CLIENT_SYSTEM:/tmp/$TEST_DATA"
    
    # Wait for TUN interface to be ready and configured
    for i in {1..30}; do
        server_ready=false
        client_ready=false
        
        # Check server TUN interface
        if ssh "$SERVER_SYSTEM" "ip addr show tun0 2>/dev/null | grep -q '10.0.0.1/24' && ip link show tun0 2>/dev/null | grep -q 'UP'"; then
            server_ready=true
        fi
        
        # Check client TUN interface
        if ssh "$CLIENT_SYSTEM" "ip addr show tun0 2>/dev/null | grep -q '10.0.0.2/24' && ip link show tun0 2>/dev/null | grep -q 'UP'"; then
            client_ready=true
        fi
        
        if [ "$server_ready" = true ] && [ "$client_ready" = true ]; then
            log "TUN interfaces configured successfully"
            break
        fi
        
        if [ $i -eq 30 ]; then
            log "TUN interfaces failed to initialize"
            log "Server TUN status:"
            ssh "$SERVER_SYSTEM" "ip addr show tun0 2>/dev/null || echo 'No tun0 interface'"
            log "Client TUN status:"
            ssh "$CLIENT_SYSTEM" "ip addr show tun0 2>/dev/null || echo 'No tun0 interface'"
            log "Server logs:"
            ssh "$SERVER_SYSTEM" "cat /tmp/server.log"
            log "Client logs:"
            ssh "$CLIENT_SYSTEM" "cat /tmp/client.log"
            cleanup "$SERVER_SYSTEM"
            cleanup "$CLIENT_SYSTEM"
            return 1
        fi
        sleep 1
    done
    
    # Test data transfer through server IP
    if ! ssh "$CLIENT_SYSTEM" "nc -w 5 ${SERVER_SYSTEM#*@} $TEST_PORT < /tmp/$TEST_DATA"; then
        log "Data transfer failed"
        log "Server logs:"
        ssh "$SERVER_SYSTEM" "cat /tmp/server.log"
        log "Client logs:"
        ssh "$CLIENT_SYSTEM" "cat /tmp/client.log"
        cleanup "$SERVER_SYSTEM"
        cleanup "$CLIENT_SYSTEM"
        return 1
    fi
    log "Data transfer successful"
    
    # Wait for certificate expiration (15 seconds)
    log "Waiting for certificate expiration..."
    sleep 15
    
    # Force kill processes after expiration
    cleanup "$SERVER_SYSTEM"
    cleanup "$CLIENT_SYSTEM"
    
    # Check final logs for proper shutdown messages
    log "Checking shutdown logs..."
    log "Server shutdown logs:"
    ssh "$SERVER_SYSTEM" "cat /tmp/server.log | grep -i 'shutdown\|expire\|terminate'"
    log "Client shutdown logs:"
    ssh "$CLIENT_SYSTEM" "cat /tmp/client.log | grep -i 'shutdown\|expire\|terminate'"
    
    log "Basic temporary certificate test completed successfully"
    return 0
}

# Test mixed mode (server temp cert, client real cert)
test_mixed_mode() {
    log "Starting mixed mode test..."
    
    # Clean up any existing processes
    cleanup "$SERVER_SYSTEM"
    cleanup "$CLIENT_SYSTEM"
    
    # Create temporary directory for server
    local server_temp_dir=$(ssh "$SERVER_SYSTEM" "mktemp -d")
    
    # Create test configuration files
    local server_config=$(create_config "$SERVER_SYSTEM" "server" "$server_temp_dir")
    local client_config=$(create_config "$CLIENT_SYSTEM" "client" "/etc/sssonector/certs")
    
    # Start server with temporary certificates
    log "Starting server with temporary certificates..."
    ssh "$SERVER_SYSTEM" "sudo -E env TEMP_DIR=$server_temp_dir sssonector -mode server -test-without-certs -config $server_config -keyfile $server_temp_dir > /tmp/server.log 2>&1" &
    
    # Wait for server to start
    for i in {1..10}; do
        if ssh "$SERVER_SYSTEM" "sudo netstat -tulpn | grep $TEST_PORT"; then
            log "Server is listening on port $TEST_PORT"
            break
        fi
        if [ $i -eq 10 ]; then
            log "Server failed to start and listen on port $TEST_PORT"
            log "Server logs:"
            ssh "$SERVER_SYSTEM" "cat /tmp/server.log"
            return 1
        fi
        sleep 1
    done
    
    # Start client with real certificates
    log "Starting client with real certificates..."
    ssh "$CLIENT_SYSTEM" "sssonector -mode client -config $client_config > /tmp/client.log 2>&1" &
    sleep 5
    
    # Check if connection was properly rejected
    if is_running "$CLIENT_SYSTEM"; then
        log "Error: Client should not connect with real certs to temp cert server"
        log "Server logs:"
        ssh "$SERVER_SYSTEM" "cat /tmp/server.log"
        log "Client logs:"
        ssh "$CLIENT_SYSTEM" "cat /tmp/client.log"
        cleanup "$SERVER_SYSTEM"
        cleanup "$CLIENT_SYSTEM"
        return 1
    fi
    
    # Check logs for proper rejection messages
    log "Checking rejection logs..."
    log "Server logs:"
    ssh "$SERVER_SYSTEM" "cat /tmp/server.log | grep -i 'reject\|invalid\|error'"
    log "Client logs:"
    ssh "$CLIENT_SYSTEM" "cat /tmp/client.log | grep -i 'reject\|invalid\|error'"
    
    log "Mixed mode test completed successfully"
    return 0
}

# Test concurrent connections
test_concurrent() {
    log "Starting concurrent connections test..."
    
    # Clean up any existing processes
    cleanup "$SERVER_SYSTEM"
    cleanup "$CLIENT_SYSTEM"
    
    # Create temporary directories
    local server_temp_dir=$(ssh "$SERVER_SYSTEM" "mktemp -d")
    
    # Create test configuration files
    local server_config=$(create_config "$SERVER_SYSTEM" "server" "$server_temp_dir")
    local client_config=$(create_config "$CLIENT_SYSTEM" "client" "$server_temp_dir")
    
    # Start server with temporary certificates
    log "Starting server with temporary certificates..."
    ssh "$SERVER_SYSTEM" "sudo -E env TEMP_DIR=$server_temp_dir sssonector -mode server -test-without-certs -config $server_config -keyfile $server_temp_dir > /tmp/server.log 2>&1" &
    
    # Wait for server to start
    for i in {1..10}; do
        if ssh "$SERVER_SYSTEM" "sudo netstat -tulpn | grep $TEST_PORT"; then
            log "Server is listening on port $TEST_PORT"
            break
        fi
        if [ $i -eq 10 ]; then
            log "Server failed to start and listen on port $TEST_PORT"
            log "Server logs:"
            ssh "$SERVER_SYSTEM" "cat /tmp/server.log"
            return 1
        fi
        sleep 1
    done
    
    # Start multiple clients
    for i in {1..3}; do
        local client_temp_dir=$(ssh "$CLIENT_SYSTEM" "mktemp -d")
        log "Starting client $i..."
        ssh "$CLIENT_SYSTEM" "sudo -E env TEMP_DIR=$client_temp_dir sssonector -mode client -test-without-certs -config $client_config -keyfile $client_temp_dir > /tmp/client_${i}.log 2>&1" &
        sleep 2
        
        # Verify client started
        if ! is_running "$CLIENT_SYSTEM"; then
            log "Failed to start client $i"
            log "Client $i logs:"
            ssh "$CLIENT_SYSTEM" "cat /tmp/client_${i}.log"
            cleanup "$SERVER_SYSTEM"
            cleanup "$CLIENT_SYSTEM"
            return 1
        fi
    done
    
    # Wait for certificate expiration
    log "Waiting for certificate expiration..."
    sleep 15
    
    # Force kill processes after expiration
    cleanup "$SERVER_SYSTEM"
    cleanup "$CLIENT_SYSTEM"
    
    # Check final logs for proper shutdown messages
    log "Checking shutdown logs..."
    log "Server shutdown logs:"
    ssh "$SERVER_SYSTEM" "cat /tmp/server.log | grep -i 'shutdown\|expire\|terminate'"
    for i in {1..3}; do
        log "Client $i shutdown logs:"
        ssh "$CLIENT_SYSTEM" "cat /tmp/client_${i}.log | grep -i 'shutdown\|expire\|terminate'"
    done
    
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
