#!/bin/bash

# Core functionality test script for SSSonector
# This script runs a series of tests to verify basic tunnel functionality

set -euo pipefail

SERVER_HOST="sblanken@192.168.50.210"
CLIENT_HOST="sblanken@192.168.50.211"

# Color codes for output
GREEN='\033[0;32m'
RED='\033[0;31m'
YELLOW='\033[1;33m'
NC='\033[0m'

# Test counters
TESTS_RUN=0
TESTS_PASSED=0
TESTS_FAILED=0

log_info() {
    echo -e "${GREEN}[INFO]${NC} $1"
}

log_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

log_warn() {
    echo -e "${YELLOW}[WARN]${NC} $1"
}

run_test() {
    local test_name=$1
    local test_cmd=$2
    local expected_exit_code=${3:-0}
    
    echo -e "\n${YELLOW}Running test:${NC} $test_name"
    TESTS_RUN=$((TESTS_RUN + 1))
    
    if eval "$test_cmd"; then
        if [ $? -eq $expected_exit_code ]; then
            echo -e "${GREEN}✓ Test passed:${NC} $test_name"
            TESTS_PASSED=$((TESTS_PASSED + 1))
            return 0
        fi
    fi
    
    echo -e "${RED}✗ Test failed:${NC} $test_name"
    TESTS_FAILED=$((TESTS_FAILED + 1))
    return 1
}

# Test: Server process is running
test_server_process() {
    run_test "Server Process Check" "ssh $SERVER_HOST 'pgrep -f sssonector'"
}

# Test: Client process is running
test_client_process() {
    run_test "Client Process Check" "ssh $CLIENT_HOST 'pgrep -f sssonector'"
}

# Test: Server TUN interface exists and is up
test_server_interface() {
    run_test "Server Interface Check" "ssh $SERVER_HOST 'ip link show tun0 | grep -q UP'"
}

# Test: Client TUN interface exists and is up
test_client_interface() {
    run_test "Client Interface Check" "ssh $CLIENT_HOST 'ip link show tun0 | grep -q UP'"
}

# Test: Server TUN interface is properly configured
test_server_listening() {
    # The server uses TUN interface for tunnel traffic rather than TCP port
    run_test "Server TUN Interface Check" "ssh $SERVER_HOST 'ip link show tun0 | grep -q UP && ip addr show tun0 | grep -q \"inet 10.0.0.1\"'"
}

# Test: Client TUN interface is properly configured
test_client_connection() {
    # The client uses TUN interface for tunnel traffic
    run_test "Client TUN Interface Check" "ssh $CLIENT_HOST 'ip link show tun0 | grep -q UP && ip addr show tun0 | grep -q \"inet 10.0.0.2\"'"
}

# Test: Basic ping connectivity
test_ping_connectivity() {
    run_test "Ping Test (Client → Server)" "ssh $CLIENT_HOST 'ping -c 3 -W 2 10.0.0.1'"
}

# Test: Reverse ping connectivity
test_reverse_ping() {
    run_test "Ping Test (Server → Client)" "ssh $SERVER_HOST 'ping -c 3 -W 2 10.0.0.2'"
}

# Test: MTU configuration
test_mtu_config() {
    run_test "MTU Configuration Check" "ssh $SERVER_HOST 'ip link show tun0 | grep -q \"mtu 1500\"'"
}

# Test: IP forwarding is enabled
test_ip_forwarding() {
    run_test "IP Forwarding Check" "ssh $SERVER_HOST 'sysctl net.ipv4.ip_forward | grep -q \"= 1\"'"
}

# Test: Server certificate exists and is readable
test_server_cert() {
    run_test "Server Certificate Check" "ssh $SERVER_HOST 'test -r ~/sssonector/certs/server.crt'"
}

# Test: Client certificate exists and is readable
test_client_cert() {
    run_test "Client Certificate Check" "ssh $CLIENT_HOST 'test -r ~/sssonector/certs/client.crt'"
}

# Test: Server log file exists and is writable
test_server_log() {
    run_test "Server Log Check" "ssh $SERVER_HOST 'test -w ~/sssonector/log/sssonector.log'"
}

# Test: Client log file exists and is writable
test_client_log() {
    run_test "Client Log Check" "ssh $CLIENT_HOST 'test -w ~/sssonector/log/sssonector.log'"
}

# Test: Server state directory exists and is writable
test_server_state() {
    run_test "Server State Directory Check" "ssh $SERVER_HOST 'test -w ~/sssonector/state'"
}

# Test: Client state directory exists and is writable
test_client_state() {
    run_test "Client State Directory Check" "ssh $CLIENT_HOST 'test -w ~/sssonector/state'"
}

# Run all tests
run_all_tests() {
    log_info "Starting core functionality tests..."
    
    # Process tests
    test_server_process
    test_client_process
    
    # Interface tests
    test_server_interface
    test_client_interface
    test_mtu_config
    
    # Network tests
    test_server_listening
    test_client_connection
    test_ping_connectivity
    test_reverse_ping
    test_ip_forwarding
    
    # File system tests
    test_server_cert
    test_client_cert
    test_server_log
    test_client_log
    test_server_state
    test_client_state
    
    # Print summary
    echo -e "\n${YELLOW}Test Summary:${NC}"
    echo "Tests run:    $TESTS_RUN"
    echo -e "${GREEN}Tests passed: $TESTS_PASSED${NC}"
    echo -e "${RED}Tests failed: $TESTS_FAILED${NC}"
    
    # Return overall success/failure
    [ $TESTS_FAILED -eq 0 ]
}

# Main execution
main() {
    # Ensure tunnel is running
    if ! ./tunnel_control.sh status &>/dev/null; then
        log_error "Tunnel is not running. Please start it first with: ./tunnel_control.sh start"
        exit 1
    fi
    
    # Run tests
    if run_all_tests; then
        log_info "All tests passed successfully"
        exit 0
    else
        log_error "Some tests failed"
        exit 1
    fi
}

# Execute main function
main
