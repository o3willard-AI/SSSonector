# QA Environment Validator

Part of Project SENTINEL - SSSonector ENvironment Testing & Integration Layer

## Overview

The QA Environment Validator ensures consistent and reliable testing environments by validating all aspects of a SSSonector deployment. It performs comprehensive checks on system tools, directory structure, permissions, configurations, certificates, network settings, and binaries.

## Features

1. System Tool Validation
   - Verifies presence of required tools
   - Checks tool executability
   - Validates tool versions

2. Directory Structure Validation
   - Verifies required directories exist
   - Checks directory hierarchy
   - Validates directory permissions

3. Permission Validation
   - Directory permissions (755)
   - Configuration files (644)
   - Certificates (644)
   - Private keys (600)
   - Binaries (755)

4. Configuration Validation
   - YAML syntax validation
   - Required fields verification
   - Value constraints checking
   - Security policy compliance

5. Certificate Validation
   - CA certificate verification
   - Certificate chain validation
   - Certificate expiration checking
   - Key pair validation

6. Network Validation
   - Port availability checking
   - Network configuration validation
   - Service accessibility verification

7. Binary Validation
   - Binary presence verification
   - Version information checking
   - Checksum validation
   - Executable permissions

## Usage

```bash
# Make script executable
chmod +x qa_validator.sh

# Run validation with default path
./qa_validator.sh

# Run validation with custom path
./qa_validator.sh -d /custom/path/to/sssonector

# Show help
./qa_validator.sh -h
```

## Exit Codes

- 0: All validations passed
- 1: One or more validations failed

## Directory Structure

The validator expects the following directory structure:
```
/opt/sssonector/          # Default base directory
├── bin/                  # Binary executables
├── config/              # Configuration files
├── certs/               # Certificates and keys
├── log/                 # Log files
├── state/               # Runtime state
└── tools/               # Utility scripts
```

## Required Tools

The validator checks for these system tools:
- yq: YAML processing
- openssl: Certificate operations
- netstat: Network validation
- curl: HTTP checks
- sha256sum: Checksum validation

## Integration

This tool is typically used:
1. Before running tests
2. After environment setup
3. During CI/CD pipelines
4. After configuration changes

## Error Handling

The validator provides detailed error messages for:
- Missing tools or directories
- Invalid permissions
- Configuration errors
- Certificate issues
- Network problems
- Binary validation failures

## Dependencies

- bash
- yq (for YAML processing)
- openssl (for certificate validation)
- netstat (for network validation)
- curl (for HTTP checks)
- sha256sum (for checksum validation)

## Security Considerations

1. Permission Requirements
   - Must run with sufficient permissions to read all files
   - Requires access to certificates and private keys
   - Needs network access for port checking

2. Certificate Handling
   - Validates certificate chain integrity
   - Checks certificate permissions
   - Verifies private key security

3. Configuration Security
   - Validates security-related settings
   - Checks for sensitive information
   - Verifies access controls

## Version History

- 1.0.0: Initial release
  - Basic environment validation
  - Configuration checking
  - Certificate validation
  - Network validation
  - Binary verification

## Future Enhancements

1. Additional Validations
   - Process monitoring
   - Resource availability
   - System requirements
   - Dependency versions

2. Integration Features
   - CI/CD pipeline integration
   - Automated remediation
   - Reporting capabilities
   - Metric collection

3. Enhanced Security
   - TPM verification
   - Secure boot validation
   - Runtime integrity checking
   - Policy enforcement
