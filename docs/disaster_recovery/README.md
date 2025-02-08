# SSSonector Disaster Recovery System

## Overview
This disaster recovery system provides comprehensive backup, restore, and validation capabilities for the SSSonector project. It ensures that both the development environment and project state can be reliably recovered in case of system failure or data loss.

## Directory Structure
```
disaster_recovery/
├── project/                      # Project state documentation
│   ├── project_state.md         # Current development status
│   ├── qa_environment.md        # QA environment details
│   ├── development_threads.md   # Active development threads
│   └── recovery_instructions.md # Recovery procedures
├── backups/                     # Backup storage
│   ├── source/                  # Source code backups
│   ├── configs/                 # Configuration backups
│   ├── tests/                   # Test data backups
│   └── qa_data/                # QA environment backups
└── scripts/                     # Recovery scripts
    ├── backup.sh               # Backup script
    ├── restore.sh              # Restore script
    └── validate.sh             # Validation script
```

## Quick Start

### Creating a Backup
```bash
cd /home/sblanken/Desktop/go/src/github.com/o3willard-AI/SSSonector
./docs/disaster_recovery/scripts/backup.sh
```

### Restoring from Backup
```bash
# Restore latest backup
./docs/disaster_recovery/scripts/restore.sh

# Restore specific backup
./docs/disaster_recovery/scripts/restore.sh 20250206_123456
```

### Validating Environment
```bash
./docs/disaster_recovery/scripts/validate.sh
```

## Backup Contents

### 1. Source Code
- Complete project source tree
- Build configurations
- Dependencies
- Test suites

### 2. Configurations
- Project configurations
- SNMP settings
- Test environment configs
- Build settings

### 3. Test Data
- Test certificates
- Test files
- QA data
- Validation data

### 4. QA Environment
- VM configurations
- Test scripts
- Environment variables
- Service settings

## Recovery Process

### 1. Environment Setup
- Verify Go installation (1.21+)
- Check system packages
- Validate TUN module
- Configure network

### 2. Project Recovery
- Restore source code
- Rebuild project
- Run tests
- Verify functionality

### 3. QA Environment Recovery
- Restore VM configurations
- Configure services
- Verify connectivity
- Run validation tests

## Validation Checks

### 1. Environment Validation
- Go version check
- Package verification
- Module validation
- Directory structure

### 2. Build Validation
- Dependency check
- Build process
- Test execution
- Performance metrics

### 3. QA Validation
- VM connectivity
- Service status
- SNMP functionality
- ntopng integration

## Backup Schedule
- Daily automated backups
- Keep last 5 backups
- Manual backup before major changes
- Validation after each backup

## Recovery Time Objectives
1. Development Environment: < 30 minutes
2. QA Environment: < 1 hour
3. Full System: < 2 hours

## Prerequisites

### Software Requirements
- Go 1.21+
- Ubuntu 24.04
- VirtualBox 7.0+
- iproute2
- snmpd
- ntopng

### System Requirements
- 16GB RAM minimum
- 100GB disk space
- TUN/TAP kernel module
- Virtualization support

## Troubleshooting

### Common Issues
1. TUN module not loaded
   ```bash
   sudo modprobe tun
   ```

2. Permission issues
   ```bash
   sudo chown -R $(whoami):$(whoami) /home/test/
   ```

3. Service failures
   ```bash
   sudo systemctl restart snmpd
   sudo systemctl restart ntopng
   ```

### Recovery Verification
- Check build status
- Verify service connectivity
- Run test suite
- Validate metrics

## Emergency Contacts
- Development Team: [Contact Info]
- Infrastructure Team: [Contact Info]
- QA Team: [Contact Info]

## Documentation
- Project Documentation: /docs/
- QA Environment: /docs/virtualbox_testing.md
- Test Results: /test/test_results.md
- Build Instructions: /docs/installation.md

## Recovery Checklist
1. [ ] Verify environment prerequisites
2. [ ] Run restore script
3. [ ] Validate restoration
4. [ ] Check QA environment
5. [ ] Run test suite
6. [ ] Verify services
7. [ ] Document recovery

## Best Practices
1. Regular backup verification
2. Keep documentation current
3. Test recovery procedures
4. Monitor backup status
5. Update contact information
6. Document changes
7. Maintain scripts

## Script Usage

### backup.sh
```bash
# Create backup
./backup.sh

# Backup specific components
./backup.sh --source-only
./backup.sh --config-only
./backup.sh --qa-only
```

### restore.sh
```bash
# Restore latest
./restore.sh

# Restore specific backup
./restore.sh 20250206_123456

# Restore specific components
./restore.sh --source-only 20250206_123456
```

### validate.sh
```bash
# Full validation
./validate.sh

# Specific checks
./validate.sh --environment
./validate.sh --build
./validate.sh --qa
