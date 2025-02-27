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

# Test: Server TUN interface exists and is up with correct flags
test_server_interface() {
    run_test "Server Interface Check" "ssh $SERVER_HOST 'ip link show tun0 | grep -q UP && \
        grep -q \"flags: 0x0001\" /sys/class/net/tun0/tun_flags'"
}

# Test: Client TUN interface exists and is up with correct flags
test_client_interface() {
    run_test "Client Interface Check" "ssh $CLIENT_HOST 'ip link show tun0 | grep -q UP && \
        grep -q \"flags: 0x0001\" /sys/class/net/tun0/tun_flags'"
}

# Test: Server TUN interface is properly configured with all required settings
test_server_listening() {
    # The server uses TUN interface for tunnel traffic rather than TCP port
    run_test "Server TUN Interface Check" "ssh $SERVER_HOST 'ip link show tun0 | grep -q UP && \
        ip addr show tun0 | grep -q \"inet 10.0.0.1\" && \
        ip link show tun0 | grep -q \"mtu 1500\" && \
        ip link show tun0 | grep -q \"qlen 500\"'"
}

# Test: Client TUN interface is properly configured with all required settings
test_client_connection() {
    # The client uses TUN interface for tunnel traffic
    run_test "Client TUN Interface Check" "ssh $CLIENT_HOST 'ip link show tun0 | grep -q UP && \
        ip addr show tun0 | grep -q \"inet 10.0.0.2\" && \
        ip link show tun0 | grep -q \"mtu 1500\" && \
        ip link show tun0 | grep -q \"qlen 500\"'"
}

# Test: TUN interface packet handling
test_tun_packet_handling() {
    run_test "TUN Packet Handling" "ssh $CLIENT_HOST 'ping -s 1472 -c 1 10.0.0.1 && \
        ping -s 1473 -c 1 -M do 10.0.0.1 2>&1 | grep -q \"Message too long\"'"
}

# Test: TUN interface performance
test_tun_performance() {
    run_test "TUN Performance Check" "ssh $CLIENT_HOST 'for i in {1..10}; do ping -c 1 -W 1 10.0.0.1 >/dev/null && echo \$i; done | wc -l | grep -q \"10\"'"
}

# Test: TUN interface state transitions
test_tun_state_transitions() {
    run_test "TUN State Transitions" "ssh $SERVER_HOST 'ip link set tun0 down && \
        ip link show tun0 | grep -q DOWN && \
        ip link set tun0 up && \
        ip link show tun0 | grep -q UP'"
}

# Test: TUN interface error handling
test_tun_error_handling() {
    run_test "TUN Error Handling" "ssh $SERVER_HOST 'ip link set tun0 mtu 9000 2>&1 | grep -q \"Invalid argument\"'"
}

# Test: TCP to TUN transition timing
test_tcp_tun_transition() {
    run_test "TCP to TUN Transition" "ssh $CLIENT_HOST '
        jq -r \"select(.startup_log != null) | select(.startup_log.operation == \\\"Starting transfer\\\")\" ~/sssonector/log/sssonector.log > /dev/null
    '"
}

# Test: Connection establishment phases
test_connection_establishment() {
    run_test "Connection Establishment Phases" "ssh $CLIENT_HOST '
        jq -r \"select(.startup_log != null) | .startup_log.phase\" ~/sssonector/log/sssonector.log | sort | uniq | grep -q \"Connection\"
    '"
}

# Test: Resource cleanup after transition
test_resource_cleanup() {
    run_test "Resource Cleanup" "ssh $CLIENT_HOST '
        jq -r \"select(.startup_log != null) | select(.startup_log.operation == \\\"Cleanup adapter\\\")\" ~/sssonector/log/sssonector.log > /dev/null
    '"
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

# Test: Server config integrity
test_server_config_integrity() {
    run_test "Server Config Integrity" "ssh $SERVER_HOST '
        # Check permissions
        test \$(stat -c %a ~/sssonector/config/config.yaml) = \"644\" &&
        # Check ownership
        test \$(stat -c %U:%G ~/sssonector/config/config.yaml) = \"root:root\" &&
        # Check type field
        jq -e \".type == \\\"server\\\"\" ~/sssonector/config/config.yaml > /dev/null
    '"
}

# Test: Client config integrity
test_client_config_integrity() {
    run_test "Client Config Integrity" "ssh $CLIENT_HOST '
        # Check permissions
        test \$(stat -c %a ~/sssonector/config/config.yaml) = \"644\" &&
        # Check ownership
        test \$(stat -c %U:%G ~/sssonector/config/config.yaml) = \"root:root\" &&
        # Check type field
        jq -e \".type == \\\"client\\\"\" ~/sssonector/config/config.yaml > /dev/null
    '"
}

# Test: Server startup logging phases
test_server_startup_logs() {
    run_test "Server Startup Logging Phases" "ssh $SERVER_HOST '
        log_entries=\$(jq -r \"select(.startup_log != null) | .startup_log.phase\" ~/sssonector/log/sssonector.log)
        echo \"\$log_entries\" | grep -q \"PreStartup\" &&
        echo \"\$log_entries\" | grep -q \"Initialization\" &&
        echo \"\$log_entries\" | grep -q \"Connection\" &&
        echo \"\$log_entries\" | grep -q \"Listen\"
    '"
}

# Test: Client startup logging phases
test_client_startup_logs() {
    run_test "Client Startup Logging Phases" "ssh $CLIENT_HOST '
        log_entries=\$(jq -r \"select(.startup_log != null) | .startup_log.phase\" ~/sssonector/log/sssonector.log)
        echo \"\$log_entries\" | grep -q \"PreStartup\" &&
        echo \"\$log_entries\" | grep -q \"Initialization\" &&
        echo \"\$log_entries\" | grep -q \"Connection\"
    '"
}

# Test: Server startup operation timing
test_server_operation_timing() {
    run_test "Server Operation Timing" "ssh $SERVER_HOST '
        jq -e \"select(.startup_log != null) | select(.startup_log.duration != null)\" ~/sssonector/log/sssonector.log > /dev/null
    '"
}

# Test: Client startup operation timing
test_client_operation_timing() {
    run_test "Client Operation Timing" "ssh $CLIENT_HOST '
        jq -e \"select(.startup_log != null) | select(.startup_log.duration != null)\" ~/sssonector/log/sssonector.log > /dev/null
    '"
}

# Test: Server resource state tracking
test_server_resource_state() {
    run_test "Server Resource State Tracking" "ssh $SERVER_HOST '
        jq -e \"select(.startup_log != null) | select(.startup_log.details.state != null)\" ~/sssonector/log/sssonector.log > /dev/null
    '"
}

# Test: Client resource state tracking
test_client_resource_state() {
    run_test "Client Resource State Tracking" "ssh $CLIENT_HOST '
        jq -e \"select(.startup_log != null) | select(.startup_log.details.state != null)\" ~/sssonector/log/sssonector.log > /dev/null
    '"
}

# Test: Server startup log format
test_server_log_format() {
    run_test "Server Log Format" "ssh $SERVER_HOST '
        jq -e \"select(.startup_log != null) | .startup_log | 
            select(.phase != null and .component != null and .operation != null and .timestamp != null)
        \" ~/sssonector/log/sssonector.log > /dev/null
    '"
}

# Test: Client startup log format
test_client_log_format() {
    run_test "Client Log Format" "ssh $CLIENT_HOST '
        jq -e \"select(.startup_log != null) | .startup_log | 
            select(.phase != null and .component != null and .operation != null and .timestamp != null)
        \" ~/sssonector/log/sssonector.log > /dev/null
    '"
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
    test_tun_packet_handling
    test_tun_performance
    test_tun_state_transitions
    test_tun_error_handling
    
    # Connection transition tests
    test_tcp_tun_transition
    test_connection_establishment
    test_resource_cleanup
    
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
    test_server_config_integrity
    test_client_config_integrity
    
    # Startup logging tests
    test_server_startup_logs
    test_client_startup_logs
    test_server_operation_timing
    test_client_operation_timing
    test_server_resource_state
    test_client_resource_state
    test_server_log_format
    test_client_log_format
    
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
