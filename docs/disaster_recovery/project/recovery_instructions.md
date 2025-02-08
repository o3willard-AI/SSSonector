# SSSonector Recovery Instructions

## Project Overview
SSSonector is a secure SSL tunnel implementation designed for high-performance, reliable data transfer with strong security guarantees. The project focuses on providing a robust, cross-platform solution for encrypted network tunneling.

### Key Features
- TUN interface-based networking for kernel-level performance
- Certificate-based authentication for strong security
- Rate limiting and comprehensive monitoring
- Cross-platform support (Linux/macOS/Windows)

## Critical Paths and Files

### 1. Source Code
```
/home/sblanken/Desktop/go/src/github.com/o3willard-AI/SSSonector/
├── cmd/tunnel/                    # Entry points
│   ├── main.go                   # Main entry and flag handling
│   ├── server.go                 # Server implementation
│   └── client.go                 # Client implementation
├── internal/
│   ├── adapter/                  # Platform-specific adapters
│   ├── cert/                     # Certificate management
│   ├── monitor/                  # Monitoring system
│   ├── throttle/                 # Rate limiting
│   ├── tunnel/                   # Core tunnel logic
│   └── connection/               # Connection management
└── test/                         # Test suite
```

### 2. Documentation
```
/docs/
├── project_context.md            # Project overview
├── code_structure_snapshot.md    # Code organization
├── ai_context_restoration.md     # Development history
├── test_results.md              # Test status
├── certificate_management.md     # Certificate docs
└── installation.md              # Setup guide
```

### 3. QA Environment
```
VMs:
- Server (192.168.50.210)
- Client (192.168.50.211)
- SNMP Monitor (192.168.50.212)

Test Data:
- /home/test/data/DryFire_v4_10.zip
- /home/test/certs/
- /etc/snmp/metrics_range.conf
```

## Development State

### 1. Active Development
- SNMP Monitoring:
  * Community string validation fix
  * Enterprise MIB implementation
  * ntopng integration
- Rate Limiting:
  * Certification testing (5-100 Mbps)
  * Performance optimization
- Cross-platform Support:
  * Linux implementation stable
  * macOS/Windows in progress

### 2. Completed Features
- Certificate Management System
- TUN Interface Enhancement
- Process Management
- Core Tunnel Implementation

### 3. Pending Work
- Enterprise MIB (.1.3.6.1.4.1.54321)
- Rate Limiting Certification
- Performance Optimization
- Cross-platform Testing

## Recovery Steps

### 1. Environment Setup
```bash
# Required Software
- Go 1.21+
- Ubuntu 24.04
- VirtualBox 7.0+
- iproute2
- snmpd
- ntopng

# Environment Variables
export GOPATH=/home/test/go
export PATH=/usr/local/go/bin:$PATH
export SSSONECTOR_HOME=/home/test/sssonector
```

### 2. Source Code Recovery
```bash
cd /home/sblanken/Desktop/go/src/github.com/o3willard-AI/SSSonector
go mod download
go mod tidy
make build
```

### 3. QA Environment Recovery
```bash
# Deploy test environment
./deploy_test_environment.sh

# Verify environment
./verify_vm_access.exp
./check_qa_env.exp
./verify_snmp.exp
```

### 4. Test Data Recovery
```bash
# Certificate test data
cp -r /home/test/certs/ /home/test/sssonector/test/

# Rate limiting test file
cp DryFire_v4_10.zip /home/test/data/

# SNMP configuration
cp metrics_range.conf /etc/snmp/
```

## Validation Steps

### 1. Build Verification
```bash
make build
make test
```

### 2. Basic Functionality
- Certificate generation
- TUN interface creation
- Basic tunnel operation
- Process management

### 3. Advanced Features
- SNMP monitoring
- Rate limiting
- Cross-platform support
- Performance metrics

## Current Issues

### 1. Known Issues
- SNMP community string validation needs improvement
- Enterprise MIB not yet implemented
- Rate limiting certification incomplete
- Performance optimization pending

### 2. Workarounds
- Use basic SNMP monitoring until MIB implementation
- Manual rate limiting verification
- Platform-specific adaptations

## Next Steps

### 1. Immediate Actions
1. Complete SNMP community string validation
2. Finish rate limiting certification
3. Begin enterprise MIB implementation

### 2. Short-term Goals
1. Complete enterprise MIB
2. Integrate ntopng metrics
3. Optimize performance

### 3. Long-term Plans
1. Enhanced monitoring
2. Cross-platform improvements
3. Security hardening

## Contact Information

### Development Team
- SNMP Team: [Contact Info]
- Performance Team: [Contact Info]
- Infrastructure Team: [Contact Info]

### Support Resources
- Documentation: /docs/
- Test Results: /test/test_results.md
- Build Scripts: /scripts/
- QA Environment: VirtualBox VMs

## Recovery Verification

### 1. Basic Tests
```bash
# Run certificate tests
./test/run_cert_tests.sh

# Test temporary certificates
./test/test_temp_certs.sh

# Verify SNMP
./test_snmp_comprehensive.exp
```

### 2. Advanced Tests
```bash
# Rate limiting tests
./test_rate_limit_server_to_client.exp
./test_rate_limit_client_to_server.exp

# Monitor metrics
./monitor_snmp_metrics.exp
```

### 3. System Tests
```bash
# Environment checks
./check_qa_env.exp
./verify_snmp_query.exp
./verify_snmp_remote.exp
```

## Disaster Prevention

### 1. Regular Backups
- Source code repository
- Test environment snapshots
- Configuration files
- Test data

### 2. Documentation Updates
- Keep development threads current
- Update test results
- Maintain environment docs
- Track known issues

### 3. Environment Maintenance
- Regular VM snapshots
- Configuration backups
- Test data preservation
- Script updates
