# Release Notes

## v1.2.0 (Unreleased)

### HTTPS Facade for Firewall Traversal
- Added HTTPS facade server that runs on port 443 alongside tunnel instances
- Server presents a legitimate HTTPS web page to browsers and port scanners
- Tunnel connections use standard WebSocket upgrade protocol (RFC 6455) to traverse firewalls and DPI
- HMAC-SHA256 signed tokens with configurable TTL for secure tunnel negotiation
- Client automatically falls back to facade when direct tunnel ports are blocked
- Facade proxies connections to local tunnel ports transparently -- tunnel layer is unaware
- Token secret can be explicitly configured or automatically derived from the shared CA certificate
- No double encryption: when connecting via facade, the tunnel's TLS layer is skipped
- Full configuration support with `facade:` section in server and client YAML configs
- 26 unit tests covering token lifecycle, server handlers, client fallback, and data transfer

### New Configuration Options
- `facade.enabled` -- Enable/disable HTTPS facade (server) or fallback (client)
- `facade.listen_address` / `facade.listen_port` -- Facade server bind address (server, default :443)
- `facade.server_address` / `facade.server_port` -- Facade server to connect to (client, default :443)
- `facade.tunnel_ports` -- List of tunnel ports the facade routes to (server)
- `facade.direct_timeout` -- How long client waits for direct connection before fallback (default 3s)
- `facade.token_secret` -- Shared HMAC secret (empty = derive from CA certificate)
- `facade.token_ttl` -- Token validity duration (default 30s)
- `facade.web_root` -- Custom HTML content for the facade web page
- `facade.tls` -- Optional separate TLS cert/key/ca for the facade

### Architecture
- New `internal/facade` package: `server.go`, `client.go`, `token.go`, `proxy.go`
- Facade integrates into existing tunnel Server/Client via composition
- `FacadeConfig` and `FacadeTLSConfig` added to configuration type system
- Config validator extended with facade-specific validation rules
- Config loader handles facade defaults in version upgrade paths

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
