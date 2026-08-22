# HTTPS Facade: Firewall Traversal via HTTP Upgrade

## Overview

SSSonector means "Secure SSL Connector" -- tunnel traffic should be indistinguishable
from normal HTTPS web traffic to firewalls and network infrastructure. Many enterprise
networks only permit outbound traffic on port 443 (HTTPS), blocking custom ports like
8443, 8444, etc.

The HTTPS Facade provides a mechanism for tunnel connections to operate over port 443
using standard HTTP/1.1 protocol upgrade, making traffic appear identical to normal
HTTPS/WebSocket connections to firewalls and deep packet inspection (DPI) systems.

## Design Principles

1. **Stealth**: Traffic must look like standard HTTPS to DPI and firewalls
2. **Standards-based**: Use HTTP/1.1 `101 Switching Protocols` (same mechanism as WebSocket)
3. **Zero enumeration**: No path-based port discovery; unauthorized probes see a normal website
4. **Secure negotiation**: HMAC-signed tokens prevent unauthorized tunnel establishment
5. **Backward compatible**: Direct-port connections still work; facade is a fallback
6. **Minimal overhead**: After the HTTP upgrade, the connection is raw TCP -- zero framing overhead

## Architecture

```
                        FIREWALL (port 443 open, 8443+ blocked)
                            |
  CLIENT                    |                    SERVER
  ┌──────────────┐          |          ┌──────────────────────────┐
  │ Try direct   │──── 8443 ────X      │                          │
  │ connect first│          |          │  ┌────────────────────┐  │
  │              │          |          │  │ HTTPS Facade (:443)│  │
  │ Fallback to  │──── 443 ──────────►│  │                    │  │
  │ HTTPS Facade │          |          │  │ GET / → "web page" │  │
  │              │          |          │  │ Upgrade → hijack   │  │
  │ HTTP Upgrade │◄─── 101 ──────────│  │ → route to tunnel  │  │
  │              │          |          │  └────────┬───────────┘  │
  │ Raw tunnel   │◄─── TCP ──────────►│           │               │
  │ data flows   │          |          │  ┌───────▼────────────┐  │
  └──────────────┘          |          │  │ Tunnel Instance    │  │
                            |          │  │ (tun0, 10.0.0.1)   │  │
                            |          │  └────────────────────┘  │
                            |          └──────────────────────────┘
```

## Protocol Flow

### Step 1: Client Attempts Direct Connection (Existing Behavior)
```
Client ──TCP SYN──► Server:8443
       ◄─── RST/timeout (port blocked by firewall)
```
The client detects failure within a short timeout (default 3 seconds).

### Step 2: Client Falls Back to HTTPS Facade
```
Client ──TLS Handshake──► Server:443
Client ──HTTP Request──►

GET /connect HTTP/1.1
Host: server.example.com
Upgrade: websocket
Connection: Upgrade
Sec-WebSocket-Key: <base64-random>
Sec-WebSocket-Version: 13
X-Tunnel-Token: <HMAC-signed-token>

Server ◄── Validates token, maps to tunnel instance

◄── HTTP Response ──

HTTP/1.1 101 Switching Protocols
Upgrade: websocket
Connection: Upgrade
Sec-WebSocket-Accept: <computed-accept>

── Connection is now hijacked ──
── Raw bidirectional tunnel data flows ──
```

### Why WebSocket Upgrade Specifically?

Using the actual WebSocket upgrade headers (rather than a custom `Upgrade: sssonector`)
is deliberate:
- Firewalls and proxies are specifically configured to **permit** WebSocket upgrades
- DPI signatures recognize WebSocket as legitimate HTTPS traffic
- Enterprise proxy servers know how to pass through WebSocket connections
- A custom upgrade protocol name would be flagged as suspicious

**After the 101 response, no WebSocket framing is used.** The connection is hijacked
at the TCP level and raw tunnel data flows. This is technically a protocol violation of
the WebSocket spec, but it's invisible to network infrastructure because the TLS
encryption makes the post-upgrade payload opaque.

### Token Format

The `X-Tunnel-Token` header carries a signed token:
```
base64(port || timestamp || hmac-sha256(port || timestamp, shared_secret))
```

Where:
- `port`: The target tunnel port (e.g., 8443) -- 2 bytes, big-endian
- `timestamp`: Unix timestamp -- 8 bytes, big-endian (token valid for 30 seconds)
- `shared_secret`: Derived from the TLS client certificate fingerprint, or a
  pre-shared key configured in the YAML

This ensures:
- Only authorized clients can establish tunnels
- Tokens are not replayable (timestamp expiration)
- Target port is cryptographically bound to the token

## Configuration Schema

### Server Configuration

New `facade` section added to the server config:

```yaml
type: server
config:
  mode: server
  # ... existing config ...
  tunnel:
    listen_address: 0.0.0.0
    listen_port: 8443          # Direct tunnel port (still used internally)
    protocol: tcp
  facade:
    enabled: true              # Enable the HTTPS facade
    listen_address: 0.0.0.0    # Facade listen address
    listen_port: 443           # Facade port (typically 443)
    hostname: "server.example.com"  # Server hostname for TLS SNI
    web_root: "hello world"    # Response for GET / (or path to HTML file)
    token_secret: "<high-entropy-secret>"  # REQUIRED for HMAC tokens
    token_ttl: 30s             # Token validity duration
    tls:                       # TLS config for the facade (can differ from tunnel TLS)
      cert_file: ""            # If empty, uses auth.cert_file
      key_file: ""             # If empty, uses auth.key_file
      ca_file: ""              # If empty, uses auth.ca_file
    tunnel_ports:              # List of tunnel ports this facade routes to
      - 8443
      - 8444
      - 8445
```

### Client Configuration

New `facade` section added to the client config:

```yaml
type: client
config:
  mode: client
  # ... existing config ...
  tunnel:
    server_address: 192.168.1.22
    server_port: 8443          # Primary direct port (tried first)
    protocol: tcp
  facade:
    enabled: true              # Enable facade fallback
    server_address: ""         # Facade server address (defaults to tunnel.server_address)
    server_port: 443           # Facade port to connect to
    direct_timeout: 3s         # How long to wait for direct connection before fallback
    token_secret: "<same-as-server>"       # REQUIRED; must match server exactly
```

## Implementation Plan

### Phase 1: Configuration Types and Validation

**Files to modify:**
- `internal/config/types/types.go` -- Add `FacadeConfig` struct
- `internal/config/validator/validator.go` -- Add facade validation rules
- `internal/config/loader.go` -- Handle facade config in version upgrades
- `configs/server.yaml` -- Add facade section
- `configs/client.yaml` -- Add facade section
- `templates/server.yaml.template` -- Add facade template variables
- `templates/client.yaml.template` -- Add facade template variables

**New types in `types.go`:**
```go
// FacadeConfig represents HTTPS facade configuration for firewall traversal
type FacadeConfig struct {
    Enabled       bool          `yaml:"enabled" json:"enabled"`
    ListenAddress string        `yaml:"listen_address" json:"listen_address"`
    ListenPort    int           `yaml:"listen_port" json:"listen_port"`
    ServerAddress string        `yaml:"server_address" json:"server_address"`
    ServerPort    int           `yaml:"server_port" json:"server_port"`
    Hostname      string        `yaml:"hostname" json:"hostname"`
    WebRoot       string        `yaml:"web_root" json:"web_root"`
    TokenSecret   string        `yaml:"token_secret" json:"token_secret"`
    TokenTTL      time.Duration `yaml:"token_ttl" json:"token_ttl"`
    DirectTimeout time.Duration `yaml:"direct_timeout" json:"direct_timeout"`
    TLS           FacadeTLS     `yaml:"tls" json:"tls"`
    TunnelPorts   []int         `yaml:"tunnel_ports" json:"tunnel_ports"`
}

// FacadeTLS represents TLS configuration specific to the facade
type FacadeTLS struct {
    CertFile string `yaml:"cert_file" json:"cert_file"`
    KeyFile  string `yaml:"key_file" json:"key_file"`
    CAFile   string `yaml:"ca_file" json:"ca_file"`
}
```

**Add `Facade FacadeConfig` field to `Config` struct.**

**Validation rules:**
- If `facade.enabled` is true on server: `listen_port` must be set (default 443),
  `tunnel_ports` must not be empty
- If `facade.enabled` is true on client: `server_port` must be set (default 443),
  `direct_timeout` must be positive (default 3s)
- `token_ttl` must be between 5s and 120s (default 30s)
- If `facade.tls` cert/key/ca are empty, they inherit from `auth` section
- `tunnel_ports` entries must be valid port numbers (1-65535)
- `facade.listen_port` must not conflict with `tunnel.listen_port`

### Phase 2: HTTPS Facade Server (`internal/facade/`)

**New package: `internal/facade/`**

**Files to create:**

#### `internal/facade/server.go` -- HTTPS Facade Server
The core facade server that:
- Starts an `http.Server` with TLS on port 443 (or configured port)
- Routes:
  - `GET /` -- Returns the configured web page (looks like a real website)
  - `GET /health` -- Returns `{"status":"ok"}` (optional, for monitoring)
  - `GET /connect` with `Upgrade: websocket` headers -- Tunnel negotiation
  - All other paths -- Returns `404 Not Found`
- On valid upgrade request:
  1. Validates the `X-Tunnel-Token` header
  2. Extracts the target port from the token
  3. Verifies the port is in the `tunnel_ports` list
  4. Responds with `101 Switching Protocols` (WebSocket accept headers)
  5. Calls `http.Hijacker` to get the raw `net.Conn`
  6. Dials the local tunnel port (e.g., `127.0.0.1:8443`)
  7. Starts bidirectional `io.Copy` between the hijacked conn and the local tunnel conn
- Uses the existing `TLSManager` for TLS configuration
- Implements graceful shutdown via `context.Context`

```go
// Server represents the HTTPS facade server
type Server struct {
    config     *types.FacadeConfig
    authConfig *types.AuthConfig
    logger     *zap.Logger
    httpServer *http.Server
    ctx        context.Context
    cancel     context.CancelFunc
    wg         sync.WaitGroup
}

func NewServer(cfg *types.FacadeConfig, authCfg *types.AuthConfig, logger *zap.Logger) *Server
func (s *Server) Start() error
func (s *Server) Stop() error
func (s *Server) handleRoot(w http.ResponseWriter, r *http.Request)
func (s *Server) handleConnect(w http.ResponseWriter, r *http.Request)
func (s *Server) validateToken(token string) (port int, err error)
func (s *Server) proxyToTunnel(hijackedConn net.Conn, tunnelPort int)
```

#### `internal/facade/token.go` -- Token Generation and Validation
```go
// GenerateToken creates an HMAC-signed token encoding the target port
func GenerateToken(port int, secret []byte) (string, error)

// ValidateToken verifies and extracts the port from a token
func ValidateToken(token string, secret []byte, ttl time.Duration) (int, error)

// ResolveSecret resolves the token secret; an explicit token_secret is
// mandatory (derivation from public material is prohibited)
func ResolveSecret(tokenSecret string, _ string) ([]byte, error)
```

Token structure (binary, then base64-encoded):
```
[2 bytes: port big-endian][8 bytes: unix timestamp big-endian][32 bytes: HMAC-SHA256]
```

The HMAC is computed over `port || timestamp` using the shared secret.

#### `internal/facade/proxy.go` -- Connection Proxying
```go
// Proxy bridges a hijacked HTTP connection to a local tunnel port
func Proxy(ctx context.Context, clientConn net.Conn, tunnelAddr string, logger *zap.Logger) error
```

This function:
1. Dials the local tunnel address (e.g., `127.0.0.1:8443`)
2. Starts bidirectional `io.Copy` (similar to `Transfer` but without rate limiting,
   since the tunnel instance handles its own rate limiting)
3. Returns when either side closes or context is cancelled

### Phase 3: HTTPS Facade Client (`internal/facade/`)

#### `internal/facade/client.go` -- Facade Client with Fallback
```go
// Client represents the HTTPS facade client
type Client struct {
    config     *types.FacadeConfig
    tunnelCfg  *types.TunnelConfig
    authConfig *types.AuthConfig
    logger     *zap.Logger
}

func NewClient(cfg *types.FacadeConfig, tunnelCfg *types.TunnelConfig, authCfg *types.AuthConfig, logger *zap.Logger) *Client

// Connect attempts direct connection first, falls back to facade
// Returns a net.Conn that can be used by the tunnel transfer logic
func (c *Client) Connect(ctx context.Context) (net.Conn, error)

// connectDirect attempts a direct TCP connection to the tunnel port
func (c *Client) connectDirect(ctx context.Context) (net.Conn, error)

// connectViaFacade establishes connection through the HTTPS facade
func (c *Client) connectViaFacade(ctx context.Context) (net.Conn, error)
```

The `Connect` method:
1. Tries `connectDirect()` with `direct_timeout` (default 3s)
2. If direct fails, logs the fallback and calls `connectViaFacade()`
3. `connectViaFacade()`:
   a. Establishes TLS connection to server:443
   b. Generates an HMAC token for the target port
   c. Sends HTTP GET with WebSocket upgrade headers and `X-Tunnel-Token`
   d. Reads the response; expects `101 Switching Protocols`
   e. Returns the underlying `net.Conn` (the TLS connection, now hijacked from HTTP)

**Key design point**: The returned `net.Conn` from `connectViaFacade` is a TLS
connection to port 443. The existing tunnel TLS handshake does NOT happen again on
top of this -- the facade connection is already TLS-encrypted. The tunnel code must
be aware that when using the facade, the connection is pre-encrypted.

### Phase 4: Integration with Existing Tunnel Code

**Files to modify:**
- `internal/tunnel/tunnel.go` -- Integrate facade into Server and Client
- `cmd/daemon/main.go` -- Start facade server alongside tunnel server

#### Server-side Integration (`tunnel.go`)

In `Server.Start()`:
```go
// After starting the tunnel listener on the configured port...
if s.config.Config.Facade.Enabled {
    s.facade = facade.NewServer(&s.config.Config.Facade, &s.config.Config.Auth, s.logger)
    if err := s.facade.Start(); err != nil {
        return fmt.Errorf("failed to start HTTPS facade: %w", err)
    }
}
```

The facade server proxies incoming facade connections to the local tunnel listener.
The tunnel `acceptLoop` sees these as normal incoming TCP connections -- it doesn't
know or care that they arrived via the facade. This is the key architectural insight:
**the facade is transparent to the tunnel layer**.

In `Server.Stop()`:
```go
if s.facade != nil {
    s.facade.Stop()
}
```

#### Client-side Integration (`tunnel.go`)

In `Client.connectLoop()`, replace the direct `net.DialTimeout` with:
```go
var conn net.Conn
var err error

if c.facadeClient != nil {
    conn, err = c.facadeClient.Connect(c.ctx)
} else {
    conn, err = net.DialTimeout("tcp4", serverAddr, 10*time.Second)
}
```

When the facade client is used and falls back to the facade connection, the returned
`net.Conn` is already TLS-encrypted. The tunnel's TLS wrapping must be skipped in
this case. The facade client's `Connect` method should return both the connection and
a flag indicating whether it was established via the facade:

```go
type ConnectResult struct {
    Conn      net.Conn
    ViaFacade bool  // If true, connection is already TLS-encrypted
}
```

In the TLS wrapping section of `connectLoop`:
```go
if result.ViaFacade {
    // Connection is already TLS-encrypted via the facade
    tunnelConn = result.Conn
} else if c.tlsManager != nil {
    tunnelConn, err = c.tlsManager.WrapConn(conn, false)
    // ...
}
```

#### Daemon Integration (`cmd/daemon/main.go`)

For the multi-instance case where a single facade server on port 443 routes to
multiple tunnel instances: the facade server should be started once in the daemon,
not per-instance. This requires a coordination mechanism.

**Option A (simpler, recommended for initial implementation):** Each server instance
that has `facade.enabled: true` starts its own facade. The first instance to bind
port 443 wins; subsequent instances log a warning that the facade is already running.
The facade discovers tunnel ports by reading the `tunnel_ports` config list.

**Option B (future enhancement):** A dedicated facade coordinator process manages
port 443 and routes to all registered tunnel instances via a shared registry.

### Phase 5: Testing

**New test files:**

#### `internal/facade/server_test.go`
- `TestFacadeServerStartStop` -- Lifecycle
- `TestFacadeRootHandler` -- GET / returns web page
- `TestFacade404Handler` -- Unknown paths return 404
- `TestFacadeUpgradeHandler` -- Valid upgrade request returns 101
- `TestFacadeInvalidToken` -- Bad token returns 403
- `TestFacadeExpiredToken` -- Expired token returns 403
- `TestFacadeUnknownPort` -- Token for unconfigured port returns 403
- `TestFacadeProxyToTunnel` -- End-to-end proxy test

#### `internal/facade/token_test.go`
- `TestTokenGenerateValidate` -- Round-trip token generation and validation
- `TestTokenExpiry` -- Expired tokens are rejected
- `TestTokenTampering` -- Modified tokens are rejected
- `TestTokenWrongSecret` -- Wrong secret rejects token
- `TestResolveSecret` -- Explicit-secret requirement and fail-closed behavior

#### `internal/facade/client_test.go`
- `TestClientDirectConnect` -- Direct connection succeeds when port is open
- `TestClientFallbackToFacade` -- Falls back when direct port is blocked
- `TestClientFacadeConnect` -- Facade connection establishes correctly

#### `internal/tunnel/tunnel_test.go` (additions)
- `TestServerWithFacade` -- Server starts facade alongside tunnel
- `TestClientWithFacade` -- Client connects via facade when direct fails
- `TestEndToEndViaFacade` -- Full tunnel data transfer via facade

#### Integration test: `test/integration/facade_test.go`
- Starts a server with facade enabled
- Starts a client configured to use facade
- Verifies data flows through the tunnel
- Verifies GET / returns the web page
- Verifies unknown paths return 404

### Phase 6: Documentation and Configuration Updates

**Files to update:**
- `docs/implementation/ARCHITECTURE.md` -- Add facade component
- `configs/server.yaml` -- Add facade section
- `configs/client.yaml` -- Add facade section
- `templates/server.yaml.template` -- Add facade template variables
- `templates/client.yaml.template` -- Add facade template variables
- `README.md` -- Add facade section to features list
- `Issues.md` -- Track facade implementation progress

**New documentation:**
- `docs/implementation/https_facade.md` -- This file (design document)
- `docs/config/facade.md` -- Facade configuration reference
- `docs/deployment/firewall_traversal.md` -- Deployment guide for restricted networks

## File Inventory

### New Files
| File | Description |
|---|---|
| `internal/facade/server.go` | HTTPS facade server with HTTP upgrade handling |
| `internal/facade/client.go` | Facade client with direct-connect fallback |
| `internal/facade/token.go` | HMAC token generation and validation |
| `internal/facade/proxy.go` | Bidirectional TCP proxy for hijacked connections |
| `internal/facade/server_test.go` | Server unit tests |
| `internal/facade/client_test.go` | Client unit tests |
| `internal/facade/token_test.go` | Token unit tests |
| `docs/implementation/https_facade.md` | This design document |
| `docs/config/facade.md` | Configuration reference |
| `docs/deployment/firewall_traversal.md` | Deployment guide |

### Modified Files
| File | Change |
|---|---|
| `internal/config/types/types.go` | Add `FacadeConfig`, `FacadeTLS` structs; add `Facade` field to `Config` |
| `internal/config/validator/validator.go` | Add `validateFacadeConfig()` |
| `internal/config/loader.go` | Handle facade config in version upgrades |
| `internal/tunnel/tunnel.go` | Integrate facade into `Server` and `Client` |
| `cmd/daemon/main.go` | No changes needed (facade starts via tunnel server) |
| `configs/server.yaml` | Add `facade:` section |
| `configs/client.yaml` | Add `facade:` section |
| `templates/server.yaml.template` | Add facade template variables |
| `templates/client.yaml.template` | Add facade template variables |
| `docs/implementation/ARCHITECTURE.md` | Add facade component |
| `README.md` | Add facade to features |
| `Issues.md` | Track facade progress |

## Security Considerations

1. **Token replay**: Tokens expire after `token_ttl` (default 30s). Clock skew between
   server and client should be less than the TTL.

2. **Token secret management**: If `token_secret` is empty, the secret is derived from
   the SHA-256 hash of the CA certificate's raw DER bytes. This means any client with
   the correct CA cert can derive the secret -- which is appropriate since they already
   have the CA cert for mTLS. For higher security, an explicit `token_secret` can be
   configured.

3. **Port scanning resistance**: The facade returns 404 for all unknown paths. The
   `/connect` endpoint only responds to properly-formatted WebSocket upgrade requests
   with valid tokens. An attacker without the shared secret cannot determine what
   tunnel ports exist.

4. **TLS layer**: The facade uses its own TLS configuration (or inherits from `auth`).
   The tunnel's mTLS authentication is NOT used for the facade HTTP layer -- the facade
   performs its own authentication via HMAC tokens. After the upgrade, the tunnel data
   is already encrypted by the facade's TLS layer.

5. **No double encryption**: When a client connects via the facade, the tunnel's TLS
   wrapping is skipped since the facade already provides TLS encryption. This avoids
   unnecessary overhead and latency.

## Implementation Order

1. **Phase 1** (Config): Add types and validation -- no behavioral changes
2. **Phase 2** (Server): Build the facade server -- can be tested independently
3. **Phase 3** (Client): Build the facade client -- can be tested against Phase 2
4. **Phase 4** (Integration): Wire into tunnel code -- end-to-end functionality
5. **Phase 5** (Testing): Comprehensive test coverage
6. **Phase 6** (Docs): Update all documentation

Estimated effort: ~1500-2000 lines of new code across all phases.
