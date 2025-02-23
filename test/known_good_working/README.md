# Known Good Working State

This directory contains a snapshot of the SSSonector QA testing system in a known good working state, captured on 2/22/2025. It serves as a reference point for troubleshooting and verification.

## Purpose

1. Provide a baseline for comparing test environments
2. Document working configuration and permissions
3. Capture solutions to common issues
4. Enable quick recovery from broken states

## Directory Contents

- `WORKING_STATE.md` - Comprehensive documentation of the working state
- `verify_state.sh` - Automated script to verify environment against known good state
- QA test scripts in their working state:
  - `cleanup_resources.sh` - Resource cleanup script
  - `setup_certificates.sh` - Certificate generation and deployment
  - `tunnel_control.sh` - Tunnel lifecycle management
  - `core_functionality_test.sh` - Main test suite
  - Other supporting scripts

### Quick Verification

To quickly verify if your environment matches this known good state:

```bash
# Make the script executable if needed
chmod +x verify_state.sh

# Run the verification
./verify_state.sh
```

The script will:
1. Check file permissions and ownership
2. Verify network configuration
3. Test connectivity
4. Compare script versions
5. Run the full test suite
6. Provide a detailed report of any differences

## Using This Reference

### For Troubleshooting

1. Compare your current scripts with these reference versions:
   ```bash
   diff -r /path/to/your/qa_scripts/ /path/to/known_good_working/
   ```

2. Compare file permissions:
   ```bash
   # In your environment
   find /home/sblanken/sssonector -type f -exec stat -c "%n %a" {} \; > current_perms.txt
   
   # Compare with documented permissions in WORKING_STATE.md
   ```

3. Verify network configuration matches documented state:
   ```bash
   ip addr show tun0
   ip route show
   ```

### For New Deployments

1. Use these scripts as templates
2. Follow the deployment process in WORKING_STATE.md
3. Verify against the documented working state

### Key Insights

1. TUN Interface vs TCP Port
   - The system uses TUN interfaces for data transfer
   - Port 8080 is only used for initial setup
   - Tests should verify TUN interface state, not TCP ports

2. Permission Requirements
   - Config files must be owned by root:root
   - Certificate keys require 600 permissions
   - Binary needs execute permissions (755)

3. System Requirements
   - IP forwarding must be enabled
   - TUN module must be loaded
   - sudo access required for network operations

## Maintaining This Reference

1. Only update this directory when a new known good state is established
2. Document any changes in WORKING_STATE.md
3. Keep script versions synchronized
4. Verify all tests pass before updating

## Comparing Against This State

To verify if your environment matches this known good state:

1. Run the test suite:
   ```bash
   ./core_functionality_test.sh
   ```

2. Compare configurations:
   ```bash
   # Compare script versions
   diff -r /path/to/your/qa_scripts/ .

   # Compare permissions
   find /home/sblanken/sssonector -type f -exec stat -c "%n %a" {} \; > current_perms.txt
   # Compare with permissions listed in WORKING_STATE.md
   ```

3. Verify network setup:
   ```bash
   # Check TUN interface
   ip addr show tun0
   
   # Verify routes
   ip route show
   ```

4. Check system configuration:
   ```bash
   # IP forwarding
   sysctl net.ipv4.ip_forward
   
   # TUN module
   lsmod | grep tun
   ```

## Important Notes

1. This is a snapshot of a working state - not all possible configurations
2. Scripts are specific to Linux systems
3. IP addresses and paths are specific to our QA environment
4. Adjust paths and IPs as needed for your environment
5. Always verify changes in a test environment first
