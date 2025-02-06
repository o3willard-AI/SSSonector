# Release Notes

## v1.1.0 (2025-02-06)

### Performance & Reliability Improvements
- Enhanced tunnel data transfer reliability with improved EOF handling
- Optimized buffer management for better performance with large packets
- Added retry mechanism for network operations with exponential backoff
- Improved handling of temporary disconnections
- Enhanced metrics collection accuracy

### Tunnel Improvements
- Added support for handling packets of varying sizes efficiently
- Implemented chunked data transfer to prevent buffer overflow
- Enhanced error recovery and connection stability
- Improved MTU handling and packet fragmentation
- Added better support for high-throughput scenarios

### Monitoring Enhancements
- Added detailed metrics for bidirectional data transfer
- Improved accuracy of error tracking and reporting
- Enhanced SNMP integration for better observability
- Added granular performance metrics collection

### Bug Fixes
- Fixed data loss issues during network interruptions
- Resolved connection stalling under high load
- Fixed metrics reporting accuracy issues
- Improved cleanup of resources on connection termination

### Documentation
- Updated installation guides with new configuration options
- Added troubleshooting section for common connectivity issues
- Enhanced monitoring documentation with new metrics details
- Updated cross-platform compatibility notes

## v1.0.0 (Initial Release)
[Previous release notes...]
