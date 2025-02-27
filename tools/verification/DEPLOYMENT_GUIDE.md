# SSSonector Verification System Deployment Guide

## Overview

This guide documents the successful deployment of the SSSonector Verification System to the QA environment and provides detailed, repeatable instructions for ongoing use and future deployments.

## Deployment Success

The SSSonector Verification System has been successfully deployed to the QA environment:

- **Server**: 192.168.50.210
- **Client**: 192.168.50.211
- **User**: sblanken (previously documented incorrectly as "qauser")

All verification modules are functioning correctly, and initial verification checks have confirmed that both the QA server and client meet the requirements for running SSSonector.

## Deployment Details

### Components Deployed

1. **Core Verification Script**
   - `unified_verifier.sh`: Main verification script
   - Location: `/opt/sssonector/tools/verification/unified_verifier.sh`
   - Symlink: `/usr/local/bin/verify-environment`

2. **Verification Modules**
   - System Module: OpenSSL, TUN, resources verification
   - Network Module: IP forwarding, interfaces, connectivity
   - Security Module: Certificates, memory protections, namespaces
   - Performance Module: System metrics, network performance, limits

3. **Configuration**
   - Environment-specific settings in `environments.yaml`
   - QA-specific thresholds and requirements

4. **Common Utilities**
   - Logging functions
   - Environment detection
   - Result tracking
   - State management

5. **Deployment Scripts**
   - `deploy.sh`: Main deployment script
   - `deploy_to_qa.sh`: QA-specific deployment script

### Directory Structure on QA Systems

```
/opt/sssonector/tools/verification/
├── unified_verifier.sh    # Main verification script
├── config/               # Environment configurations
│   └── environments.yaml
├── lib/                  # Common utilities
│   └── common.sh
├── modules/              # Verification modules
│   ├── system/          # System requirements
│   ├── network/         # Network configuration
│   ├── security/        # Security settings
│   └── performance/     # Performance metrics
└── reports/             # Verification reports
```

## Repeatable Deployment Instructions

To deploy the verification system to a new environment, follow these steps:

### Prerequisites

1. SSH access to target systems
2. Sudo privileges on target systems
3. SSSonector source code repository

### Deployment Steps

1. **Clone the Repository**
   ```bash
   git clone https://github.com/o3willard-AI/SSSonector.git
   cd SSSonector/tools/verification
   ```

2. **Make Scripts Executable**
   ```bash
   chmod +x unified_verifier.sh modules/*/*.sh lib/*.sh deploy.sh
   ```

3. **Configure Target Systems**
   Edit `deploy_to_qa.sh` with the appropriate server details:
   ```bash
   # QA environment details
   QA_SERVER="192.168.50.210"  # Replace with target server IP
   QA_CLIENT="192.168.50.211"  # Replace with target client IP
   QA_USER="sblanken"          # Replace with SSH username
   QA_SUDO_PASSWORD="101abn"   # Replace with sudo password
   ```

4. **Run Deployment Script**
   ```bash
   ./deploy_to_qa.sh
   ```

5. **Verify Deployment**
   After deployment, verify that the system is working correctly:
   ```bash
   # On server
   ssh sblanken@192.168.50.210
   sudo verify-environment --modules system,network

   # On client
   ssh sblanken@192.168.50.211
   sudo verify-environment --modules system,network
   ```

## Usage Guide

### Basic Usage

The verification system can be run on both QA server and client using the `verify-environment` command:

```bash
# Run all verifications
verify-environment

# Run with debug output
verify-environment --debug
```

### Running Specific Modules

You can run specific verification modules:

```bash
# Run only system verification
verify-environment --modules system

# Run system and network verification
verify-environment --modules system,network

# Run all modules except performance
verify-environment --skip performance
```

### Scheduled Verification

For ongoing verification, set up a cron job to run the verification system regularly:

```bash
# Edit crontab
crontab -e

# Add a daily verification at 2 AM
0 2 * * * /usr/local/bin/verify-environment --modules system,network > /var/log/sssonector/verification_$(date +\%Y\%m\%d).log 2>&1
```

### Viewing Reports

Verification reports are stored in the reports directory:

```bash
# List available reports
ls -la /opt/sssonector/tools/verification/reports/

# View the latest report
cat /opt/sssonector/tools/verification/reports/$(ls -t /opt/sssonector/tools/verification/reports/ | head -n1)/report.md
```

### Troubleshooting

If verification fails, check the verification log for details:

```bash
# View the latest verification log
cat /opt/sssonector/tools/verification/reports/$(ls -t /opt/sssonector/tools/verification/reports/ | head -n1)/verification.log
```

Common issues and solutions:

1. **OpenSSL Version**
   ```bash
   # Check OpenSSL version
   openssl version
   
   # Install or update OpenSSL
   sudo apt-get update && sudo apt-get install -y openssl
   ```

2. **TUN Module**
   ```bash
   # Check if TUN module is loaded
   lsmod | grep tun
   
   # Load TUN module
   sudo modprobe tun
   ```

3. **IP Forwarding**
   ```bash
   # Check IP forwarding status
   cat /proc/sys/net/ipv4/ip_forward
   
   # Enable IP forwarding
   echo 1 | sudo tee /proc/sys/net/ipv4/ip_forward
   
   # Make IP forwarding persistent
   echo "net.ipv4.ip_forward = 1" | sudo tee -a /etc/sysctl.conf
   sudo sysctl -p
   ```

4. **File Descriptor Limits**
   ```bash
   # Check current limits
   ulimit -n
   
   # Increase limits
   echo "* soft nofile 65535" | sudo tee -a /etc/security/limits.conf
   echo "* hard nofile 65535" | sudo tee -a /etc/security/limits.conf
   ```

## Integration with CI/CD

The verification system can be integrated into CI/CD pipelines:

```bash
# Example Jenkins pipeline step
stage('Verify Environment') {
  steps {
    sh 'ssh sblanken@192.168.50.210 "sudo verify-environment --modules system,network"'
    sh 'ssh sblanken@192.168.50.211 "sudo verify-environment --modules system,network"'
  }
}
```

## Updating the Verification System

To update the verification system:

1. **Update Local Repository**
   ```bash
   git pull origin main
   cd tools/verification
   ```

2. **Redeploy to QA Environment**
   ```bash
   ./deploy_to_qa.sh
   ```

## Conclusion

The SSSonector Verification System is now fully deployed and operational in the QA environment. It provides a robust framework for validating SSSonector's operating environment, ensuring consistent and reliable operation across different deployments.

By following the repeatable deployment and usage instructions in this guide, you can maintain the verification system and deploy it to additional environments as needed.

---

Document Version: 1.0  
Last Updated: February 25, 2025
