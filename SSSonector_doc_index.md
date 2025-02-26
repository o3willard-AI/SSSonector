# SSSonector Documentation Index

## Core Documentation

1. [README.md](README.md)
   - Overview
   - Installation
   - Usage
   - Configuration
   - Testing
   - Security Features
   - Performance
   - Requirements
   - Contributing
   - License

2. [PREREQUISITES.md](docs/PREREQUISITES.md)
   - System Requirements
   - Network Requirements
   - IP Forwarding
   - Firewall Rules
   - Port Requirements
   - TUN Interface
   - Verification
   - Troubleshooting

3. [BUGFIX_REPORT_2025_02_25.md](docs/BUGFIX_REPORT_2025_02_25.md)
   - Race Condition Fixes in Transfer Start/Stop Methods
   - Mock Connection EOF Handling Improvements
   - Client-Server Communication Asymmetry Analysis
   - Build Information for All OS Versions
   - Test Results and Next Steps

### Test Documentation
1. [CORE_TEST](test/known_good_working/CORE_TEST.md)
   - Core Functionality Test

2. [Test Framework](test/README.md)
   - Framework Overview
   - Test Structure
   - Scenario Descriptions
   - Usage Instructions
   - Configuration Guide
   - Troubleshooting

3. [Test Scenarios]
   - [Certificate Generation](test/scenarios/01_cert_generation/run.sh)
     * Certificate creation and validation
     * Property verification
     * Installation procedures
   - [Basic Connectivity](test/scenarios/02_basic_connectivity/run.sh)
     * Tunnel establishment
     * Bidirectional communication
     * Process health monitoring
   - [Performance Testing](test/scenarios/03_performance/run.sh)
     * Throughput measurement
     * Latency testing
     * Resource monitoring
   - [Security Testing](test/scenarios/04_security/run.sh)
     * Certificate validation
     * TLS version enforcement
     * Access control
     * Process isolation

4. [Test Utilities](test/lib/)
   - [Common Functions](test/lib/common.sh)
     * Logging utilities
     * Process management
     * Network utilities
     * Certificate operations
   - [Process Management](test/lib/process_utils.sh)
     * Process tracking
     * Resource monitoring
     * Cleanup procedures

5. [Test Configurations](test/configs/)
   - [Server Configuration](test/configs/server.yaml)
     * Network settings
     * Security parameters
     * Monitoring options
   - [Client Configuration](test/configs/client.yaml)
     * Connection parameters
     * Security settings
     * Performance options

### Environment Verification
1. [Verification System](tools/verification/README.md)
   - System Overview
   - Module Documentation
   - Configuration Guide
   - Usage Instructions
   - Deployment Process
   - QA Testing Process
   - Troubleshooting

2. [Verification Modules](tools/verification/modules/)
   - [System Verification](tools/verification/modules/system/verify.sh)
     * OpenSSL Configuration
     * TUN Module Support
     * System Resources
     * File Descriptor Limits
   - [Network Verification](tools/verification/modules/network/verify.sh)
     * IP Forwarding
     * Interface Configuration
     * Port Availability
     * Network Connectivity
   - [Security Verification](tools/verification/modules/security/verify.sh)
     * Certificate Validation
     * Memory Protections
     * Namespace Support
     * Capability Verification
   - [Performance Verification](tools/verification/modules/performance/verify.sh)
     * System Performance
     * Network Performance
     * Resource Limits
     * Monitoring System

3. [Verification Tools](tools/verification/)
   - [Unified Verifier](tools/verification/unified_verifier.sh)
     * Main verification script
     * Module orchestration
     * Report generation
   - [Deployment Script](tools/verification/deploy.sh)
     * QA environment deployment
     * Remote installation
     * Initial verification
   - [Common Utilities](tools/verification/lib/common.sh)
     * Logging functions
     * Environment detection
     * Result tracking
     * State management
   - [Cleanup Script](tools/verification/cleanup_qa.sh)
     * QA environment cleanup
     * Process termination
     * Interface removal
     * Certificate cleanup
   - [SSSonector Deployment](tools/verification/deploy_sssonector.sh)
     * Certificate generation
     * Configuration creation
     * Binary deployment
     * Permission setup
   - [Sanity Checks](tools/verification/run_sanity_checks.sh)
     * Server/client testing
     * Tunnel verification
     * Packet transmission testing
     * Tunnel closure verification
   - [Automated QA Testing](tools/verification/run_qa_tests.sh)
     * End-to-end testing
     * Environment preparation
     * Test report generation
     * Comprehensive validation

4. [Environment Configurations](tools/verification/config/)
   - [Environment Settings](tools/verification/config/environments.yaml)
     * Common requirements
     * Environment-specific thresholds
     * Resource limits
     * Monitoring configuration

5. [Deployment Documentation](tools/verification/)
   - [Deployment Guide](tools/verification/DEPLOYMENT_GUIDE.md)
     * Repeatable deployment instructions
     * Usage guide
     * Troubleshooting
     * CI/CD integration
   - [QA Deployment Report](tools/verification/qa_deployment_report.md)
     * Deployment details
     * Verification results
     * QA environment status
   - [Legacy Deprecation](tools/verification/LEGACY_DEPRECATION.md)
     * Legacy scripts identification
     * Deprecation process
     * Migration guidance
   - [QA Testing Guide](tools/verification/QA_TESTING_GUIDE.md)
     * Comprehensive testing process
     * Order of operations
     * Debugging procedures
     * Best practices
   - [QA Testing Overview](tools/verification/QA_TESTING.md)
     * Basic testing guide
     * Environment setup
     * Tool usage
   - [Investigation Guide](tools/verification/INVESTIGATION_GUIDE.md)
     * Systematic troubleshooting approach
     * Firewall rules investigation
     * Routing tables verification
     * Kernel parameters investigation
     * Packet capture analysis
     * MTU investigation
     * Packet filtering investigation
   - [Last Mile Connectivity Guide](tools/verification/LAST_MILE_CONNECTIVITY.md)
     * Last mile connectivity issues
     * Root causes and solutions
     * Firewall rules fixes
     * Kernel parameters adjustments
     * MTU settings
     * Routing configuration
     * Permanent fixes
   - [Connectivity Investigation Report](tools/verification/CONNECTIVITY_INVESTIGATION_REPORT.md)
     * Investigation process
     * Key findings
     * Root cause analysis
     * Tools created
     * Recommendations
     * Next steps

6. [Legacy Scripts (DEPRECATED)](sentinel/tools/qa_setup/)
   - [Verify QA Environment](sentinel/tools/qa_setup/verify_qa_environment.sh) **(DEPRECATED)**
     * *Use [Unified Verifier](tools/verification/unified_verifier.sh) instead*
   - [Setup QA Environment](sentinel/tools/qa_setup/setup_qa_environment.sh) **(DEPRECATED)**
     * *Use [Deployment Script](tools/verification/deploy.sh) instead*
   - [QA Certification](sentinel/tools/qa_setup/qa_certification.sh) **(DEPRECATED)**
     * *Use verification reports generated by the new system instead*
   - [Environment Verification](tools/verify_environment.sh) **(DEPRECATED)**
     * *Use [Unified Verifier](tools/verification/unified_verifier.sh) instead*

## Features

[Previous content...]

### Testing Framework
- Modular test scenarios
- Process management utilities
- Resource monitoring
- Security validation
- Performance measurement
- Certificate handling
- Automated cleanup
- Environment verification
- Cross-environment validation
- Automated deployment

[Rest of the original content...]

## Testing
- Scenario-based testing
- Certificate validation
- Connectivity verification
- Performance measurement
- Security assessment
- Resource monitoring
- Process isolation testing
- Environment verification
- System requirements validation
- Network configuration testing
- Automated QA testing
- Environment cleanup
- SSSonector deployment
- Sanity checks
- Tunnel establishment testing
- Packet transmission validation
- IP forwarding verification
- TUN interface validation
- Comprehensive test reporting
- End-to-end test automation

## Best Practices
[Previous content plus:]
- Test scenario organization
- Process management
- Resource monitoring
- Certificate handling
- Security validation
- Performance testing
- Environment verification
- Cross-environment deployment
- Regular system validation
- Configuration management
- Scheduled environment verification
- Deployment automation
- Legacy script deprecation
- Standardized verification methodology
- Clean QA environment before testing
- Structured QA testing process
- Comprehensive test documentation
- Automated end-to-end testing
- Detailed debugging procedures
- Repeatable testing workflows
- IP forwarding verification
- Tunnel establishment validation
- Packet transmission testing
- Systematic troubleshooting approach
- Firewall rules investigation
- Routing tables verification
- Kernel parameters investigation
- Packet capture analysis
- MTU investigation
- Packet filtering investigation
- Safe channel closing to prevent race conditions
- Proper EOF handling in mock connections for tests
- Client-server communication symmetry verification
- Comprehensive bug fix documentation

## Release Information
Current Version: v2.0.0-91-g7a97894-dirty
- Enhanced startup logging
- Version information in logs
- Cross-platform builds
- Performance improvements
- Extended documentation
- Revised test framework
- Improved security testing
- Environment verification system
- QA deployment automation
- Configuration validation
- Comprehensive deployment guide
- Legacy script deprecation documentation
- Systematic troubleshooting tools
- Tunnel connectivity investigation
- Last mile connectivity fixes
- Connectivity investigation report
- Race condition fixes in Transfer Start/Stop methods
- Improved EOF handling in mock connections for tests
- Client-server communication asymmetry documentation
