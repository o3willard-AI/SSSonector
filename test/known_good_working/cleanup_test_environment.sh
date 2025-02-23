#!/bin/bash

# Configuration
CLIENT_IP="192.168.50.211"
SERVER_IP="192.168.50.210"
MONITOR_IP="192.168.50.212"
TEST_FILE="/home/test/data/DryFire_v4_10.zip"
SSSONECTOR_HOME="/home/test/sssonector"

# Colors for output
GREEN='\033[0;32m'
RED='\033[0;31m'
YELLOW='\033[1;33m'
NC='\033[0m'

echo -e "${GREEN}Starting comprehensive test environment cleanup...${NC}"

# Function to verify test file exists
verify_test_file() {
    local host=$1
    echo -e "${YELLOW}Verifying test file on $host...${NC}"
    if ssh sblanken@${host} "[ -f ${TEST_FILE} ]"; then
        echo -e "${GREEN}Test file exists on $host${NC}"
        return 0
    else
        echo -e "${RED}Test file missing on $host${NC}"
        return 1
    fi
}

# Function to cleanup a remote system
cleanup_system() {
    local host=$1
    local name=$2
    echo -e "${YELLOW}Cleaning up $name ($host)...${NC}"
    
    # Kill all test-related processes
    ssh sblanken@${host} "sudo pkill -f sssonector; \
                         sudo pkill -f 'nc -l'; \
                         sudo pkill -f 'test_rate_limit'; \
                         sudo pkill -f 'monitor_snmp'; \
                         sudo pkill -f 'vsftpd'"

    # Clean up test directories and files
    ssh sblanken@${host} "rm -rf ${SSSONECTOR_HOME}/logs/*; \
                         rm -f ~/test_data_generator.py ~/tunnel_monitor.py ~/monitor; \
                         rm -f ~/.sssonector/test_*.log; \
                         sudo rm -f /var/log/sssonector/*.log; \
                         rm -f ~/rate_limit_*.log; \
                         rm -f ~/test_results_*.txt"

    # Reset rate limiting configuration
    ssh sblanken@${host} "if [ -f ${SSSONECTOR_HOME}/configs/rate_limit.conf ]; then \
                           sudo rm -f ${SSSONECTOR_HOME}/configs/rate_limit.conf; \
                         fi"

    # Clean up any temporary certificates
    ssh sblanken@${host} "rm -f ${SSSONECTOR_HOME}/certs/temp_*.pem"

    # Reset FTP configuration to default
    ssh sblanken@${host} "if [ -f /etc/vsftpd.conf ]; then \
                           sudo cp /etc/vsftpd.conf.orig /etc/vsftpd.conf 2>/dev/null; \
                         fi"

    echo -e "${GREEN}Cleanup completed on $name${NC}"
}

# Function to verify system is ready for testing
verify_system() {
    local host=$1
    local name=$2
    echo -e "${YELLOW}Verifying $name ($host) is ready...${NC}"

    # Check SSH connectivity
    if ! ssh -q sblanken@${host} "exit"; then
        echo -e "${RED}SSH connection failed to $name${NC}"
        return 1
    fi

    # Check if sssonector directory exists
    if ! ssh sblanken@${host} "[ -d ${SSSONECTOR_HOME} ]"; then
        echo -e "${RED}SSSonector directory missing on $name${NC}"
        return 1
    fi

    echo -e "${GREEN}System verification passed for $name${NC}"
    return 0
}

# Main cleanup process
echo "Starting cleanup process..."

# Clean up each system
for system in "Server:${SERVER_IP}" "Client:${CLIENT_IP}" "Monitor:${MONITOR_IP}"; do
    name=${system%:*}
    ip=${system#*:}
    
    cleanup_system ${ip} ${name}
    if ! verify_system ${ip} ${name}; then
        echo -e "${RED}Failed to verify $name system${NC}"
        exit 1
    fi
done

# Verify test file exists on both server and client
verify_test_file ${SERVER_IP} || echo -e "${RED}Warning: Test file missing on server${NC}"
verify_test_file ${CLIENT_IP} || echo -e "${RED}Warning: Test file missing on client${NC}"

echo -e "${GREEN}Cleanup completed successfully!${NC}"
echo
echo "Environment is ready for rate limiting certification testing."
echo "To deploy test environment:"
echo "./deploy_test_environment.sh"
