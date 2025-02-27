# SSSonector Minimal Functionality Test

This document describes the minimal functionality test for SSSonector, which verifies the core functionality of the SSSonector communication utility in different deployment scenarios.

## Overview

The minimal functionality test verifies the following capabilities:

1. Standard Linux installation/deployment
2. Three deployment scenarios:
   - Client foreground, Server foreground
   - Client background, Server foreground
   - Client background, Server background
3. Clean tunnel opening
4. Bidirectional packet transmission (20 packets each way)
5. Clean tunnel closure when client exits

## Test Implementation

The test is implemented in the `minimal_functionality_test.sh` script, which:

- Deploys SSSonector to the QA environment
- Creates appropriate configuration files for each scenario
- Starts the server and client in the specified modes
- Verifies tunnel establishment
- Sends 20 packets from client to server
- Sends 20 packets from server to client
- Gracefully terminates the client and verifies clean tunnel closure
- Generates detailed test reports with timing measurements

## Packet Types

The test sends three types of packets to simulate real-world traffic:

1. **HTTP-like packets**: Simulating web traffic with HTTP headers and requests
2. **FTP-like packets**: Simulating file transfer commands
3. **Database-like packets**: Simulating SQL queries and transactions

## Timing Measurements

The test captures detailed timing measurements for:

- Server startup time
- Client startup time
- Tunnel establishment time
- Individual packet transmission times
- Client shutdown time
- Tunnel closure time
- Server shutdown time

## Usage

### Running All Test Scenarios

To run all three test scenarios:

```bash
cd /home/sblanken/Desktop/go/src/github.com/o3willard-AI/SSSonector/tools/verification
./minimal_functionality_test.sh
```

### Running a Specific Test Scenario

To run a specific test scenario:

```bash
./minimal_functionality_test.sh <server_mode> <client_mode>
```

Where:
- `server_mode`: `foreground` or `background`
- `client_mode`: `foreground` or `background`

Example:
```bash
./minimal_functionality_test.sh foreground background
```

## Test Reports

The test generates detailed reports for each scenario in Markdown format:

- `/tmp/sssonector_test_report_foreground_foreground.md`
- `/tmp/sssonector_test_report_foreground_background.md`
- `/tmp/sssonector_test_report_background_background.md`

## Logs

The test generates the following logs:

- Test execution log: `/tmp/sssonector_test.log`
- Timing measurements: `/tmp/sssonector_timing.log`
- Timing results CSV: `/tmp/timing_results.csv`

## Requirements

- The QA environment must be accessible via SSH
- The `sshpass` utility must be installed (the script will attempt to install it if missing)
- The `bc` utility must be installed for timing calculations
- Python 3 must be installed on the QA servers for the HTTP server

## Error Handling

The test includes robust error handling with:

- Retry mechanisms for transient failures
- Detailed error logging
- Graceful cleanup on failure
- Comprehensive test reports

## Extending the Test

To add new test scenarios or packet types:

1. Add new packet generation functions in the script
2. Modify the `send_client_to_server_packets` and `send_server_to_client_packets` functions
3. Add new test scenarios in the `main` function
