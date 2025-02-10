# QA Environment Documentation

This document provides detailed information about the SSSonector QA test environment setup and configuration.

## QA Test Systems

### 1. Server System (192.168.50.210)
- **Purpose**: Testing SSSonector server mode functionality
- **Access**:
  - SSH certificate authentication from Dev system
  - Fallback credentials:
    - Username: sblanken
    - Password: 101abn
    - Sudo password: 101abn (can be used without human approval)
- **Test Resources**:
  - FTP server installed
  - Large test file: `/home/sblanken/DryFire_v4_10.zip`

### 2. Client System (192.168.50.211)
- **Purpose**: Testing SSSonector client mode functionality
- **Access**:
  - SSH certificate authentication from Dev system
  - Fallback credentials:
    - Username: sblanken
    - Password: 101abn
    - Sudo password: 101abn (can be used without human approval)
- **Test Resources**:
  - FTP server installed
  - Large test file: `/home/sblanken/DryFire_v4_10.zip`

### 3. Monitor System (192.168.50.212)
- **Purpose**: Testing SSSonector SNMP service compatibility with common SNMP servers/collectors
- **Access**:
  - SSH certificate authentication from Dev system
  - Fallback credentials:
    - Username: sblanken
    - Password: 101abn
    - Sudo password: 101abn (can be used without human approval)

## QA Process

### Build and Deployment
1. Build OS binaries locally on Dev system
2. Use deployment scripts to copy binaries to QA test systems
3. Clean up existing SSSonector applications and artifacts between builds
4. Deploy and test new build

### Key Scripts
- `deploy_qa.sh`: Deploys SSSonector to QA environment
- `cleanup_test_environment.sh`: Removes all SSSonector artifacts from QA systems
- `server_sanity_check.sh`: Validates server deployment and basic functionality
- Various test scripts in `/test/qa_scripts/` directory

### Environment Management
- All systems accessible via SSH certificates for automated deployment
- Sudo access available for all required operations
- Clean environment ensured between test runs
- Consistent test data across systems

## Best Practices

1. Environment Preparation
   - Always clean up before new deployments
   - Verify SSH access before starting tests
   - Check system resources and services

2. Test Execution
   - Follow test procedures in qa_guide.md
   - Document any deviations from standard process
   - Maintain test logs and results

3. Issue Reporting
   - Include specific system information
   - Document exact steps to reproduce
   - Capture relevant logs and metrics

## Environment Maintenance

1. Regular Tasks
   - Verify SSH certificates
   - Check disk space and system resources
   - Update test data files as needed
   - Validate FTP server functionality

2. System Updates
   - Coordinate system updates with test schedule
   - Verify SSSonector compatibility after updates
   - Update documentation for any system changes

## Reference Information

### Network Configuration
```
Dev System -> QA Network (192.168.50.0/24)
├── Server  : 192.168.50.210
├── Client  : 192.168.50.211
└── Monitor : 192.168.50.212
```

### Required Services
- SSH server (all systems)
- FTP server (Server and Client systems)
- SNMP services (Monitor system)

### Test Resources
- Standard test file: DryFire_v4_10.zip
- Test scripts directory: /test/qa_scripts/
- Configuration templates: /test/qa_scripts/config/
