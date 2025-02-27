# SSSonector Verification Tools

This directory contains tools for verifying the functionality of SSSonector, a high-performance, enterprise-grade communications utility designed to allow critical services to connect to and exchange data with one another over the public internet without needing a VPN.

## Overview

The verification tools are designed to help you:

1. Set up a QA environment for testing SSSonector
2. Fix code issues that may be causing connectivity problems
3. Run comprehensive tests to verify SSSonector functionality
4. Generate detailed test reports

## Quick Start

To get started with the verification tools, follow these steps:

1. Run the setup script to prepare your QA environment:
   ```bash
   ./setup_qa_environment.sh
   ```

2. Edit the QA environment configuration file:
   ```bash
   nano qa_environment.conf
   ```

3. Fix the transfer logic in SSSonector:
   ```bash
   ./fix_transfer_logic.sh
   ```

4. Run the enhanced QA tests:
   ```bash
   ./enhanced_qa_testing.sh
   ```

5. Review the test results in the `qa_results_*` directory.

## Scripts

### setup_qa_environment.sh

This script sets up the QA environment for SSSonector testing. It:

- Makes other scripts executable
- Creates a QA environment configuration file
- Checks if required tools (Go, sshpass, openssl) are installed
- Checks if the SSSonector binary exists and is executable

### fix_transfer_logic.sh

This script fixes the transfer logic in SSSonector. It:

- Backs up files before making changes
- Improves error handling in the `copy` function
- Adds debug logging for packet transmission
- Improves buffer handling
- Adds flush mechanism to ensure packets are sent immediately
- Adds retry mechanism for failed writes
- Improves test code to better simulate real-world behavior
- Adds synchronization to prevent race conditions
- Improves connection retry logic
- Adds more detailed logging for connection attempts and tunnel establishment

### enhanced_qa_testing.sh

This script runs comprehensive QA tests for SSSonector. It:

- Loads configuration from `qa_environment.conf`
- Validates the QA environment
- Cleans up the QA environment
- Generates certificates
- Creates configuration files
- Deploys SSSonector
- Applies network fixes
- Runs test scenarios
- Collects logs and packet captures
- Generates a test report

## Configuration

The QA environment configuration file (`qa_environment.conf`) contains:

- QA server details (IP address, username, password)
- Test settings (timeout, packet count, retry count)

Example:
```
# SSSonector QA Environment Configuration
# Edit this file to match your environment

# QA environment details
QA_SERVER="192.168.50.210"
QA_CLIENT="192.168.50.211"
QA_USER="sblanken"
QA_SUDO_PASSWORD="your_password_here"

# Test settings
TEST_TIMEOUT=300  # Timeout in seconds
PACKET_COUNT=20   # Number of packets to send in each direction
RETRY_COUNT=3     # Number of retries for failed tests
```

## Test Scenarios

The enhanced QA testing script runs the following test scenarios:

1. **Scenario 1**: Client foreground / Server foreground
2. **Scenario 2**: Client background / Server foreground
3. **Scenario 3**: Client background / Server background

Each scenario tests:
- Starting the server and client
- Checking tunnel interfaces
- Testing connectivity (ping from client to server and server to client)
- Stopping the client and server
- Collecting logs

## Test Results

The test results are stored in the `qa_results_*` directory, which contains:

- Logs for each test scenario
- Packet captures
- Test report

The test report includes:
- Test information (date, time, server, client, etc.)
- Test results for each scenario
- Logs and packet captures
- Next steps

## Documentation

For more detailed information, see the following documents:

- [QA Testing Plan](QA_TESTING_PLAN.md): A comprehensive plan for revamping the SSSonector QA testing process and addressing connectivity issues.
- [Last Mile Connectivity Guide](LAST_MILE_CONNECTIVITY.md): Information about last mile connectivity issues and how to resolve them.
- [Connectivity Investigation Report](CONNECTIVITY_INVESTIGATION_REPORT.md): Documentation of the investigation into connectivity issues.

## Troubleshooting

If you encounter issues with the verification tools, check the following:

1. **SSH Connectivity**: Ensure that you can SSH to the QA server and client without a password using sshpass.
2. **Sudo Access**: Ensure that the QA user has sudo access on the QA server and client.
3. **Firewall Rules**: Ensure that the firewall rules allow ICMP traffic and traffic on the tun0 interface.
4. **IP Forwarding**: Ensure that IP forwarding is enabled on the QA server and client.
5. **TUN Module**: Ensure that the TUN module is loaded on the QA server and client.

If you still encounter issues, check the logs in the `qa_results_*` directory for more information.
