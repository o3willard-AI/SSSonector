# Core Functionality Test Script

This directory contains automated test scripts for validating SSSonector's core functionality.

## core_functionality_test.sh

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
