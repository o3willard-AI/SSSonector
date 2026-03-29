# SSSonector Architecture

## Overview

SSSonector (**Secure SSL Connector**) is a high-performance communications utility designed to enable secure service-to-service communication over the public internet without VPN requirements. Tunnel traffic is designed to be indistinguishable from normal HTTPS web traffic, enabling traversal of firewalls and network infrastructure that only permits standard web browsing. This document outlines the system architecture and key components.

## Core Components

### 1. Service Layer

The service layer provides the core functionality and is built around a clean interface-based design:

```go
service.Service
├── Start()      // Service lifecycle
├── Stop()       // management
├── Reload()     // operations
├── Status()     // Status reporting
├── Metrics()    // Performance metrics
└── Health()     // Health checking
```

The service implementation follows the composition pattern, with a base service providing common functionality that can be extended by specific service types.

### 2. Control System

The control system enables management and monitoring of the service through a Unix domain socket interface:

```
ControlServer <-> Unix Socket <-> ControlClient
     │                                │
     └── Service                     └── CLI/API
```

Components:
- **ControlServer**: Handles command routing and execution
- **ControlClient**: Provides client-side command interface
- **Unix Socket**: Secure local communication channel
- **Command Protocol**: JSON-based message format

Commands:
- Status reporting
- Metrics collection
- Health checking
- Lifecycle management
- Configuration reloading

### 3. Configuration Management

```
AppConfig
├── Mode          // Operating mode (server/client)
├── Network       // Network configuration
├── Tunnel        // Tunnel settings
├── Facade        // HTTPS facade for firewall traversal
├── Monitor       // Monitoring configuration
├── Security      // Security settings
└── Throttle      // Rate limiting
```

Features:
- JSON/YAML support
- Environment variable overrides
- Hot reload capability
- Validation system

### 4. HTTPS Facade (Firewall Traversal)

The facade enables tunnel connections to traverse firewalls that only allow standard HTTPS traffic (port 443).

```
Facade
├── Server              // HTTPS web server on port 443
│   ├── Web Handler     // Returns legitimate web page for GET /
│   ├── Upgrade Handler // Handles WebSocket upgrade with HMAC token
│   └── Proxy           // Bridges hijacked connection to tunnel port
├── Client              // Two-stage connection with fallback
│   ├── Direct Connect  // Try configured tunnel port first
│   └── Facade Fallback // Fall back to HTTPS upgrade on port 443
├── Token               // HMAC-SHA256 signed authentication tokens
│   ├── Generation      // Port + timestamp + signature
│   ├── Validation      // Signature verification + TTL check
│   └── Secret Derivation // From CA certificate or explicit config
└── TLS                 // Separate TLS config for facade layer
```

Connection flow:
```
Client                                 Facade (:443)              Tunnel (:8443)
  │                                       │                           │
  ├── TLS Handshake ──────────────────►   │                           │
  ├── GET /connect + Upgrade headers ──►  │                           │
  │   + X-Tunnel-Token (HMAC signed)      │                           │
  │                                       ├── Validate token          │
  │                                       ├── Verify port allowed     │
  ◄── 101 Switching Protocols ──────────  │                           │
  │                                       ├── Hijack connection       │
  │                                       ├── Dial 127.0.0.1:8443 ──►│
  │◄ ─ ─ ─ Raw bidirectional tunnel data ─ ─ ─ ─ ─ ─ ─ ─ ─ ─ ─ ─ ─►│
```

Security:
- HMAC-SHA256 tokens prevent unauthorized tunnel establishment
- Tokens expire after configurable TTL (default 30s)
- Unknown paths return 404 (no port enumeration possible)
- Standard WebSocket upgrade headers -- indistinguishable from normal web traffic
- Optional client certificate verification via `VerifyClientCertIfGiven`

See [HTTPS Facade Design](https_facade.md) for full implementation details.

### 5. Network Layer

```
Tunnel
├── Adapter           // Network interface management
├── Protocol          // TCP/UDP handling
├── Encryption        // Data security
└── Compression      // Optional compression
```

Capabilities:
- Interface creation/management
- Protocol multiplexing
- Traffic encryption
- Optional compression

### 6. Security System

```
Security
├── Authentication    // Client/server auth
├── Certificates     // TLS/certificate management
├── Access Control   // Network ACLs
└── Audit Logging    // Security event logging
```

Features:
- Certificate-based authentication
- Automatic certificate rotation
- Network-level access control
- Security event auditing

### 7. Monitoring System

```
Monitor
├── Metrics          // Performance metrics
├── Health Checks    // Service health
├── SNMP            // SNMP integration
└── Prometheus      // Prometheus metrics
```

Metrics:
- Performance statistics
- Resource utilization
- Connection tracking
- Error rates

### 8. Rate Limiting

```
Throttle
├── Rate Limiting    // Traffic control
├── Burst Handling   // Burst allowance
└── Token Bucket    // Rate algorithm
```

Features:
- Per-connection limits
- Burst allowance
- Fair queuing

## Communication Flow

1. Service Initialization
```
Start
  ├── Load Configuration
  ├── Initialize Components
  ├── Start Control Server
  └── Begin Monitoring
```

2. Client Connection (Direct)
```
Connect
  ├── TCP to configured port (e.g. 8443)
  ├── TLS Handshake (mTLS)
  ├── Rate Limit Check
  └── Begin Transfer
```

3. Client Connection (Via HTTPS Facade)
```
Connect
  ├── Try direct port → fails (blocked by firewall)
  ├── TLS to facade port 443
  ├── HTTP WebSocket Upgrade + HMAC Token
  ├── 101 Switching Protocols
  ├── Connection hijacked → proxy to tunnel port
  └── Begin Transfer (TLS already established, no double encryption)
```

4. Data Transfer
```
Transfer
  ├── Encryption
  ├── Compression (optional)
  ├── Rate Limiting
  └── Monitoring
```

## Control Flow

1. Command Execution
```
Command
  ├── Client Request
  ├── Socket Transfer
  ├── Server Processing
  └── Response Return
```

2. Configuration Updates
```
Reload
  ├── Load New Config
  ├── Validate Changes
  ├── Apply Updates
  └── Notify Components
```

3. Health Checks
```
Health
  ├── Component Checks
  ├── Resource Checks
  ├── Connectivity Tests
  └── Status Report
```

## Error Handling

The error handling system uses typed errors with specific error codes:

```go
ServiceError
├── Code        // Error classification
├── Message     // Human-readable description
└── Details     // Additional context
```

Common error scenarios:
- Connection failures
- Configuration errors
- Resource exhaustion
- Security violations

## Platform Support

The architecture supports multiple platforms through abstraction layers:

```
Platform
├── Linux       // Full feature set
├── Windows     // Core functionality
└── macOS       // Development support
```

Platform-specific components:
- Network interface management
- Process management
- Security integration
- Resource monitoring

## Future Considerations

1. Scalability
   - Cluster support
   - Load balancing
   - High availability

2. Monitoring
   - Enhanced metrics
   - Distributed tracing
   - Log aggregation

3. Security
   - Additional auth methods
   - Enhanced auditing
   - Compliance features

4. Performance
   - Protocol optimizations
   - Resource efficiency
   - Caching improvements
