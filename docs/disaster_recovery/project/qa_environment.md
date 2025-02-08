# SSSonector QA Environment Documentation

## Virtual Machine Configuration

### 1. Server VM (192.168.50.210)
- **Hostname**: sssonector-qa-server
- **Status**: Operational
- **Services**:
  * SSH: Operational
  * Sudo: Operational
- **Purpose**: Primary test server for SSSonector
- **Current Build**: Latest from main branch
- **Test Files**: Located in /home/test/sssonector/

### 2. Client VM (192.168.50.211)
- **Hostname**: sssonector-qa-client
- **Status**: Operational
- **Services**:
  * SSH: Operational
  * Sudo: Operational
  * SNMP Client: Operational
- **Purpose**: Test client for tunnel connections
- **Test Data**: DryFire_v4_10.zip (3.3GB)
- **Location**: /home/test/data/

### 3. SNMP Server VM (192.168.50.212)
- **Hostname**: sssonector-qa-monitor
- **Status**: Operational
- **Services**:
  * SSH: Operational
  * Sudo: Operational
  * SNMPD: Active (Port 161)
  * ntopng: Active (Port 3000)
- **Configuration**:
  * SNMP Community: public
  * Interface Stats: Available
  * System Info: Available
  * Enterprise MIB: Not Configured
  * Network: 192.168.50.0/24
  * Interface: enp0s3

## Active Test Scripts

### 1. Rate Limiting Tests
- `test_rate_limit_server_to_client.exp`
- `test_rate_limit_client_to_server.exp`
- Status: In Progress
- Test Points: 5, 25, 50, 75, 100 Mbps
- Results: Pending completion

### 2. SNMP Monitoring Tests
- `test_snmp_comprehensive.exp`
- `monitor_snmp_metrics.exp`
- `dynamic_snmp_metrics.exp`
- `verify_snmp_query.exp`
- `verify_snmp_remote.exp`
- Status: Basic tests passed
- Enterprise MIB: Implementation pending

### 3. Environment Management
- `cleanup_test_environment.sh`
- `deploy_test_environment.sh`
- `setup_snmp_monitoring.sh`
- Purpose: Environment maintenance
- Status: Operational

### 4. Automation Scripts
- `verify_vm_access.exp`
- `check_qa_env.exp`
- `verify_snmp.exp`
- Purpose: Environment validation
- Status: Active and maintained

## Test Data

### 1. Rate Limiting Test File
- Name: DryFire_v4_10.zip
- Size: 3.3GB
- Location: /home/test/data/
- Purpose: Throughput testing
- Status: In use for certification

### 2. SNMP Test Data
- Metrics Range File: /etc/snmp/metrics_range.conf
- Sample Data: /var/lib/snmp/test_data/
- Purpose: Monitoring validation
- Status: Active

### 3. Certificate Test Data
- Location: /home/test/certs/
- Types:
  * Test certificates
  * Temporary certificates
  * Invalid certificates
- Purpose: Certificate validation testing
- Status: Complete and verified

## Environment Variables

### 1. System Configuration
```bash
GOPATH=/home/test/go
PATH=/usr/local/go/bin:$PATH
SSSONECTOR_HOME=/home/test/sssonector
```

### 2. Test Configuration
```bash
TEST_SERVER=192.168.50.210
TEST_CLIENT=192.168.50.211
SNMP_SERVER=192.168.50.212
TEST_FILE_PATH=/home/test/data/DryFire_v4_10.zip
```

### 3. SNMP Configuration
```bash
SNMP_COMMUNITY=public
SNMP_PORT=161
SNMP_VERSION=2c
ENTERPRISE_OID=.1.3.6.1.4.1.54321
```

## Required Cleanup

### 1. Before Tests
- Remove old test certificates
- Clear SNMP statistics
- Reset rate limiting counters
- Clean log files

### 2. After Tests
- Remove temporary files
- Archive test results
- Reset VM states
- Clear test data

## Test Environment Dependencies
- Go 1.21+
- Ubuntu 24.04
- VirtualBox 7.0+
- iproute2
- snmpd
- ntopng
- expect (for automation)

## Network Configuration
- Subnet: 192.168.50.0/24
- VirtualBox Host-Only Network
- All VMs connected
- Internet access via NAT
- Firewall rules configured for testing

## Monitoring Setup
- SNMP polling every 5 seconds
- Metrics logged to CSV
- ntopng dashboard available
- Real-time monitoring active
- Alert thresholds configured
