#!/bin/bash
# Script to run test scenarios for SSSonector

# Configuration
QA_SERVER="192.168.50.210"
QA_CLIENT="192.168.50.211"
LOG_DIR="test_logs"
PING_COUNT=20
SUDO_PASS="101abn"

# Function to check command status
check_status() {
    if [ $? -eq 0 ]; then
        echo "[✓] $1"
    else
        echo "[✗] $1"
        exit 1
    fi
}

# Create log directory
mkdir -p $LOG_DIR

# Function to run server
run_server() {
    local mode=$1  # foreground or background
    echo "Starting server in $mode mode..."
    
    # Create temporary script
    local TEMP_SCRIPT=$(mktemp)
    cat << 'EOF' > $TEMP_SCRIPT
#!/bin/bash
if [ "$1" = "foreground" ]; then
    # Create log directory with proper permissions
    sudo mkdir -p /var/log/sssonector
    sudo chown -R $USER:$USER /var/log/sssonector
    
    # Run server
    sudo sssonector --mode server --foreground > /var/log/sssonector/server.log 2>&1
else
    sudo systemctl start sssonector@server
fi
EOF
    
    # Copy script to remote system
    scp $TEMP_SCRIPT $QA_SERVER:/tmp/run_server.sh
    
    # Make script executable and run with sudo
    if [ "$mode" = "foreground" ]; then
        ssh $QA_SERVER "chmod +x /tmp/run_server.sh && echo $SUDO_PASS | sudo -S /tmp/run_server.sh foreground &"
        SERVER_PID=$!
    else
        ssh $QA_SERVER "chmod +x /tmp/run_server.sh && echo $SUDO_PASS | sudo -S /tmp/run_server.sh background"
    fi
    
    # Clean up
    ssh $QA_SERVER "rm /tmp/run_server.sh"
    rm $TEMP_SCRIPT
    
    # Wait for server to start
    sleep 5
    check_status "Server startup"
}

# Function to run client
run_client() {
    local mode=$1  # foreground or background
    echo "Starting client in $mode mode..."
    
    # Create temporary script
    local TEMP_SCRIPT=$(mktemp)
    cat << 'EOF' > $TEMP_SCRIPT
#!/bin/bash
if [ "$1" = "foreground" ]; then
    # Create log directory with proper permissions
    sudo mkdir -p /var/log/sssonector
    sudo chown -R $USER:$USER /var/log/sssonector
    
    # Run client
    sudo sssonector --mode client --foreground > /var/log/sssonector/client.log 2>&1
else
    sudo systemctl start sssonector@client
fi
EOF
    
    # Copy script to remote system
    scp $TEMP_SCRIPT $QA_CLIENT:/tmp/run_client.sh
    
    # Make script executable and run with sudo
    if [ "$mode" = "foreground" ]; then
        ssh $QA_CLIENT "chmod +x /tmp/run_client.sh && echo $SUDO_PASS | sudo -S /tmp/run_client.sh foreground &"
        CLIENT_PID=$!
    else
        ssh $QA_CLIENT "chmod +x /tmp/run_client.sh && echo $SUDO_PASS | sudo -S /tmp/run_client.sh background"
    fi
    
    # Clean up
    ssh $QA_CLIENT "rm /tmp/run_client.sh"
    rm $TEMP_SCRIPT
    
    # Wait for client to start
    sleep 5
    check_status "Client startup"
}

# Function to test connectivity
test_connectivity() {
    local scenario=$1
    local timestamp=$(date +%Y%m%d_%H%M%S)
    local log_file="$LOG_DIR/${scenario}_${timestamp}.log"
    
    echo "Testing connectivity for scenario: $scenario" | tee -a $log_file
    
    # Create temporary script for ping tests
    local TEMP_SCRIPT=$(mktemp)
    cat << 'EOF' > $TEMP_SCRIPT
#!/bin/bash
sudo ping -c $1 $2
EOF
    
    # Copy script to remote systems
    scp $TEMP_SCRIPT $QA_CLIENT:/tmp/ping_test.sh
    scp $TEMP_SCRIPT $QA_SERVER:/tmp/ping_test.sh
    
    # Test client to server
    echo "Testing client → server connectivity..." | tee -a $log_file
    ssh $QA_CLIENT "chmod +x /tmp/ping_test.sh && echo $SUDO_PASS | sudo -S /tmp/ping_test.sh $PING_COUNT 10.0.0.1" >> $log_file
    check_status "Client to server ping test"
    
    # Test server to client
    echo "Testing server → client connectivity..." | tee -a $log_file
    ssh $QA_SERVER "chmod +x /tmp/ping_test.sh && echo $SUDO_PASS | sudo -S /tmp/ping_test.sh $PING_COUNT 10.0.0.2" >> $log_file
    check_status "Server to client ping test"
    
    # Clean up
    ssh $QA_CLIENT "rm /tmp/ping_test.sh"
    ssh $QA_SERVER "rm /tmp/ping_test.sh"
    rm $TEMP_SCRIPT
    
    # Collect metrics
    echo "Collecting metrics..." | tee -a $log_file
    ssh $QA_SERVER "echo $SUDO_PASS | sudo -S journalctl -u sssonector --no-pager -n 50" >> $log_file
    ssh $QA_CLIENT "echo $SUDO_PASS | sudo -S journalctl -u sssonector --no-pager -n 50" >> $log_file
    
    # Collect logs
    echo "Collecting logs..." | tee -a $log_file
    ssh $QA_SERVER "echo $SUDO_PASS | sudo -S cat /var/log/sssonector/server.log" >> $log_file
    ssh $QA_CLIENT "echo $SUDO_PASS | sudo -S cat /var/log/sssonector/client.log" >> $log_file
    
    return 0
}

# Function to stop services
stop_services() {
    local scenario=$1
    echo "Stopping services for scenario: $scenario"
    
    # Create temporary script
    local TEMP_SCRIPT=$(mktemp)
    cat << 'EOF' > $TEMP_SCRIPT
#!/bin/bash
if [ "$1" = "foreground" ]; then
    sudo pkill -f "sssonector --mode"
    sudo rm -f /var/log/sssonector/*.log
else
    sudo systemctl stop sssonector@$2
fi
EOF
    
    # Copy script to remote systems
    scp $TEMP_SCRIPT $QA_CLIENT:/tmp/stop_service.sh
    scp $TEMP_SCRIPT $QA_SERVER:/tmp/stop_service.sh
    
    if [ "$scenario" = "foreground" ]; then
        ssh $QA_CLIENT "chmod +x /tmp/stop_service.sh && echo $SUDO_PASS | sudo -S /tmp/stop_service.sh foreground"
        ssh $QA_SERVER "chmod +x /tmp/stop_service.sh && echo $SUDO_PASS | sudo -S /tmp/stop_service.sh foreground"
    else
        ssh $QA_CLIENT "chmod +x /tmp/stop_service.sh && echo $SUDO_PASS | sudo -S /tmp/stop_service.sh background client"
        ssh $QA_SERVER "chmod +x /tmp/stop_service.sh && echo $SUDO_PASS | sudo -S /tmp/stop_service.sh background server"
    fi
    
    # Clean up
    ssh $QA_CLIENT "rm /tmp/stop_service.sh"
    ssh $QA_SERVER "rm /tmp/stop_service.sh"
    rm $TEMP_SCRIPT
    
    # Wait for cleanup
    sleep 5
    check_status "Service shutdown"
    
    # Verify cleanup
    echo "Verifying cleanup..."
    ssh $QA_CLIENT "echo $SUDO_PASS | sudo -S ip link show | grep -q tun0" || echo "[✓] Client TUN interface removed"
    ssh $QA_SERVER "echo $SUDO_PASS | sudo -S ip link show | grep -q tun0" || echo "[✓] Server TUN interface removed"
}

# Function to run a test scenario
run_scenario() {
    local server_mode=$1
    local client_mode=$2
    local scenario_name="server_${server_mode}_client_${client_mode}"
    
    echo "=== Running Scenario: $scenario_name ==="
    echo "Starting services..."
    
    # Start server
    run_server $server_mode
    check_status "Server startup"
    
    # Start client
    run_client $client_mode
    check_status "Client startup"
    
    # Test connectivity
    test_connectivity $scenario_name
    check_status "Connectivity test"
    
    # Stop services
    stop_services $scenario_name
    check_status "Service shutdown"
    
    echo "=== Scenario $scenario_name completed ==="
    echo
}

# Main test execution
echo "Starting SSSonector test scenarios..."
date > "$LOG_DIR/test_run.log"

# Scenario 1: Foreground Client/Server
run_scenario "foreground" "foreground"

# Scenario 2: Background Client, Foreground Server
run_scenario "foreground" "background"

# Scenario 3: Background Client/Server
run_scenario "background" "background"

# Generate test report
echo "Generating test report..."
cat << EOF > "$LOG_DIR/test_report.txt"
SSSonector Test Report
=====================
Date: $(date)
Test Environment:
- Server: $QA_SERVER
- Client: $QA_CLIENT
- Ping Count: $PING_COUNT

Test Results:
$(for f in $LOG_DIR/*.log; do
    echo "=== $(basename $f) ==="
    grep -E "^\[.*\]|ping statistics|packets transmitted" "$f"
    echo
done)

System Status:
Server:
$(ssh $QA_SERVER "echo $SUDO_PASS | sudo -S systemctl status sssonector")

Client:
$(ssh $QA_CLIENT "echo $SUDO_PASS | sudo -S systemctl status sssonector")
EOF

echo "Tests completed. Results available in $LOG_DIR/"
