# QA Testing Scripts

This directory contains scripts for automated testing of SSSonector in a QA environment.

## Security Improvements

Recent security improvements include:
- Removed hardcoded credentials
- Implemented SSH key-based authentication
- Added environment validation
- Improved error handling and logging
- Created shared functions for common operations

## Setup Instructions

1. Configure SSH Key Authentication:
   ```bash
   # Generate SSH key pair for QA testing
   ssh-keygen -t ed25519 -f ~/.ssh/qa_ssh_key -C "qa@sssonector"
   
   # Copy public key to QA VMs
   ssh-copy-id -i ~/.ssh/qa_ssh_key.pub qauser@192.168.50.210
   ssh-copy-id -i ~/.ssh/qa_ssh_key.pub qauser@192.168.50.211
   ```

2. Configure Environment:
   - Copy `config.env.example` to `config.env`
   - Update variables in `config.env`:
     ```bash
     # VM Configuration
     export QA_SERVER_VM="192.168.50.210"
     export QA_CLIENT_VM="192.168.50.211"
     export QA_SSH_KEY="/path/to/.ssh/qa_ssh_key"
     export QA_SSH_USER="qauser"
     
     # Test Parameters
     export QA_LOG_DIR="sanity_test_logs"
     export QA_PING_COUNT=20
     export QA_TUNNEL_TIMEOUT=30
     export QA_CLEANUP_TIMEOUT=10
     ```

3. Set File Permissions:
   ```bash
   chmod 600 ~/.ssh/qa_ssh_key
   chmod +x sanity_check.sh cleanup.sh
   ```

## Script Overview

### common.sh
- Shared functions for QA testing
- Logging utilities
- Remote command execution
- Installation verification
- Tunnel verification
- Connectivity testing
- Process cleanup verification
- Log collection

### sanity_check.sh
- Main test orchestration
- Runs multiple test scenarios:
  1. Both server and client in foreground
  2. Mixed mode (server foreground, client background)
  3. Both server and client as services
- Verifies installation, connectivity, and cleanup

### cleanup.sh
- Aggressive cleanup of SSSonector processes
- Removes TUN interfaces
- Cleans up system files
- Resets systemd services
- Verifies complete cleanup

## Usage

1. Run Sanity Check:
   ```bash
   ./sanity_check.sh
   ```

2. Run Cleanup:
   ```bash
   ./cleanup.sh
   ```

## Logs

Test logs are stored in the configured `QA_LOG_DIR` with the following structure:
```
sanity_test_logs/
├── scenario1_foreground_both_YYYYMMDD_HHMMSS/
│   ├── server_journal.log
│   ├── client_journal.log
│   ├── server_app.log
│   ├── client_app.log
│   ├── server_network.log
│   ├── client_network.log
│   ├── server_process.log
│   ├── client_process.log
│   ├── server_stats.log
│   ├── client_stats.log
│   ├── server_resources.log
│   └── client_resources.log
└── ...
```

## Error Handling

The scripts include comprehensive error handling:
- Environment validation
- SSH connection verification
- Process startup confirmation
- Tunnel establishment checks
- Connectivity verification
- Cleanup confirmation

## Troubleshooting

1. SSH Connection Issues:
   - Verify SSH key permissions (should be 600)
   - Check VM connectivity
   - Verify user permissions on VMs

2. Process Cleanup Issues:
   - Check systemd service status
   - Verify sudo permissions for cleanup operations
   - Use cleanup.sh with --force flag if needed
   - Check system logs for stuck processes

3. Tunnel Interface Issues:
   - Verify network interface permissions
   - Check kernel module availability (tun)
   - Verify network configuration
   - Check routing table conflicts

4. Log Collection Issues:
   - Verify disk space availability
   - Check file permissions in log directories
   - Ensure journald service is running
   - Verify log rotation settings

## Recent Changes

### February 2025
1. Security Enhancements:
   - Removed hardcoded sudo password
   - Implemented SSH key-based authentication
   - Added environment validation
   - Improved error handling

2. Testing Improvements:
   - Added shared functions library (common.sh)
   - Enhanced logging with severity levels
   - Added detailed process verification
   - Improved cleanup procedures
   - Added comprehensive log collection

3. Documentation:
   - Added setup instructions
   - Improved troubleshooting guide
   - Added log structure documentation
   - Added usage examples

## Contributing

When making changes to the QA scripts:
1. Update config.env.example if adding new variables
2. Test changes in isolation before committing
3. Update documentation as needed
4. Follow the established logging patterns
5. Maintain backward compatibility when possible

## Future Improvements

1. Testing Enhancements:
   - Add performance testing scenarios
   - Implement parallel test execution
   - Add network condition simulation
   - Enhance metric collection

2. Automation:
   - Add CI/CD integration
   - Implement automatic VM provisioning
   - Add test result reporting
   - Implement automatic issue creation

3. Monitoring:
   - Add real-time test progress monitoring
   - Implement test result visualization
   - Add performance trend analysis
   - Enhance error reporting

## Deprecated Scripts

The following scripts have been deprecated and moved to backup as they do not utilize 
the new reliability improvements:

1. Old test scripts without state transition handling:
   - test_ssh_auth.exp
   - test_local_auth.exp
   - test_remote_auth.exp
   - verify_ssh_basic.exp
   - verify_snmp_basic.exp

2. Scripts now covered by core_sanity_check.sh:
   - sanity_check.sh
   - test_scenarios.sh
   - test_rate_limit.sh

3. Deprecated setup scripts:
   - setup_sssonector_repo.exp
   - setup_sssonector_repo2.exp
   - verify_automation.exp

These scripts have been replaced by the enhanced reliability testing framework in:
- server_sanity_check.sh
- core_sanity_check.sh

The new scripts include:
- State transition verification
- Resource cleanup validation
- Connection tracking
- Statistics monitoring
