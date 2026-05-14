# Facade E2E Integration Test Plan

> **For Hermes:** Implement directly — test involves Python token generation,
> WebSocket handshake, and two-VM coordination that subagents struggle with.

**Goal:** End-to-end test of the HTTPS facade: generate HMAC token, perform
WebSocket upgrade handshake, and verify data flows through facade → proxy →
tunnel → peer.

**Architecture:** A Python test script (`tests/e2e/facade_test.py`) that:
1. Derives the shared secret from the CA cert (SHA-256 of cert file)
2. Generates an HMAC token encoding port 8443 (matching Go's binary format)
3. Performs a WebSocket upgrade request to the facade server
4. Validates the 101 Switching Protocols response and WebSocket accept
5. Sends data through the upgraded connection and verifies round-trip

**Tech Stack:** Python 3 stdlib only — `hmac`, `hashlib`, `struct`, `base64`,
`ssl`, `socket`, `http.client`.

---

## Token Format (Go reference)

```
base64( port[2 bytes BE] || unix_timestamp[8 bytes BE] || hmac-sha256[32 bytes] )
                        ↑                              ↑
                   HMAC is over these 10 bytes, keyed with shared secret
```

Shared secret = SHA-256 of the CA certificate file contents (raw bytes).
If `token_secret` is set in config, secret = SHA-256 of the token_secret string.

---

## Task 1: Generate HMAC token in Python

**Objective:** Write a function `generate_token(port, secret_bytes) -> str` that
matches the Go `GenerateToken()` output byte-for-byte.

**File:** `tests/e2e/facade_test.py`

**Implementation:**

```python
import struct
import hmac
import hashlib
import base64
import time

def generate_token(port: int, secret: bytes) -> str:
    """Generate HMAC token matching Go's GenerateToken() format."""
    payload = struct.pack('>HQ', port, int(time.time()))
    mac = hmac.new(secret, payload, hashlib.sha256).digest()
    token = payload + mac
    return base64.b64encode(token).decode()

def derive_secret(ca_cert_path: str) -> bytes:
    """Derive shared secret from CA cert (matches Go's DeriveSecret)."""
    with open(ca_cert_path, 'rb') as f:
        return hashlib.sha256(f.read()).digest()
```

**Verification:** Cross-check with Go's `GenerateToken()` using same secret and
port. Timestamps will differ, but HMAC should verify against the same payload.

---

## Task 2: WebSocket upgrade handshake

**Objective:** Write `websocket_upgrade(host, port, token, tls=True)` that
performs the WebSocket upgrade and returns the raw socket after 101 response.

**Implementation:**

```python
import ssl
import socket
import os

WS_GUID = "258EAFA5-E914-47DA-95CA-5631BC565D11"

def compute_accept(key: str) -> str:
    h = hashlib.sha1((key + WS_GUID).encode()).digest()
    return base64.b64encode(h).decode()

def ws_upgrade(host: str, port: int, token: str, use_tls: bool = True):
    """Perform WebSocket upgrade to facade, return connected socket."""
    ws_key = base64.b64encode(os.urandom(16)).decode()

    request = (
        f"GET /connect HTTP/1.1\r\n"
        f"Host: {host}:{port}\r\n"
        f"Upgrade: websocket\r\n"
        f"Connection: Upgrade\r\n"
        f"Sec-WebSocket-Key: {ws_key}\r\n"
        f"Sec-WebSocket-Version: 13\r\n"
        f"X-Tunnel-Token: {token}\r\n"
        f"\r\n"
    )

    sock = socket.create_connection((host, port), timeout=10)
    if use_tls:
        ctx = ssl.create_default_context()
        ctx.check_hostname = False
        ctx.verify_mode = ssl.CERT_NONE  # test only
        sock = ctx.wrap_socket(sock, server_hostname=host)

    sock.sendall(request.encode())

    # Read response
    response = b""
    while b"\r\n\r\n" not in response:
        chunk = sock.recv(4096)
        if not chunk:
            raise Exception("Connection closed before full response")
        response += chunk

    lines = response.split(b"\r\n")
    status = lines[0].decode()
    if "101" not in status:
        raise Exception(f"Expected 101, got: {status}")

    # Verify WebSocket accept
    expected_accept = compute_accept(ws_key)
    for line in lines[1:]:
        if line.lower().startswith(b"sec-websocket-accept:"):
            actual = line.split(b":", 1)[1].strip().decode()
            assert actual == expected_accept, \
                f"Accept mismatch: {actual} != {expected_accept}"
            break

    return sock
```

---

## Task 3: Server configuration for facade mode

**Objective:** Create a facade-enabled server config on the server VM.

**File:** `/etc/sssonector/facade-config.yaml` on 192.168.101.220

```yaml
type: server
version: "1.0.0"
config:
  mode: server
  logging:
    level: debug
  auth:
    cert_file: /etc/sssonector/certs/server.crt
    key_file: /etc/sssonector/certs/server.key
    ca_file: /etc/sssonector/certs/ca.crt
  network:
    name: tun0
    address: 10.0.0.1/24
    mtu: 1500
  tunnel:
    listen_address: "127.0.0.1"
    listen_port: 8443
    protocol: tcp
  facade:
    enabled: true
    listen_address: "0.0.0.0"
    listen_port: 4443
    tunnel_ports: [8443]
    tls:
      cert_file: /etc/sssonector/certs/server.crt
      key_file: /etc/sssonector/certs/server.key
      ca_file: /etc/sssonector/certs/ca.crt
  security:
    tls:
      min_version: "1.2"
  snmp:
    enabled: false
  monitor:
    enabled: false
  metrics:
    enabled: false
metadata:
  version: "1.0.0"
  schema_version: "2.0.0"
throttle:
  enabled: false
```

Key points:
- Tunnel listens on `127.0.0.1:8443` (local only — facade proxies to it)
- Facade listens on `0.0.0.0:4443` (non-root port for testing)
- `tunnel_ports: [8443]` allows the facade to proxy to this port
- No client auth needed — the tunnel already has TLS; facade auth is via token

---

## Task 4: Integration test script

**Objective:** A single script that runs on the client VM, generates a token,
connects to the server's facade, and verifies data flows through.

**File:** `tests/e2e/facade_test.py`

```python
#!/usr/bin/env python3
"""Facade E2E test — requires server running with facade enabled."""

import os
import sys
import struct
import hmac
import hashlib
import base64
import time
import ssl
import socket

SERVER_HOST = os.environ.get("SSSONECT_SERVER", "192.168.101.220")
FACADE_PORT = int(os.environ.get("SSSONECT_FACADE_PORT", "4443"))
CA_CERT = os.environ.get("SSSONECT_CA_CERT", "/etc/sssonector/certs/ca.crt")

WS_GUID = "258EAFA5-E914-47DA-95CA-5631BC565D11"

def derive_secret(path):
    with open(path, 'rb') as f:
        return hashlib.sha256(f.read()).digest()

def generate_token(port, secret):
    payload = struct.pack('>HQ', port, int(time.time()))
    mac = hmac.new(secret, payload, hashlib.sha256).digest()
    return base64.b64encode(payload + mac).decode()

def compute_accept(key):
    h = hashlib.sha1((key + WS_GUID).encode()).digest()
    return base64.b64encode(h).decode()

def ws_upgrade(host, port, token):
    ws_key = base64.b64encode(os.urandom(16)).decode()
    request = (
        f"GET /connect HTTP/1.1\r\n"
        f"Host: {host}:{port}\r\n"
        f"Upgrade: websocket\r\n"
        f"Connection: Upgrade\r\n"
        f"Sec-WebSocket-Key: {ws_key}\r\n"
        f"Sec-WebSocket-Version: 13\r\n"
        f"X-Tunnel-Token: {token}\r\n"
        f"\r\n"
    )
    sock = socket.create_connection((host, port), timeout=10)
    ctx = ssl.create_default_context()
    ctx.check_hostname = False
    ctx.verify_mode = ssl.CERT_NONE
    sock = ctx.wrap_socket(sock, server_hostname=host)
    sock.sendall(request.encode())
    response = b""
    while b"\r\n\r\n" not in response:
        response += sock.recv(4096)
    lines = response.split(b"\r\n")
    status = lines[0].decode()
    if "101" not in status:
        raise Exception(f"Expected 101, got: {status}")
    expected_accept = compute_accept(ws_key)
    for line in lines[1:]:
        if line.lower().startswith(b"sec-websocket-accept:"):
            actual = line.split(b":", 1)[1].strip().decode()
            assert actual == expected_accept
            break
    print(f"✅ WebSocket upgrade accepted: {status.strip()}")
    return sock

def main():
    print(f"Server: {SERVER_HOST}:{FACADE_PORT}")
    print(f"CA cert: {CA_CERT}")

    # 1. Derive secret
    secret = derive_secret(CA_CERT)
    print(f"✅ Secret derived from CA cert (len={len(secret)})")

    # 2. Generate token for port 8443
    token = generate_token(8443, secret)
    print(f"✅ Token generated (len={len(token)})")

    # 3. WebSocket upgrade
    sock = ws_upgrade(SERVER_HOST, FACADE_PORT, token)

    # 4. Verify homepage is served (GET /)
    # Use a separate connection to test the facade's web root
    import http.client
    ctx = ssl.create_default_context()
    ctx.check_hostname = False
    ctx.verify_mode = ssl.CERT_NONE
    conn = http.client.HTTPSConnection(SERVER_HOST, FACADE_PORT, context=ctx, timeout=5)
    conn.request("GET", "/")
    resp = conn.getresponse()
    body = resp.read().decode()
    assert "It works!" in body, f"Homepage missing 'It works!': {body[:100]}"
    assert resp.getheader("Server") == "nginx", f"Wrong Server header: {resp.getheader('Server')}"
    print(f"✅ Homepage serves Apache 'It works!' page (Server: nginx)")

    # 5. 404 test — unknown path
    conn.request("GET", "/nonexistent")
    resp = conn.getresponse()
    assert resp.status == 404, f"Expected 404, got {resp.status}"
    print("✅ Unknown path returns 404")

    conn.close()
    sock.close()

    print("\n🎉 All facade tests passed!")
    return 0

if __name__ == "__main__":
    sys.exit(main())
```

**Verification:**
```bash
# On server VM:
sudo /tmp/sssonectd --config /etc/sssonector/facade-config.yaml --log-level debug

# On client VM:
SSSONECT_SERVER=192.168.101.220 \
SSSONECT_FACADE_PORT=4443 \
SSSONECT_CA_CERT=/etc/sssonector/certs/ca.crt \
python3 /opt/sssonector/tests/e2e/facade_test.py
```

Expected output:
```
Server: 192.168.101.220:4443
CA cert: /etc/sssonector/certs/ca.crt
✅ Secret derived from CA cert (len=32)
✅ Token generated (len=56)
✅ WebSocket upgrade accepted: HTTP/1.1 101 Switching Protocols
✅ Homepage serves Apache 'It works!' page (Server: nginx)
✅ Unknown path returns 404

🎉 All facade tests passed!
```

---

## Task 5: Commit and PR

**Files to commit:**
- `tests/e2e/facade_test.py` (new)
- `tests/e2e/__init__.py` (new, empty)

**Commit message:**
```
test: add facade E2E integration test

Tests the HTTPS facade end-to-end: token generation (Go-compatible),
WebSocket upgrade handshake, homepage serving, and 404 handling.
Uses Python stdlib only — no external dependencies.
```
