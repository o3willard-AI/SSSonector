# SSSonector Test Suite

This directory contains the test suite for SSSonector, designed to validate functionality, performance, and security.

## Test Structure

```
test/
├── configs/              # Test configurations
│   ├── certs/           # Generated certificates
│   ├── server.yaml      # Server configuration
│   └── client.yaml      # Client configuration
├── lib/                 # Common test utilities
│   ├── common.sh        # Shared functions
│   └── process_utils.sh # Process management utilities
├── logs/                # Test logs
├── scenarios/           # Test scenarios
│   ├── 01_cert_generation/     # Certificate generation tests
│   ├── 02_basic_connectivity/  # Basic connectivity tests
│   ├── 03_performance/        # Performance tests
│   └── 04_security/          # Security tests
└── run_tests.sh        # Main test runner

## Usage

Run the test suite:

```bash
./run_tests.sh -s <server_ip> -c <client_ip>
```

Options:
- `-s`: Server IP address
- `-c`: Client IP address
- `-h`: Show help message

## Test Scenarios

1. Certificate Generation (01_cert_generation)
   - Generates and validates certificates
   - Verifies certificate properties
   - Tests certificate installation

2. Basic Connectivity (02_basic_connectivity)
   - Tests tunnel establishment
   - Verifies bidirectional communication
   - Checks process health

3. Performance (03_performance)
   - Measures throughput
   - Tests latency
   - Monitors resource usage

4. Security (04_security)
   - Tests certificate validation
   - Verifies TLS version enforcement
   - Checks unauthorized access prevention
   - Tests process isolation

## Results

Test results are stored in `test/results/YYYYMMDD_HHMMSS/`:
- Individual test logs
- Environment information
- Summary report (summary.md)

## Requirements

- Linux system with TUN module support
- OpenSSL for certificate operations
- iperf3 (optional, for performance testing)
- Root/sudo access for network operations

## Configuration

Test configurations in `configs/`:
- Server and client YAML files
- Certificate paths
- Network settings
- Security parameters

## Adding New Tests

1. Create new directory in `scenarios/`
2. Implement `run.sh` following existing pattern
3. Add scenario to main test runner
4. Update documentation

## Troubleshooting

Common issues:
1. TUN module not loaded:
   ```bash
   sudo modprobe tun
   ```

2. Certificate generation fails:
   - Check OpenSSL installation
   - Verify write permissions

3. Network connectivity issues:
   - Check IP forwarding
   - Verify firewall rules
   - Ensure correct IP addresses

4. Performance test failures:
   - Check system resources
   - Verify network conditions
   - Consider iperf3 installation

## Contributing

When adding or modifying tests:
1. Follow existing patterns
2. Update documentation
3. Test thoroughly
4. Consider edge cases
