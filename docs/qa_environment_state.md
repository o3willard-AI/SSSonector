# QA Environment State

## Test Environment Configuration

### Network Configuration
- Monitor Node: 192.168.50.212
- Server Node: 192.168.50.210
- Client Node: 192.168.50.211

### Test Resources
- FTP servers installed and operational on both client and server nodes
- Test file: `/home/sblanken/DryFire_v4_10.zip` available on both machines
- Purpose: Rate limiting testing in both directions through the tunnel

### Monitoring System
- Web monitor operational on port 8080
- Direct metrics collection via script
- Current metrics:
  * Throughput: RX/TX in Mbps
  * Active Connections
  * Latency

### Rate Limiting Configuration
- Implemented using tc (traffic control)
- Test rates: 5, 10, 25, 50 Mbps
- Tolerance: ±10%
- Validation through FTP transfer tests

### Test Suite Organization
- Basic SNMP tests
- Rate limiting tests
- Integration tests
- Web monitor validation

## Current Status

### Active Services
- SNMP monitoring
- Web interface
- FTP servers
- Rate limiting controls

### Test Coverage
- Basic connectivity: Complete
- Metric validation: Complete
- Rate limiting: In progress
- Integration tests: Pending

### Known Issues
- None currently reported

## Maintenance Procedures

### Daily Checks
- Verify FTP server status
- Monitor system metrics
- Review error logs

### Weekly Tasks
- Run full test suite
- Verify rate limiting
- Update test data if needed

## Contact Information
- Technical Support: admin@example.com
- Emergency Contact: ops@example.com
