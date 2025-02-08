# SSSonector Test Suite Documentation

## Overview

This document details the structure and organization of the SSSonector test suite, focusing on SNMP monitoring and rate limiting validation.

## Test Suite Organization

### 1. Test Script Hierarchy

```
scripts/
├── test_utils.exp              # Common test utilities
├── basic/
│   ├── test_snmp_basic.exp    # Basic SNMP functionality
│   └── verify_snmp_query.exp  # SNMP query validation
├── rate_limiting/
│   ├── test_snmp_rate_limiting.exp    # Static rate limits
│   └── test_snmp_dynamic_rates.exp    # Dynamic rate adjustments
└── integration/
    ├── test_snmp_comprehensive.exp    # End-to-end testing
    └── verify_web_monitor.exp         # Web monitor validation
```

### 2. Common Test Utilities

```tcl
# test_utils.exp
proc log_test_result {test_name result message} {
    send_log "\n=== $test_name ===\n"
    send_log "Result: [expr {$result ? "PASS" : "FAIL"}]\n"
    send_log "Details: $message\n"
    return $result
}

proc setup_test_environment {} {
    global monitor_host server_host client_host
    set timeout 30
    log_file -a "test_results.log"
}

proc cleanup_test_environment {} {
    # Environment cleanup procedures
}
```

### 3. Test Categories

#### Basic SNMP Tests
```tcl
# test_snmp_basic.exp
proc test_snmp_connectivity {host} {
    send_log "\nTesting SNMP connectivity to $host...\n"
    
    spawn snmpwalk -v2c -c public $host system
    expect {
        "system.sysDescr.0" {
            log_test_result "SNMP Connectivity" 1 "Successfully connected to $host"
            return 1
        }
        timeout {
            log_test_result "SNMP Connectivity" 0 "Connection timeout to $host"
            return 0
        }
    }
}

proc verify_metric {host metric_name oid expected_pattern} {
    spawn snmpget -v2c -c public $host $oid
    expect {
        -re $expected_pattern {
            return 1
        }
        timeout {
            return 0
        }
    }
}
```

#### Rate Limiting Tests
```tcl
# test_snmp_rate_limiting.exp
set rate_limits {
    "5mbps"   5242880
    "10mbps" 10485760
    "25mbps" 26214400
}

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

#### Integration Tests
```tcl
# test_snmp_comprehensive.exp
array set test_suites {
    "basic" {
        "description" "Basic SNMP Functionality"
        "tests" {
            "connectivity" test_snmp_connectivity
            "metrics" test_basic_metrics
        }
        "required" 1
    }
    "rate_limiting" {
        "description" "Rate Limiting Features"
        "tests" {
            "static_limits" test_static_rate_limits
            "dynamic_limits" test_dynamic_rate_limits
        }
        "required" 1
    }
}
```

### 4. Test Execution Flow

#### Basic Test Flow
1. Environment Setup
   ```tcl
   if {![setup_test_environment]} {
       error "Failed to setup test environment"
   }
   ```

2. Test Execution
   ```tcl
   dict set test_results "connectivity" [test_snmp_connectivity $monitor_host]
   dict set test_results "metrics" [verify_metrics $monitor_host]
   ```

3. Result Collection
   ```tcl
   foreach {test result} [array get test_results] {
       if {$result} {
           send_log "✓ $test: PASS\n"
       } else {
           send_log "✗ $test: FAIL\n"
       }
   }
   ```

### 5. Test Data Management

#### Test File Generation
```bash
# generate_test_data.sh
SIZES=(
    "1MB"
    "10MB"
    "100MB"
    "1GB"
)

for size in "${SIZES[@]}"; do
    filename="test_data_${size}.dat"
    case $size in
        "1MB")   count=1024 ;;
        "10MB")  count=10240 ;;
        "100MB") count=102400 ;;
        "1GB")   count=1048576 ;;
    esac
    
    dd if=/dev/urandom of=$filename bs=1024 count=$count
    sha256sum $filename > "${filename}.sha256"
done
```

### 6. Test Result Reporting

#### HTML Report Generation
```tcl
proc generate_test_report {results} {
    set report_file [open "test_report.html" w]
    
    puts $report_file {
        <!DOCTYPE html>
        <html>
        <head>
            <title>SSSonector Test Results</title>
            <style>
                .pass { color: green; }
                .fail { color: red; }
            </style>
        </head>
        <body>
    }
    
    puts $report_file "<h2>Test Results Summary</h2>"
    puts $report_file "<table border='1'>"
    
    dict for {test_name result} $results {
        set class [expr {$result ? "pass" : "fail"}]
        puts $report_file "<tr class='$class'>"
        puts $report_file "<td>$test_name</td>"
        puts $report_file "<td>[expr {$result ? "PASS" : "FAIL"}]</td>"
        puts $report_file "</tr>"
    }
    
    puts $report_file "</table></body></html>"
    close $report_file
}
```

### 7. Current Test Coverage

#### Test Suite Status
| Category | Coverage | Status |
|----------|----------|---------|
| Basic Connectivity | 100% | Complete |
| Metric Validation | 75% | In Progress |
| Rate Limiting | 50% | In Progress |
| Integration Tests | 25% | Planned |

#### Priority Test Cases
1. SNMP Extend Script Validation
2. Dynamic Rate Limit Adjustments
3. High Load Performance Testing
4. Error Condition Handling

### 8. Known Issues and Workarounds

#### OID Format Issues
- **Problem**: Mismatch between numeric and named OIDs
- **Workaround**: Using NET-SNMP-EXTEND-MIB format
```tcl
# Old format
set oid ".1.3.6.1.4.1.8072.1.3.2.4.1.2..."

# New format
set oid 'NET-SNMP-EXTEND-MIB::nsExtendOutput1Line."sssonector-throughput"'
```

#### Test Environment Setup
- **Problem**: Inconsistent VM state after tests
- **Workaround**: Added cleanup procedures
```tcl
proc cleanup_test_environment {} {
    global monitor_host
    
    # Reset SNMP configuration
    spawn ssh $monitor_host "sudo systemctl restart snmpd"
    expect "Password:"
    send "password\r"
    
    # Clean test data
    spawn ssh $monitor_host "rm -f /tmp/test_*.dat"
    expect "Password:"
    send "password\r"
}
```

### 9. Future Improvements

1. Automated Test Scheduling
   - Implement Jenkins integration
   - Add periodic test runs
   - Automated result collection

2. Enhanced Reporting
   - Trend analysis
   - Performance metrics
   - Historical comparisons

3. Test Coverage Expansion
   - Error injection testing
   - Network failure scenarios
   - Security validation
