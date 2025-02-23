#!/bin/bash

# Function to generate test files
generate_test_files() {
    dd if=/dev/urandom of=/tmp/test_10mb bs=1M count=10
    dd if=/dev/urandom of=/tmp/test_50mb bs=1M count=50
    dd if=/dev/urandom of=/tmp/test_200mb bs=1M count=200
}

# Function to cleanup
cleanup() {
    echo "Cleaning up..."
    rm -f /tmp/test_*mb
    ssh 192.168.50.210 "pkill -f sssonector-client"
    ssh 192.168.50.212 "pkill -f sssonector-server"
}

# Set up trap for cleanup
trap cleanup EXIT

echo "Starting SNMP bandwidth test sequence..."

# Check if sssonector-control is available
ssh 192.168.50.212 "which sssonector-control"
if [ $? -ne 0 ]; then
    echo "Error: sssonector-control not found on 192.168.50.212. Aborting."
    exit 1
fi

# Generate test files
echo "Generating test files..."
generate_test_files

# Start tunnel server on QA monitor
echo "Starting tunnel server..."
ssh 192.168.50.212 "cd /opt/sssonector && ./sssonector-server -config server.yaml" &
server_pid=$!
sleep 5

# Start tunnel client on test system
echo "Starting tunnel client..."
ssh 192.168.50.210 "cd /opt/sssonector && ./sssonector-client -config client.yaml" &
client_pid=$!
sleep 5

# Function to set rate and check exit code
set_rate() {
    ssh 192.168.50.212 "sssonector-control set-rate $1"
    if [ $? -ne 0 ]; then
        echo "Error setting rate to $1. Aborting."
        cleanup
        exit 1
    fi
}

# Phase 1: Low Bandwidth (1 Mbps)
echo "Phase 1: Low Bandwidth Test (0-30s)"
set_rate 1024 # 1 Mbps
scp /tmp/test_10mb 192.168.50.210:/tmp/
if [ $? -ne 0 ]; then
    echo "Error transferring test_10mb. Aborting."
    cleanup
    exit 1
fi
sleep 30

# Phase 2: Medium Bandwidth (10 Mbps)
echo "Phase 2: Medium Bandwidth Test (30-60s)"
set_rate 10240 # 10 Mbps
scp /tmp/test_50mb 192.168.50.210:/tmp/
if [ $? -ne 0 ]; then
    echo "Error transferring test_50mb. Aborting."
    cleanup
    exit 1
fi
sleep 30

# Phase 3: High Bandwidth (100 Mbps)
echo "Phase 3: High Bandwidth Test (60-90s)"
set_rate 102400 # 100 Mbps
scp /tmp/test_200mb 192.168.50.210:/tmp/
if [ $? -ne 0 ]; then
    echo "Error transferring test_200mb. Aborting."
    cleanup
    exit 1
fi
sleep 30

# Phase 4: Cool Down
echo "Phase 4: Cool Down (90-120s)"
set_rate 1024 # Back to 1 Mbps
sleep 30

echo "Test sequence completed."
cleanup
