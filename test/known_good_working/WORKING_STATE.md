# SSSonector QA Known Good Working State

This directory contains a snapshot of the QA testing system in a known good working state, verified on 2/22/2025. This can be used as a reference point for troubleshooting or when setting up new QA environments.

## QA Environment Setup

### Test Systems
- Server: 192.168.50.210 (Linux)
- Client: 192.168.50.211 (Linux)
- Both systems run under user 'sblanken' with sudo access

### Directory Structure
```
~/sssonector/
├── bin/           # Binary files (chmod 755)
├── config/        # Configuration files (chmod 644, owned by root:root)
├── certs/         # TLS certificates (chmod 600 for keys, 644 for certs)
├── log/           # Log files (chmod 644)
└── state/         # State files (chmod 644)
```

### Required Permissions
1. sudo access for:
   - TUN interface creation
   - Network configuration
   - Port binding
   - Service management
2. File permissions:
   - Binary: 755 (rwxr-xr-x)
   - Config files: 644 (rw-r--r--), owned by root:root
   - Certificates: 600 (rw-------) for keys, 644 (rw-r--r--) for certs
   - Log directory: 755 (rwxr-xr-x)
   - State directory: 755 (rwxr-xr-x)

### Required System Configuration
1. IP forwarding enabled:
   ```bash
   sysctl net.ipv4.ip_forward=1
   ```
2. TUN module loaded:
   ```bash
   modprobe tun
   ```

## Deployment Process

### Binary Deployment
1. Build from source using build-all.sh
2. Copy binary to ~/sssonector/bin/
3. Set ownership and permissions:
   ```bash
   chown root:root ~/sssonector/bin/sssonector
   chmod 755 ~/sssonector/bin/sssonector
   ```

### Configuration Deployment
1. Copy config files to ~/sssonector/config/
2. Set ownership and permissions:
   ```bash
   chown -R root:root ~/sssonector/config
   chmod 644 ~/sssonector/config/*.yaml
   chmod 755 ~/sssonector/config
   ```

### Certificate Deployment
1. Generate certificates using setup_certificates.sh
2. Certificates are automatically installed with correct permissions

## Test Scripts Evolution

### Initial Issues and Solutions

1. TCP Port Check Issue
   - Initial problem: Tests were failing because they looked for TCP port 8080
   - Root cause: SSSonector uses TUN interfaces for data transfer, not TCP sockets
   - Solution: Updated tests to verify TUN interface configuration instead
   ```bash
   # Old test (failed):
   ssh $SERVER_HOST 'sudo ss -tuln | grep -q :8080'
   
   # New test (works):
   ssh $SERVER_HOST 'ip link show tun0 | grep -q UP && ip addr show tun0 | grep -q "inet 10.0.0.1"'
   ```

2. Certificate Permission Issues
   - Initial problem: Service couldn't read certificates
   - Solution: Implemented proper ownership (root:root) and permissions (600/644)

3. Config File Access
   - Initial problem: Service failed to load config
   - Solution: Set root:root ownership and 644 permissions

### Test Script Components

1. cleanup_resources.sh
   - Ensures clean state before tests
   - Removes existing TUN interfaces
   - Kills any running processes
   - Verifies port availability

2. setup_certificates.sh
   - Generates CA, server, and client certificates
   - Distributes certificates with correct permissions
   - Verifies certificate installation

3. tunnel_control.sh
   - Manages tunnel lifecycle
   - Handles clean startup/shutdown
   - Verifies tunnel status

4. core_functionality_test.sh
   - Comprehensive test suite
   - Verifies all aspects of tunnel operation
   - Uses TUN interface checks instead of TCP ports

## Environment Variables

```bash
# Server Environment
SSSONECTOR_CONFIG=/home/sblanken/sssonector/config/config.yaml
SSSONECTOR_LOG_DIR=/home/sblanken/sssonector/log
SSSONECTOR_STATE_DIR=/home/sblanken/sssonector/state

# Test Environment
SERVER_HOST=sblanken@192.168.50.210
CLIENT_HOST=sblanken@192.168.50.211
```

## Network Configuration

### Server
- TUN Interface: tun0
- TUN IP: 10.0.0.1/24
- Physical Interface: enp0s3
- Physical IP: 192.168.50.210

### Client
- TUN Interface: tun0
- TUN IP: 10.0.0.2/24
- Physical Interface: enp0s3
- Physical IP: 192.168.50.211

## Important Notes

1. SSSonector is installed directly from built binary, not as a package
2. All scripts use BASH for compatibility
3. TUN interface is used for data transfer, not TCP sockets
4. Initial connection setup may use TCP port 8080, but ongoing communication is through TUN
5. All test scripts are idempotent and can be run multiple times
6. Scripts handle their own cleanup to ensure clean state

## Verification Process

To verify a system matches this known good state:

1. Compare file permissions and ownership
2. Verify network configuration
3. Check certificate setup
4. Run the full test suite
5. Compare test results

## Troubleshooting Common Issues

1. Test failures due to incorrect permissions:
   ```bash
   # Fix config permissions
   sudo chown -R root:root ~/sssonector/config
   sudo chmod 644 ~/sssonector/config/*.yaml
   sudo chmod 755 ~/sssonector/config
   
   # Fix certificate permissions
   sudo chown -R root:root ~/sssonector/certs
   sudo chmod 600 ~/sssonector/certs/*.key
   sudo chmod 644 ~/sssonector/certs/*.crt
   sudo chmod 755 ~/sssonector/certs
   ```

2. TUN interface issues:
   ```bash
   # Remove existing interface
   sudo ip link delete tun0
   
   # Verify TUN module
   lsmod | grep tun
   ```

3. Connection issues:
   ```bash
   # Check IP forwarding
   sysctl net.ipv4.ip_forward
   
   # Verify routes
   ip route show
