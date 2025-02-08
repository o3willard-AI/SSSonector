# Rate Limiting QA Certification

## Overview
This document outlines the certification process for SSSonector's rate limiting functionality, including test procedures, validation criteria, and monitoring requirements.

## Test Configuration

### Environment Setup
- Server VM (192.168.50.210)
  * Role: Primary tunnel endpoint
  * SNMP port: 10161
  * Configuration: /etc/sssonector/server.yaml

- Client VM (192.168.50.211)
  * Role: Secondary tunnel endpoint
  * SNMP port: 10162
  * Configuration: /etc/sssonector/client.yaml

- Monitor VM (192.168.50.212)
  * Role: SNMP monitoring and metrics collection
  * Test data generation
  * Performance validation

### Test Data
- File: DryFire_v4_10.zip (3.3GB)
- Location: /home/test/data/
- Purpose: Throughput testing and rate limit validation

## Test Cases

### 1. Server-to-Client Rate Limiting
- **Test Points**: 5, 25, 50, 75, 100 Mbps
- **Validation Method**: 
  * Direct file transfer timing
  * SNMP metrics validation
  * Bandwidth utilization monitoring
- **Script**: test_rate_limit_server_to_client.exp

### 2. Client-to-Server Rate Limiting
- **Test Points**: 5, 25, 50, 75, 100 Mbps
- **Validation Method**:
  * Direct file transfer timing
  * SNMP metrics validation
  * Bandwidth utilization monitoring
- **Script**: test_rate_limit_client_to_server.exp

### 3. SNMP Monitoring Integration
- **Metrics to Monitor**:
  * Bytes In/Out (.1.3.6.1.4.1.54321.1.1-2)
  * Active Connections (.1.3.6.1.4.1.54321.1.7)
  * CPU Usage (.1.3.6.1.4.1.54321.1.8)
  * Memory Usage (.1.3.6.1.4.1.54321.1.9)
  * Rate Limits (.1.3.6.1.4.1.54321.3.2-3)

## Validation Criteria

### 1. Rate Accuracy
- Actual transfer rate within ±5% of configured limit
- Consistent rate over transfer duration
- No sudden spikes or drops in throughput

### 2. System Impact
- CPU usage remains below 80%
- Memory usage remains stable
- No resource exhaustion

### 3. SNMP Metrics
- All metrics properly exposed via SNMP
- Metrics update frequency: 1 second
- Accurate reflection of system state

## Test Procedure

### 1. Pre-test Setup
```bash
# Deploy test environment
./deploy_test_environment.sh

# Verify environment
./verify_vm_access.exp
./check_qa_env.exp
./verify_snmp.exp
```

### 2. Rate Limit Testing
```bash
# Server to Client tests
./test_rate_limit_server_to_client.exp

# Client to Server tests
./test_rate_limit_client_to_server.exp

# Monitor metrics
./monitor_snmp_metrics.exp
```

### 3. Validation
```bash
# Verify SNMP metrics
./verify_snmp_query.exp
./verify_snmp_remote.exp

# Check system status
./check_qa_env.exp
```

## Implementation Details

### Rate Limiting
- Token bucket algorithm implementation
- Configurable upload/download limits
- Burst handling: 1 second worth of tokens
- Thread-safe implementation

### SNMP Integration
- Enterprise MIB (.1.3.6.1.4.1.54321)
- Performance metrics (1.x)
- Status metrics (2.x)
- Configuration metrics (3.x)

## Known Issues

### Current Limitations
- Community string validation needs improvement
- Enterprise MIB not fully implemented
- Rate limiting certification incomplete

### Workarounds
- Use basic SNMP monitoring until MIB implementation
- Manual rate limiting verification
- Platform-specific adaptations

## Next Steps

### Immediate Actions
1. Complete rate limiting certification tests
2. Document test results
3. Implement enterprise MIB
4. Integrate ntopng metrics

### Future Improvements
1. Automated performance benchmarking
2. Enhanced monitoring capabilities
3. Cross-platform testing
4. Long-term stability validation
