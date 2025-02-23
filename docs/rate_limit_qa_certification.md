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

### 1. Token Bucket Rate Limiting
- **Base Rate Tests**:
  * Test Points: 1, 10, 50, 100, 500 MB/s
  * Validation Method:
    - Token accumulation accuracy
    - Rate enforcement precision
    - Burst control effectiveness
  * Script: test_token_bucket_base.exp

- **Burst Handling Tests**:
  * Test Points: 10%, 25%, 50%, 100% of base rate
  * Validation Method:
    - Burst size enforcement
    - Token replenishment timing
    - Concurrent request handling
  * Script: test_token_bucket_burst.exp

### 2. Dynamic Rate Adjustment
- **Rate Increase Tests**:
  * Test Points: 10%, 25%, 50%, 100% increase
  * Validation Method:
    - Adjustment accuracy
    - Cooldown enforcement
    - Maximum rate bounds
  * Script: test_dynamic_rate_increase.exp

- **Rate Decrease Tests**:
  * Test Points: 10%, 25%, 50%, 75% decrease
  * Validation Method:
    - Adjustment accuracy
    - Cooldown enforcement
    - Minimum rate bounds
  * Script: test_dynamic_rate_decrease.exp

- **Cooldown Tests**:
  * Test Points: 100ms, 500ms, 1s, 5s cooldown
  * Validation Method:
    - Cooldown timing accuracy
    - Adjustment blocking
    - State consistency
  * Script: test_rate_cooldown.exp

### 3. TCP Overhead Compensation
- **Overhead Tests**:
  * Test Points: Various packet sizes
  * Validation Method:
    - Actual vs configured rate
    - Header overhead handling
    - Throughput consistency
  * Script: test_tcp_overhead.exp

### 4. SNMP Monitoring Integration
- **Metrics to Monitor**:
  * Base Rates (.1.3.6.1.4.1.54321.1.1-2)
  * Adjusted Rates (.1.3.6.1.4.1.54321.1.3-4)
  * Token Counts (.1.3.6.1.4.1.54321.1.5-6)
  * Adjustment Counts (.1.3.6.1.4.1.54321.1.7)
  * Cooldown Status (.1.3.6.1.4.1.54321.1.8)

## Validation Criteria

### 1. Token Bucket Accuracy
- Token accumulation within ±1% of expected
- Rate limiting precision within ±2%
- Burst control within configured limits
- Thread safety under load

### 2. Dynamic Rate Adjustment
- Adjustment accuracy within ±5%
- Proper cooldown enforcement
- Correct min/max rate bounds
- Thread-safe state changes

### 3. System Performance
- CPU usage below 80%
- Memory usage stable
- No resource exhaustion
- Consistent throughput

### 4. SNMP Metrics
- All metrics properly exposed
- Update frequency: 1 second
- Accurate state reflection
- Proper error handling

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

### 2. Token Bucket Testing
```bash
# Base rate tests
./test_token_bucket_base.exp

# Burst handling tests
./test_token_bucket_burst.exp

# Monitor metrics
./monitor_token_bucket.exp
```

### 3. Dynamic Rate Testing
```bash
# Rate adjustment tests
./test_dynamic_rate_increase.exp
./test_dynamic_rate_decrease.exp

# Cooldown tests
./test_rate_cooldown.exp

# Monitor adjustments
./monitor_rate_changes.exp
```

### 4. Integration Testing
```bash
# TCP overhead tests
./test_tcp_overhead.exp

# SNMP verification
./verify_snmp_metrics.exp

# System monitoring
./monitor_system_resources.exp
```

## Implementation Details

### Token Bucket Algorithm
- Precise token accumulation
- Thread-safe operations
- Accurate timing control
- Burst size management

### Dynamic Rate Adjustment
- Smooth rate transitions
- Cooldown enforcement
- Min/max rate bounds
- Thread-safe updates

### SNMP Integration
- Enterprise MIB (.1.3.6.1.4.1.54321)
- Rate metrics (1.x)
- Token metrics (2.x)
- Adjustment metrics (3.x)

## Known Issues

### Current Limitations
- Rate precision at extreme values
- Cooldown granularity limits
- SNMP metric latency

### Workarounds
- Use recommended rate ranges
- Configure appropriate cooldowns
- Account for metric delays

## Next Steps

### Immediate Actions
1. Complete token bucket certification
2. Validate dynamic adjustment
3. Verify SNMP integration
4. Document test results

### Future Improvements
1. Enhanced precision testing
2. Extended performance validation
3. Automated stress testing
4. Cross-platform verification

## Test Scripts

### Token Bucket Tests
```bash
#!/bin/bash
# test_token_bucket_base.exp
# Tests token bucket base rate accuracy

# Configure test parameters
RATES=(1048576 10485760 52428800 104857600 524288000)
DURATION=300  # 5 minutes per test

for rate in "${RATES[@]}"; do
  # Configure rate
  ./set_rate.exp $rate
  
  # Run transfer test
  ./transfer_test.exp $DURATION
  
  # Collect metrics
  ./collect_metrics.exp
  
  # Validate results
  ./validate_rate.exp $rate
done
```

### Dynamic Rate Tests
```bash
#!/bin/bash
# test_dynamic_rate_increase.exp
# Tests dynamic rate adjustment accuracy

# Configure test parameters
BASE_RATE=1048576
ADJUSTMENTS=(10 25 50 100)
COOLDOWN=1000  # 1 second

for adj in "${ADJUSTMENTS[@]}"; do
  # Configure base rate
  ./set_rate.exp $BASE_RATE
  
  # Trigger adjustment
  ./adjust_rate.exp $adj
  
  # Monitor changes
  ./monitor_adjustment.exp
  
  # Validate results
  ./validate_adjustment.exp $adj
done
