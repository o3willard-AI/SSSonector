# Code Structure Overview

A map of the current source tree. Generated from the actual files present in
`internal/` and the supporting top-level directories — every file listed below
exists in the repository.

## Internal Packages (`internal/`)

### Tunnel (`internal/tunnel/`)
- `client.go`: Client-side tunnel lifecycle and reconnect policy
- `server.go`: Server-side tunnel listener and session handling
- `tunnel_common.go`: Shared tunnel logic and connection setup
- `tls.go`, `tls_config.go`: TLS certificate loading and profile management
- `transfer.go`: Data-plane copying between endpoints
- `counters.go`: Byte/connection counters for metrics
- `backoff.go`: Reconnect backoff scheduling
- `errors.go`: Tunnel error definitions
- `adapter_wrapper.go`: Adapter integration for the tunnel
- Tests: `counters_test.go`, `backoff_test.go`, `loopback_test.go`, `reload_test.go`, `tls_test.go`

### Certificate Management (`internal/cert/`)
- `manager.go`: Certificate lifecycle management (issue, load, rotation)
- `generator.go`: CA + peer certificate generation
- `locator.go`: Certificate discovery and path resolution
- `generator/`: Certificate generation implementation (`generator.go`)
- Tests: `manager_test.go`, `manager_filelog_test.go`

### Network Adapters (`internal/adapter/`)
- `interface.go`: Common adapter interface and types
- `interface_linux.go`: Linux TUN implementation
- `interface_darwin.go`: macOS implementation
- `interface_windows.go`: Windows implementation
- `interface_default.go`: Fallback/no-op implementation
- `types.go`: Adapter type definitions
- Tests: `interface_linux_integration_test.go`

### Monitoring (`internal/monitor/`)
- `monitor.go`: Core monitoring loop
- `metrics.go`: Performance metrics collection
- `snmp.go`: SNMP integration and handlers
- `snmp_wire.go`: SNMP wire encoding/decoding
- `mib.go`: SNMP MIB definitions
- `system_metrics.go`: OS-level metrics collection
- Tests: `monitor_test.go`, `snmp_conformance_test.go`

### Rate Limiting (`internal/throttle/`)
- `limiter.go`: Rate limiting API
- `token_bucket.go`: Token bucket algorithm
- `constants.go`: Shared throttle constants
- Tests: `limiter_test.go`, `token_bucket_internal_test.go`

### Configuration (`internal/config/`)
- `config.go`: Manager factory, loader helpers, package entry points
- `model_types.go`: Configuration data structures (`AppConfig`, section types)
- `ifaces.go`: Core interfaces (`ConfigStore`, `ConfigValidator`, `ConfigManager`)
- `filestore.go`: File-based `ConfigStore` implementation
- `mgr.go`: `ConfigManager` implementation
- `validation.go`: Configuration validation
- `loader.go`: Configuration file loading and upgrade
- Tests: `config_test.go`

### HTTPS Facade (`internal/facade/`)
- `server.go`: Facade server (disguises tunnel traffic)
- `client.go`: Facade client
- `proxy.go`: Bidirectional proxy copier
- `token.go`: Token handling
- Tests: `server_test.go`, `client_test.go`, `token_test.go`

### Provisioning (`internal/provision/`)
- `bundle.go`: `.ssp` bundle sealing/opening and KDF
- `code.go`: Pairing-code generation and normalization
- `redeem.go`: Redemption HTTPS server (`--serve`)
- `csr.go`: CSR generation and signing
- `paths.go`: Platform cert-directory defaults and key hardening
- `tty_unix.go`, `tty_windows.go`: Secret entry with TTY enforcement
- `acl_unix.go`, `acl_windows.go`: Platform ACL application
- Tests: `bundle_test.go`, `redeem_test.go`

## Command Line (`cmd/daemon/` — unified binary)
- `main.go`: Entry point and flag handling
- `provision.go`: `provision` subcommand (create/apply/verify)
- `reload.go`: Live reload of rate/log settings
- `logging.go`: Logging setup
- Tests: `reload_test.go`

## Tests & Scripts (`test/`)
- `run_cert_tests.sh`: Test suite runner
- `test_cert_generation.sh`: Certificate generation tests
- `test_temp_certs.sh`: Temporary certificate testing
- `transfer_certs.sh`: Certificate transfer tests
- `test_results.md`: Test results documentation
- `qa_scripts/`: QA test scripts

## Configuration (`configs/`)
- `server.yaml`: Server configuration template
- `client.yaml`: Client configuration template

## Installers (`installers/`)
- `windows.nsi`: Windows installer script
- `linux/`, `macos/`, `windows/`: Platform installer assets

## Documentation
- `docs/provisioning_guide.md`: Walkthrough for `provision` (create/apply/verify)
- `docs/configuration_guide.md`: Full configuration reference
- `docs/certificate_management.md`: Certificate chain and permissions
- `docs/snmp_monitoring.md`: SNMP and metrics
- `docs/rate_limiting_implementation.md`: Rate limiting details
- See `docs/README.md` for the complete index.