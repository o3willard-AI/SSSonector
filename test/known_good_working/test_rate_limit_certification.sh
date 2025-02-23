#!/bin/bash

# Configuration
CLIENT_IP="192.168.50.211"
SERVER_IP="192.168.50.210"
MONITOR_IP="192.168.50.212"
TEST_FILE="/home/test/data/DryFire_v4_10.zip"
SSSONECTOR_HOME="/home/test/sssonector"
RESULTS_DIR="${SSSONECTOR_HOME}/test_results/rate_limit"
TEST_RATES=(5 25 50 75 100) # Mbps
TEST_DURATION=300 # 5 minutes per test

# Colors for output
GREEN='\033[0;32m'
RED='\033[0;31m'
YELLOW='\033[1;33m'
NC='\033[0m'

# Create results directory
setup_results_dir() {
    local host=$1
    ssh sblanken@${host} "mkdir -p ${RESULTS_DIR}"
}

# Configure rate limit
set_rate_limit() {
    local host=$1
    local rate=$2
    echo -e "${YELLOW}Setting rate limit to ${rate} Mbps on ${host}...${NC}"
    
    ssh sblanken@${host} "cat > ${SSSONECTOR_HOME}/configs/rate_limit.conf << EOF
throttle:
  enabled: true
  rate_mbps: ${rate}
  burst_size: 1048576
EOF"
}

# Start monitoring
start_monitoring() {
    local test_name=$1
    local rate=$2
    echo -e "${YELLOW}Starting monitoring for ${test_name} at ${rate} Mbps...${NC}"
    
    ssh sblanken@${MONITOR_IP} "nohup ${SSSONECTOR_HOME}/monitor \
        --output ${RESULTS_DIR}/${test_name}_${rate}mbps.log \
        --interval 1 > /dev/null 2>&1 &"
}

# Run single direction test
run_direction_test() {
    local src_host=$1
    local dst_host=$2
    local direction=$3
    local rate=$4
    
    echo -e "${GREEN}Starting ${direction} test at ${rate} Mbps...${NC}"
    
    # Set rate limit on source
    set_rate_limit ${src_host} ${rate}
    
    # Start monitoring
    local test_name="${direction}_${rate}"
    start_monitoring ${test_name} ${rate}
    
    # Start transfer
    echo "Starting file transfer..."
    ssh sblanken@${src_host} "time ftp -n ${dst_host} << EOF
quote USER test
quote PASS test
binary
put ${TEST_FILE} test_transfer
quit
EOF" > ${RESULTS_DIR}/${test_name}_transfer.log 2>&1
    
    # Allow time for monitoring to capture final data
    sleep 5
    
    # Stop monitoring
    ssh sblanken@${MONITOR_IP} "pkill -f 'monitor --output'"
    
    echo -e "${GREEN}Completed ${direction} test at ${rate} Mbps${NC}"
}

# Analyze results
analyze_results() {
    local rate=$1
    local direction=$2
    local results_file="${RESULTS_DIR}/${direction}_${rate}mbps.log"
    
    echo -e "${YELLOW}Analyzing results for ${direction} at ${rate} Mbps...${NC}"
    
    # Copy results locally for analysis
    scp sblanken@${MONITOR_IP}:${results_file} ${RESULTS_DIR}/
    
    # Calculate average throughput and variance
    awk '
        BEGIN { sum = 0; sum_sq = 0; n = 0; }
        /throughput/ { 
            sum += $2; 
            sum_sq += $2 * $2; 
            n++; 
        }
        END { 
            if (n > 0) {
                avg = sum/n;
                variance = (sum_sq - (sum * sum)/n)/(n - 1);
                printf "Average throughput: %.2f Mbps\n", avg;
                printf "Variance: %.2f\n", variance;
                printf "Samples: %d\n", n;
            }
        }' ${results_file} > ${RESULTS_DIR}/${direction}_${rate}mbps_analysis.txt
}

# Main test execution
main() {
    echo -e "${GREEN}Starting rate limit certification testing...${NC}"
    
    # Create results directories
    setup_results_dir ${SERVER_IP}
    setup_results_dir ${CLIENT_IP}
    setup_results_dir ${MONITOR_IP}
    
    # Test each rate
    for rate in "${TEST_RATES[@]}"; do
        # Server to Client
        run_direction_test ${SERVER_IP} ${CLIENT_IP} "server_to_client" ${rate}
        analyze_results ${rate} "server_to_client"
        
        # Clean up and wait between tests
        ./cleanup_test_environment.sh
        sleep 10
        
        # Client to Server
        run_direction_test ${CLIENT_IP} ${SERVER_IP} "client_to_server" ${rate}
        analyze_results ${rate} "client_to_server"
        
        # Clean up and wait between rates
        ./cleanup_test_environment.sh
        sleep 10
    done
    
    echo -e "${GREEN}Rate limit certification testing completed${NC}"
    echo "Results available in ${RESULTS_DIR}"
}

# Run the tests
main
