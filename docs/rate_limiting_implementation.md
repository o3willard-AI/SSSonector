# SSSonector Rate Limiting Implementation

## Overview

The rate limiting system in SSSonector uses a token bucket algorithm to control bandwidth usage and is monitored through SNMP. This document details the implementation, configuration, and monitoring of the rate limiting features.

## Architecture

### 1. Token Bucket Implementation

```go
// internal/throttle/token_bucket.go
type TokenBucket struct {
    rate       float64   // tokens per second
    burstSize  float64   // maximum bucket size
    tokens     float64   // current token count
    lastUpdate time.Time // last token update time
    mu         sync.Mutex
}

func (tb *TokenBucket) Allow(tokens float64) bool {
    tb.mu.Lock()
    defer tb.mu.Unlock()

    now := time.Now()
    elapsed := now.Sub(tb.lastUpdate).Seconds()
    tb.tokens = math.Min(tb.burstSize, tb.tokens+elapsed*tb.rate)
    tb.lastUpdate = now

    if tokens <= tb.tokens {
        tb.tokens -= tokens
        return true
    }
    return false
}
```

### 2. Rate Limiting Integration

#### Connection Manager
```go
// internal/connection/manager.go
type RateLimitedConnection struct {
    conn       net.Conn
    upBucket   *TokenBucket
    downBucket *TokenBucket
    metrics    *monitor.Metrics
}

func (rlc *RateLimitedConnection) Read(p []byte) (n int, err error) {
    if !rlc.downBucket.Allow(float64(len(p))) {
        time.Sleep(time.Millisecond * 100)
        return 0, nil
    }
    n, err = rlc.conn.Read(p)
    if n > 0 {
        rlc.metrics.AddBytesIn(int64(n))
    }
    return
}

func (rlc *RateLimitedConnection) Write(p []byte) (n int, err error) {
    if !rlc.upBucket.Allow(float64(len(p))) {
        time.Sleep(time.Millisecond * 100)
        return 0, nil
    }
    n, err = rlc.conn.Write(p)
    if n > 0 {
        rlc.metrics.AddBytesOut(int64(n))
    }
    return
}
```

### 3. SNMP Integration

#### MIB Structure
```
SSSONECTOR-MIB DEFINITIONS ::= BEGIN

-- Rate Limiting OIDs
sssonectorRateLimit OBJECT IDENTIFIER ::= { sssonector 3 }

-- Upload rate limit
uploadRateLimit OBJECT-TYPE
    SYNTAX      Gauge32
    MAX-ACCESS  read-write
    STATUS      current
    DESCRIPTION "Upload rate limit in kbps"
    ::= { sssonectorRateLimit 1 }

-- Download rate limit
downloadRateLimit OBJECT-TYPE
    SYNTAX      Gauge32
    MAX-ACCESS  read-write
    STATUS      current
    DESCRIPTION "Download rate limit in kbps"
    ::= { sssonectorRateLimit 2 }

END
```

#### Rate Limit Configuration
```go
// internal/throttle/limiter.go
type RateLimiter struct {
    uploadLimit   uint32
    downloadLimit uint32
    metrics       *monitor.Metrics
}

func (rl *RateLimiter) SetUploadLimit(kbps uint32) {
    rl.uploadLimit = kbps
    rl.metrics.SetRateLimit("upload", int64(kbps))
}

func (rl *RateLimiter) SetDownloadLimit(kbps uint32) {
    rl.downloadLimit = kbps
    rl.metrics.SetRateLimit("download", int64(kbps))
}
```

## Configuration

### 1. Rate Limit Settings

#### YAML Configuration
```yaml
# configs/server.yaml
rate_limiting:
  upload:
    enabled: true
    rate_kbps: 10240  # 10 Mbps
    burst_factor: 1.5  # Allow 50% burst
  download:
    enabled: true
    rate_kbps: 10240
    burst_factor: 1.5
```

#### Dynamic Configuration via SNMP
```bash
# Set upload rate limit (10 Mbps)
snmpset -v2c -c private localhost \
    'NET-SNMP-EXTEND-MIB::nsExtendOutput1Line."sssonector-rate-up"' \
    i 10240

# Set download rate limit (10 Mbps)
snmpset -v2c -c private localhost \
    'NET-SNMP-EXTEND-MIB::nsExtendOutput1Line."sssonector-rate-down"' \
    i 10240
```

### 2. Monitoring Configuration

#### SNMP Extend Scripts
```bash
# /usr/local/bin/sssonector-rate-limits
#!/bin/bash

case "$1" in
    "upload")
        cat /var/run/sssonector/rate_up
        ;;
    "download")
        cat /var/run/sssonector/rate_down
        ;;
    *)
        echo "Unknown rate limit type: $1"
        exit 1
        ;;
esac
```

## Monitoring

### 1. Available Metrics

#### Rate Limiting Metrics
- Current rate limits
  * Upload limit (kbps)
  * Download limit (kbps)
- Actual throughput
  * Upload rate (kbps)
  * Download rate (kbps)
- Burst statistics
  * Peak upload rate
  * Peak download rate
  * Burst duration

### 2. SNMP Queries

#### Get Current Rate Limits
```bash
# Upload limit
snmpget -v2c -c public localhost \
    'NET-SNMP-EXTEND-MIB::nsExtendOutput1Line."sssonector-rate-up"'

# Download limit
snmpget -v2c -c public localhost \
    'NET-SNMP-EXTEND-MIB::nsExtendOutput1Line."sssonector-rate-down"'
```

#### Monitor Actual Throughput
```bash
# Watch throughput in real-time
watch -n 1 'snmpget -v2c -c public localhost \
    NET-SNMP-EXTEND-MIB::nsExtendOutput1Line."sssonector-throughput"'
```

## Testing

### 1. Rate Limit Validation

#### Basic Rate Test
```tcl
# test_snmp_rate_limiting.exp
proc test_rate_limit {target_rate actual_rate tolerance} {
    set lower_bound [expr {$target_rate * (1.0 - $tolerance)}]
    set upper_bound [expr {$target_rate * (1.0 + $tolerance)}]
    
    if {$actual_rate >= $lower_bound && $actual_rate <= $upper_bound} {
        return [log_test_result "Rate Limit $target_rate" 1 \
            "Rate $actual_rate within bounds"]
    } else {
        return [log_test_result "Rate Limit $target_rate" 0 \
            "Rate $actual_rate outside bounds"]
    }
}
```

#### Dynamic Rate Testing
```tcl
# test_snmp_dynamic_rates.exp
proc test_dynamic_rate_adjustment {initial_rate new_rate} {
    # Set initial rate
    spawn snmpset -v2c -c private localhost \
        'NET-SNMP-EXTEND-MIB::nsExtendOutput1Line."sssonector-rate-up"' \
        i $initial_rate
    expect "Success"
    
    # Transfer data and measure
    set initial_throughput [measure_transfer_rate]
    
    # Change rate during transfer
    spawn snmpset -v2c -c private localhost \
        'NET-SNMP-EXTEND-MIB::nsExtendOutput1Line."sssonector-rate-up"' \
        i $new_rate
    expect "Success"
    
    # Measure new throughput
    set new_throughput [measure_transfer_rate]
    
    # Verify adjustment
    return [test_rate_limit $new_rate $new_throughput 0.05]
}
```

### 2. Performance Testing

#### Stress Testing
```bash
# Generate test load
dd if=/dev/urandom of=test.dat bs=1M count=1024

# Test upload with rate limit
curl -T test.dat http://localhost:8080/upload

# Monitor throughput
watch -n 0.1 'snmpget -v2c -c public localhost \
    NET-SNMP-EXTEND-MIB::nsExtendOutput1Line."sssonector-throughput"'
```

## Troubleshooting

### 1. Common Issues

#### Rate Limiting Not Working
1. Verify token bucket initialization
```bash
# Check rate limit configuration
cat /etc/sssonector/server.yaml

# Verify runtime values
snmpwalk -v2c -c public localhost NET-SNMP-EXTEND-MIB::nsExtendOutput1Line
```

#### Inconsistent Throughput
1. Check system resources
```bash
# Monitor CPU usage
top -b -n 1

# Check network interface
ethtool enp0s3
```

2. Verify token bucket parameters
```bash
# Review burst settings
grep burst_factor /etc/sssonector/server.yaml

# Monitor token availability
tail -f /var/log/sssonector/throttle.log
```

### 2. Debugging Tools

#### Rate Limiting Logs
```bash
# Enable debug logging
sed -i 's/log_level: info/log_level: debug/' /etc/sssonector/server.yaml

# Monitor rate limiting events
tail -f /var/log/sssonector/throttle.log | grep "rate_limit"
```

## Future Improvements

1. Enhanced Rate Limiting
   - Per-connection limits
   - Time-based rate policies
   - QoS integration

2. Monitoring Enhancements
   - Historical rate tracking
   - Bandwidth usage alerts
   - Rate limit violation logging

3. Management Features
   - Rate limit schedules
   - Bandwidth quotas
   - Group-based policies
