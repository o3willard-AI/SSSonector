# SSSonector QA Environment State Documentation

## Current Environment Status

### 1. Virtual Machine Configuration

#### Monitor Node (192.168.50.212)
- **Role**: SNMP monitoring and metrics collection
- **Hostname**: sssonector-qa-monitor
- **Specifications**:
  - CPU: 4 cores
  - Memory: 8GB
  - Disk: 40GB
  - OS: Ubuntu 24.04 LTS
  - Network: Bridged (enp0s3)

#### Server Node (192.168.50.210)
- **Role**: Primary tunnel endpoint
- **Hostname**: sssonector-server
- **Specifications**:
  - CPU: 2 cores
  - Memory: 4GB
  - Disk: 20GB
  - OS: Ubuntu 24.04 LTS
  - Network: Bridged (enp0s3)

#### Client Node (192.168.50.211)
- **Role**: Secondary tunnel endpoint
- **Hostname**: sssonector-client
- **Specifications**:
  - CPU: 2 cores
  - Memory: 4GB
  - Disk: 20GB
  - OS: Ubuntu 24.04 LTS
  - Network: Bridged (enp0s3)

### 2. SNMP Configuration Status

#### Current MIB Implementation Status
- Enterprise MIB (.1.3.6.1.4.1.54321) partially implemented
- Missing sssonector.so module (Issue #1)
- NET-SNMP-EXTEND-MIB integration in progress

#### SNMP Extend Scripts
```bash
# Currently configured extends
extend sssonector-throughput /usr/local/bin/sssonector-snmp throughput
extend sssonector-connections /usr/local/bin/sssonector-snmp connections
extend sssonector-latency /usr/local/bin/sssonector-snmp latency
```

#### Known Issues
1. MIB Module Loading
   - Error: `dlopen(/usr/lib/snmp/dlmod/sssonector.so) failed: No such file or directory`
   - Status: Pending module compilation
   - Impact: Custom MIB extensions not loading

2. OID Format Mismatches
   - Previous format: Numeric OIDs (e.g., .1.3.6.1.4.1.8072.1.3.2.4.1.2...)
   - Current format: Named MIB references (e.g., NET-SNMP-EXTEND-MIB::nsExtendOutput1Line)
   - Status: Web monitor updated, test scripts pending update

### 3. Test Infrastructure

#### Test Categories
1. Basic SNMP Tests
   - Connectivity verification
   - Basic metrics validation
   - Community string validation

2. Rate Limiting Tests
   - Static rate limit validation
   - Dynamic rate adjustments
   - Throughput measurement

3. Integration Tests
   - End-to-end functionality
   - Web monitor validation
   - Performance benchmarking

#### Test Data
- Standard test files:
  - 1MB test file (validation)
  - 10MB test file (basic transfer)
  - 100MB test file (rate limiting)
  - 1GB test file (extended testing)

### 4. Monitoring Implementation

#### Web Monitor Status
- Service: Running on port 8080
- Metrics Collection:
  - Throughput (RX/TX)
  - Active Connections
  - Latency
- Update Interval: 5 seconds

#### Alert Thresholds
```yaml
throughput:
  warning: 80%  # of configured limit
  critical: 95%
connections:
  warning: 100
  critical: 150
latency:
  warning: 100ms
  critical: 200ms
```

### 5. Current Development Focus

#### Priority Items
1. MIB Implementation
   - Complete sssonector.so module
   - Update OID formats in test scripts
   - Implement proper MIB registration

2. Test Coverage
   - Basic Connectivity: 100%
   - Metric Validation: 75%
   - Rate Limiting: 50%
   - Integration Tests: 25%

#### Pending Tasks
1. Build and deploy sssonector.so
2. Update test scripts with new OID format
3. Implement remaining integration tests
4. Complete performance benchmarking suite

### 6. Environment Management

#### Validation Procedures
1. Network Connectivity
   - Inter-VM communication
   - SNMP port accessibility
   - Web monitor access

2. Service Status
   - SNMP daemon
   - Web monitor
   - Test data generation

3. Configuration Verification
   - SNMP extend scripts
   - MIB registration
   - Alert thresholds

#### Maintenance Procedures
1. Regular Tasks
   - Log rotation
   - Test data cleanup
   - Performance metrics collection

2. Backup Procedures
   - Configuration files
   - Test results
   - Custom scripts

### 7. Recent Changes

#### Last Update: 2025-02-08
1. Updated web monitor OID format
2. Added new test data generation script
3. Implemented basic rate limiting tests
4. Enhanced monitoring dashboard

#### Planned Updates
1. Complete MIB module implementation
2. Expand test coverage
3. Enhance performance monitoring
4. Implement automated test scheduling

### 8. Troubleshooting Guide

#### Common Issues
1. SNMP Connection Failures
   - Verify daemon status
   - Check firewall rules
   - Validate community strings

2. Metric Collection Issues
   - Verify extend script permissions
   - Check OID accessibility
   - Validate data formats

3. Performance Problems
   - Monitor system resources
   - Check network conditions
   - Verify rate limiting configuration

#### Debug Commands
```bash
# SNMP Validation
snmpwalk -v2c -c public localhost NET-SNMP-EXTEND-MIB::nsExtendOutput1Line
snmpget -v2c -c public localhost 'NET-SNMP-EXTEND-MIB::nsExtendOutput1Line."sssonector-throughput"'

# Service Status
systemctl status snmpd
journalctl -u snmpd -n 100

# Network Verification
netstat -tulpn | grep snmp
tcpdump -i any port 161
