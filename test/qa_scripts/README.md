# QA Test Scripts

This directory contains automated test and setup scripts for validating SSSonector's functionality.

## Setup Scripts

### setup_ssh_auth.sh
Configures SSH authentication between test machines:
- Generates SSH keys if needed
- Distributes public keys
- Verifies SSH connectivity
- Sets up passwordless sudo access for test automation

### setup_sudo_access.sh
Configures sudo access for test automation:
- Creates sudoers entries for test user
- Sets up NOPASSWD access for specific commands
- Validates sudo configuration
- Ensures proper permissions

### setup_certificates.sh
Manages TLS certificates for secure communication:
- Generates CA, server, and client certificates
- Sets up certificate chains
- Configures certificate locations
- Validates certificate setup

### setup_configs.sh
Handles configuration file management:
- Creates server and client configurations
- Sets up rate limiting parameters
- Configures monitoring settings
- Validates configuration syntax

### setup_systemd.sh
Sets up systemd service integration:
- Creates service unit files
- Configures service dependencies
- Sets up logging
- Validates service operation

### setup_binary.sh
Manages binary installation and updates:
- Installs SSSonector binaries
- Sets up proper permissions
- Creates necessary directories
- Validates binary installation

## Core Functionality Test Script

### core_functionality_test.sh

This script performs sanity checks of core SSSonector functionality across three deployment scenarios:

1. Foreground Client / Foreground Server
2. Background Client / Foreground Server
3. Background Client / Background Server

### Prerequisites

1. Two Linux machines with:
   - SSH access configured between them
   - SSSonector installed
   - Proper configuration files in `/etc/sssonector/`
   - Systemd service configured for background operation
   - Sudo access for network interface management

2. Network connectivity between the machines

### Usage

1. Set environment variables (optional):
   ```bash
   export SERVER_IP="192.168.50.210"  # Default server IP
   export CLIENT_IP="192.168.50.211"  # Default client IP
   ```

2. Run the test script:
   ```bash
   ./core_functionality_test.sh
   ```

### Test Scenarios

For each scenario, the script:
1. Verifies SSSonector installation
2. Starts server and client in specified modes
3. Verifies tunnel establishment
4. Sends 20 test packets in each direction
5. Performs clean shutdown
6. Collects and saves logs

### Test Artifacts

The script creates a timestamped directory under `test_logs/` containing:
- Packet transmission logs
- Server and client application logs
- System journal entries
- Test execution logs

### Exit Codes

- 0: All tests passed
- 1: One or more tests failed

### Troubleshooting

1. If the script fails to connect to hosts:
   - Verify SSH connectivity
   - Check IP addresses
   - Ensure proper SSH key configuration

2. If packet tests fail:
   - Check network connectivity
   - Verify SSSonector configuration
   - Review logs in test_logs directory

3. If services fail to start/stop:
   - Check systemd service configuration
   - Verify permissions
   - Review system logs

### Log Collection

Logs are collected in:
```
test_logs/
└── YYYYMMDD_HHMMSS/
    ├── fg_client_fg_server_TIMESTAMP/
    │   ├── server/
    │   ├── client/
    │   ├── server_journal.log
    │   └── client_journal.log
    ├── bg_client_fg_server_TIMESTAMP/
    │   └── ...
    └── bg_client_bg_server_TIMESTAMP/
        └── ...
```

## Test Environment Setup

To set up a complete test environment:

1. Run setup scripts in order:
```bash
./setup_ssh_auth.sh
./setup_sudo_access.sh
./setup_certificates.sh
./setup_configs.sh
./setup_systemd.sh
./setup_binary.sh
```

2. Verify setup:
```bash
# Check SSH connectivity
ssh $SERVER_IP "echo 'SSH access working'"
ssh $CLIENT_IP "echo 'SSH access working'"

# Verify sudo access
ssh $SERVER_IP "sudo -n true"
ssh $CLIENT_IP "sudo -n true"

# Check systemd service
systemctl status sssonector
```

3. Run core functionality test:
```bash
./core_functionality_test.sh
```

## Automated Test Execution

For CI/CD environments, all scripts support non-interactive execution:
```bash
# Set up complete test environment
for script in setup_*.sh; do
  ./$script -y
done

# Run tests with automatic cleanup
./core_functionality_test.sh --ci
```

## Log Management

Test logs are automatically rotated and compressed after 7 days:
```bash
find test_logs/ -type f -name "*.log" -mtime +7 -exec gzip {} \;
```

## Support and Maintenance

- Documentation: https://docs.sssonector.io/qa
- Issue Tracker: https://github.com/o3willard-AI/SSSonector/issues
- QA Team Contact: qa@sssonector.io
