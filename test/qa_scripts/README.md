# SSSonector QA Test System

This directory contains scripts for automated testing of SSSonector in a QA environment. The system handles building from source, deploying to QA systems, and running comprehensive test scenarios.

## Prerequisites

1. Two Ubuntu VMs configured as:
   - QA Server VM (2GB RAM, 20GB storage)
   - QA Client VM (2GB RAM, 20GB storage)
2. SSH access configured to both VMs
3. Go 1.21 or later installed on the development system
4. Required tools on QA systems:
   - systemd
   - ip tools
   - iptables
   - ping

## Test Scripts

### 1. build_and_deploy.sh
Builds SSSonector from source and deploys to QA systems:
- Verifies build environment
- Runs unit tests
- Creates distribution package
- Deploys to QA server and client

### 2. verify_build.sh
Verifies the installation on QA systems:
- Checks binary installation
- Validates configuration
- Verifies service installation
- Checks system dependencies

### 3. test_scenarios.sh
Runs test scenarios:
- Foreground Client/Server
- Background Client, Foreground Server
- Background Client/Server
Each scenario includes:
- Connection establishment
- Packet transmission tests
- Clean shutdown verification

### 4. run_qa_tests.sh
Master script that orchestrates the entire testing process:
- Runs all scripts in sequence
- Collects logs and metrics
- Generates comprehensive test report

## Usage

1. Configure QA system hostnames in scripts:
   ```bash
   # Edit QA_SERVER and QA_CLIENT variables in scripts if needed
   QA_SERVER="qa-server"
   QA_CLIENT="qa-client"
   ```

2. Run the complete test suite:
   ```bash
   ./run_qa_tests.sh
   ```

3. Run individual components:
   ```bash
   # Build and deploy only
   ./build_and_deploy.sh
   
   # Verify installation
   ./verify_build.sh
   
   # Run test scenarios
   ./test_scenarios.sh
   ```

## Test Results

Results are stored in timestamped directories:
```
qa_test_results_YYYYMMDD_HHMMSS/
├── build.log           # Build and deployment logs
├── verify.log         # Installation verification logs
├── scenarios.log      # Test scenario execution logs
├── logs_qa-server/    # Server system logs
├── logs_qa-client/    # Client system logs
└── final_report.md    # Comprehensive test report
```

## Test Scenarios

### Scenario 1: Foreground Client/Server
- Server runs in foreground
- Client runs in foreground
- Tests basic connectivity
- Verifies clean shutdown

### Scenario 2: Background Client, Foreground Server
- Server runs in foreground
- Client runs as service
- Tests service integration
- Verifies service management

### Scenario 3: Background Client/Server
- Both components run as services
- Tests full service deployment
- Verifies system integration

## Troubleshooting

1. SSH Connection Issues:
   - Verify SSH key configuration
   - Check network connectivity
   - Ensure correct hostnames in /etc/hosts

2. Build Failures:
   - Check Go version compatibility
   - Verify all dependencies are installed
   - Review build.log for errors

3. Test Failures:
   - Check system logs in logs_* directories
   - Verify service status on both systems
   - Review final_report.md for error patterns

## Adding New Tests

1. Create new test script in qa_scripts/
2. Update run_qa_tests.sh to include new test
3. Add test results to final report generation
4. Update this README with new test details

## Maintenance

- Review and update Go version requirements
- Keep test scenarios aligned with product features
- Maintain QA system requirements
- Update documentation for new test cases
