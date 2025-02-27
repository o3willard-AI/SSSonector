# SSSonector Documentation-Functionality Mapping

This document maps each configuration option to its corresponding functionality and test case. This mapping helps ensure that all functionality is properly documented and tested.

## Basic Configuration Options

| Configuration Option | Functionality | Test Case ID | Documentation Status |
|----------------------|--------------|-------------|---------------------|
| `mode` | Specifies whether SSSonector runs in server or client mode | CONF-001, CONF-002 | Fully Documented |
| `listen` | Specifies the address and port on which the server listens for connections | CONF-003, CONF-004 | Partially Documented (missing custom port example) |
| `server` | Specifies the address and port of the server to which the client connects | CONF-005, CONF-006 | Partially Documented (uses placeholder) |
| `interface` | Specifies the name of the TUN interface to create | CONF-007, CONF-008 | Partially Documented (missing custom name example) |
| `address` | Specifies the IP address and subnet mask for the TUN interface | CONF-009, CONF-010 | Fully Documented |

## Security Configuration Options

| Configuration Option | Functionality | Test Case ID | Documentation Status |
|----------------------|--------------|-------------|---------------------|
| `security.tls.enabled` | Specifies whether TLS is enabled for secure communication | CONF-101, CONF-102 | Fully Documented |
| `security.tls.min_version` | Specifies the minimum TLS version to use | CONF-103, CONF-104 | Fully Documented |
| `security.tls.cert_file` | Specifies the path to the certificate file | CONF-105 | Fully Documented |
| `security.tls.key_file` | Specifies the path to the private key file | CONF-106 | Fully Documented |
| `security.tls.ca_file` | Specifies the path to the CA certificate file | CONF-107 | Fully Documented |
| `security.tls.mutual_auth` | Specifies whether mutual TLS authentication is enabled | CONF-108, CONF-109 | Not Documented |
| `security.tls.verify_cert` | Specifies whether certificate verification is enabled | CONF-110, CONF-111 | Not Documented |

## Network Configuration Options

| Configuration Option | Functionality | Test Case ID | Documentation Status |
|----------------------|--------------|-------------|---------------------|
| `network.mtu` | Specifies the Maximum Transmission Unit (MTU) for the TUN interface | CONF-201, CONF-202 | Partially Documented (lacks detailed explanation) |
| `network.forwarding.enabled` | Specifies whether packet forwarding is enabled | CONF-203, CONF-204 | Not Documented |
| `network.forwarding.icmp_enabled` | Specifies whether ICMP packets are forwarded | CONF-205, CONF-206 | Not Documented |
| `network.forwarding.tcp_enabled` | Specifies whether TCP packets are forwarded | CONF-207, CONF-208 | Not Documented |
| `network.forwarding.udp_enabled` | Specifies whether UDP packets are forwarded | CONF-209, CONF-210 | Not Documented |
| `network.forwarding.http_enabled` | Specifies whether HTTP traffic is forwarded | CONF-211, CONF-212 | Not Documented |

## Logging Configuration Options

| Configuration Option | Functionality | Test Case ID | Documentation Status |
|----------------------|--------------|-------------|---------------------|
| `logging.level` | Specifies the logging level | CONF-301, CONF-302, CONF-303, CONF-304, CONF-305 | Fully Documented |
| `logging.file` | Specifies the path to the log file | CONF-306, CONF-307 | Fully Documented |
| `logging.debug_categories` | Specifies which categories of debug logs to enable | CONF-308, CONF-309, CONF-310, CONF-311, CONF-312 | Not Documented |
| `logging.format` | Specifies the format of log messages | CONF-313, CONF-314 | Not Documented |

## Monitoring Configuration Options

| Configuration Option | Functionality | Test Case ID | Documentation Status |
|----------------------|--------------|-------------|---------------------|
| `monitoring.enabled` | Specifies whether monitoring is enabled | CONF-401, CONF-402 | Fully Documented |
| `monitoring.port` | Specifies the port on which the monitoring server listens | CONF-403, CONF-404 | Fully Documented |

## Feature Functionality

| Feature | Functionality | Test Case ID | Documentation Status |
|---------|--------------|-------------|---------------------|
| ICMP Packet Forwarding | Forwards ICMP packets between the TUN interface and the physical network interfaces | FEAT-201 | Not Documented |
| TCP Packet Forwarding | Forwards TCP packets between the TUN interface and the physical network interfaces | FEAT-202 | Not Documented |
| UDP Packet Forwarding | Forwards UDP packets between the TUN interface and the physical network interfaces | FEAT-203 | Not Documented |
| HTTP Traffic Forwarding | Forwards HTTP traffic between the TUN interface and the physical network interfaces | FEAT-204 | Not Documented |
| HTTPS Traffic Forwarding | Forwards HTTPS traffic between the TUN interface and the physical network interfaces | FEAT-205 | Not Documented |
| DNS Traffic Forwarding | Forwards DNS traffic between the TUN interface and the physical network interfaces | FEAT-206 | Not Documented |
| Large File Transfer | Transfers large files between the TUN interface and the physical network interfaces | FEAT-207 | Not Documented |

## Documentation Examples

| Example | Functionality | Test Case ID | Documentation Status |
|---------|--------------|-------------|---------------------|
| Server Configuration Example | Provides a complete example of server configuration | DOC-001 | Partially Documented (missing forwarding options) |
| Client Configuration Example | Provides a complete example of client configuration | DOC-002 | Partially Documented (missing forwarding options, uses placeholder) |
| High-Performance Configuration | Provides an example of configuration for high-performance scenarios | DOC-003 | Fully Documented |
| Debugging Configuration | Provides an example of configuration for debugging issues | DOC-004 | Fully Documented |
| Low-Latency Configuration | Provides an example of configuration for low-latency scenarios | DOC-005 | Fully Documented |
| Environment Variables Example | Provides an example of using environment variables | DOC-006 | Partially Documented (incomplete) |
| Configuration File Locations | Describes where SSSonector searches for configuration files | DOC-007 | Fully Documented |
| Configuration Validation | Describes how SSSonector validates the configuration file | DOC-008 | Fully Documented |

## Error Handling

| Error | Functionality | Test Case ID | Documentation Status |
|-------|--------------|-------------|---------------------|
| Missing Required Fields | Reports an error for missing required fields | ERR-001 | Fully Documented |
| Invalid Values | Reports an error for invalid values | ERR-002 | Fully Documented |
| Incompatible Options | Reports an error for incompatible options | ERR-003 | Fully Documented |
| Non-existent File Paths | Reports an error for non-existent file paths | ERR-004 | Fully Documented |
| Inaccessible File Paths | Reports an error for inaccessible file paths | ERR-005 | Fully Documented |
| Network Failure | Handles network failure gracefully | ERR-006 | Not Documented |
| Server Restart | Client reconnects after server restart | ERR-007 | Not Documented |
| Client Restart | Client reconnects after client restart | ERR-008 | Not Documented |

## Documentation Status Summary

| Status | Count | Percentage |
|--------|-------|------------|
| Fully Documented | 16 | 44.4% |
| Partially Documented | 7 | 19.4% |
| Not Documented | 13 | 36.1% |
| **Total** | **36** | **100%** |

## Recommendations

Based on the documentation-functionality mapping, the following recommendations are made:

1. Add documentation for all undocumented configuration options, especially packet forwarding options.
2. Improve documentation for partially documented configuration options, especially by providing concrete examples.
3. Add documentation for all undocumented features, especially packet forwarding features.
4. Update all examples to include all relevant configuration options.
5. Add documentation for error handling, especially for network failures and reconnection scenarios.

These recommendations are aligned with the Documentation Update Plan and will help ensure that the SSSonector documentation is comprehensive, accurate, and provides clear guidance for users.
