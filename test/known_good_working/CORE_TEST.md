# Core Functionality Sanity Check

This document describes how to run the core functionality sanity check for SSSonector.

## Overview

The core sanity check verifies the following basic functionality:
1. Standard Linux installation and deployment
2. Server and client running in foreground mode
3. Clean tunnel establishment
4. Bidirectional packet transmission (20 packets each way)
5. Clean tunnel shutdown

## Prerequisites

1. Two Linux VMs with:
   - Network connectivity between them
   - SSH access configured
   - Sudo privileges
   - TUN module loaded (`modprobe tun`)

2. Latest SSSonector package installed on both VMs:
   ```bash
   # On both VMs
   sudo dpkg -i sssonector_1.0.0_amd64.deb
   ```

3. Configuration files in place:
   ```bash
   # Server VM
   /etc/sssonector/config.yaml:
   mode: server
   listen: 0.0.0.0:443
   interface: tun0
   address: 10.0.0.1/24

   # Client VM
   /etc/sssonector/config.yaml:
   mode: client
   server: <server-ip>:443
   interface: tun0
   address: 10.0.0.2/24
   ```

## Test Environment Setup

1. Copy `config.env.example` to `config.env`:
   ```bash
   cp config.env.example config.env
   ```

2. Edit `config.env` with your VM details:
   ```bash
   # VM Configuration
   export QA_SERVER_VM="192.168.50.210"  # Server VM IP
   export QA_CLIENT_VM="192.168.50.211"  # Client VM IP
   export QA_SSH_KEY="/path/to/qa_ssh_key"  # SSH private key
   export QA_SSH_USER="qauser"  # SSH username
   ```

## Running the Test

1. Make sure you're in the qa_scripts directory:
   ```bash
   cd test/qa_scripts
   ```

2. Run the core sanity check:
   ```bash
   ./core_sanity_check.sh
   ```

3. Monitor the test output:
   - The script will show progress with timestamped logs
   - Each step's success/failure will be clearly indicated
   - All test artifacts are saved to a timestamped directory

## Test Steps

1. Environment Validation:
   - Verifies SSH connectivity
   - Checks required files and permissions
   - Validates configuration

2. Installation Check:
   - Verifies binary installation
   - Checks service files
   - Validates configurations

3. Process Start:
   - Starts server in foreground mode
   - Starts client in foreground mode
   - Verifies processes are running

4. Tunnel Testing:
   - Verifies tunnel interfaces
   - Tests bidirectional connectivity
   - Collects performance metrics

5. Clean Shutdown:
   - Gracefully stops client
   - Verifies tunnel cleanup
   - Checks for resource leaks

## Test Artifacts

The test creates a directory named `core_test_YYYYMMDD_HHMMSS` containing:
- Application logs (server.log, client.log)
- System logs (journal entries)
- Network state captures
- Tunnel statistics
- Test execution log

## Success Criteria

The test is considered successful if:
1. All installation checks pass
2. Both processes start without errors
3. Tunnel establishes successfully
4. All 40 test packets (20 each way) succeed
5. Client shuts down cleanly
6. No resource leaks are detected

## Troubleshooting

1. SSH Connection Issues:
   - Verify VM IPs in config.env
   - Check SSH key permissions
   - Validate network connectivity

2. Process Start Failures:
   - Check system logs
   - Verify TUN module is loaded
   - Check file permissions

3. Tunnel Issues:
   - Verify network configuration
   - Check firewall rules
   - Review process logs

4. Cleanup Failures:
   - Check for stuck processes
   - Review system logs
   - Verify TUN interface state

## Support

For issues and support:
1. Check the detailed logs in the test output directory
2. Review system logs on both VMs
3. File an issue with the test output attached
