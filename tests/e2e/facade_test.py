#!/usr/bin/env python3
"""Facade E2E integration test for SSSonector.

Tests the HTTPS facade end-to-end:
1. HMAC token generation (Go-compatible binary format)
2. WebSocket upgrade handshake via /connect
3. Homepage serving (Apache 'It works!' page)
4. 404 handling for unknown paths

Requires:
- Server running with facade enabled on FACADE_PORT
- CA certificate accessible at CA_CERT path
- Python 3 stdlib only (no external dependencies)

Usage:
  SSSONECT_SERVER=192.168.101.220 \
  SSSONECT_FACADE_PORT=4443 \
  SSSONECT_CA_CERT=/etc/sssonector/certs/ca.crt \
  python3 facade_test.py
"""

import os
import sys
import struct
import hmac
import hashlib
import base64
import time
import ssl
import socket
import http.client

# ── Configuration ────────────────────────────────────────────────────────────
SERVER_HOST = os.environ.get("SSSONECT_SERVER", "192.168.101.220")
FACADE_PORT = int(os.environ.get("SSSONECT_FACADE_PORT", "4443"))
CA_CERT = os.environ.get("SSSONECT_CA_CERT", "/etc/sssonector/certs/ca.crt")
TUNNEL_PORT = int(os.environ.get("SSSONECT_TUNNEL_PORT", "8443"))

# WebSocket magic GUID from RFC 6455 §4.2.2
WS_GUID = "258EAFA5-E914-47DA-95CA-5631BC565D11"


# ── Token Generation (matches Go's GenerateToken) ────────────────────────────

def derive_secret(path: str) -> bytes:
    """Derive shared secret from CA certificate file.

    Matches Go's DeriveSecret(): SHA-256 of the raw file contents.
    If the file doesn't exist, returns empty bytes (test will fail clearly).
    """
    try:
        with open(path, "rb") as f:
            return hashlib.sha256(f.read()).digest()
    except FileNotFoundError:
        print(f"FATAL: CA cert not found at {path}", file=sys.stderr)
        sys.exit(1)


def generate_token(port: int, secret: bytes) -> str:
    """Generate HMAC-signed token matching Go's GenerateToken().

    Token format (Go reference):
        base64( port[2 bytes BE] || unix_timestamp[8 bytes BE] || hmac-sha256[32 bytes] )

    The HMAC is computed over the 10-byte payload (port + timestamp),
    keyed with the shared secret.
    """
    payload = struct.pack(">HQ", port, int(time.time()))
    mac = hmac.new(secret, payload, hashlib.sha256).digest()
    token = payload + mac
    return base64.b64encode(token).decode()


# ── WebSocket Upgrade Handshake ──────────────────────────────────────────────

def compute_accept(key: str) -> str:
    """Compute Sec-WebSocket-Accept per RFC 6455 §4.2.2."""
    h = hashlib.sha1((key + WS_GUID).encode()).digest()
    return base64.b64encode(h).decode()


def ws_upgrade(host: str, port: int, token: str) -> socket.socket:
    """Perform WebSocket upgrade to the facade /connect endpoint.

    Returns the raw socket after a successful 101 Switching Protocols response.
    Raises Exception on any failure.
    """
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

    # Wrap in TLS — skip verification since we're testing with self-signed certs
    ctx = ssl.create_default_context()
    ctx.check_hostname = False
    ctx.verify_mode = ssl.CERT_NONE
    sock = ctx.wrap_socket(sock, server_hostname=host)

    sock.sendall(request.encode())

    # Read response headers until \r\n\r\n
    response = b""
    while b"\r\n\r\n" not in response:
        chunk = sock.recv(4096)
        if not chunk:
            raise ConnectionError("Server closed connection before full response")
        response += chunk

    header_part, _ = response.split(b"\r\n\r\n", 1)
    lines = header_part.split(b"\r\n")

    # Check status line
    status = lines[0].decode()
    if "101" not in status:
        raise AssertionError(f"Expected 101 Switching Protocols, got: {status.strip()}")

    # Verify Sec-WebSocket-Accept
    expected_accept = compute_accept(ws_key)
    accept_found = False
    for line in lines[1:]:
        if line.lower().startswith(b"sec-websocket-accept:"):
            actual = line.split(b":", 1)[1].strip().decode()
            if actual != expected_accept:
                raise AssertionError(
                    f"WebSocket accept mismatch: got {actual}, expected {expected_accept}"
                )
            accept_found = True
            break

    if not accept_found:
        raise AssertionError("Response missing Sec-WebSocket-Accept header")

    return sock


# ── HTTP Helpers ─────────────────────────────────────────────────────────────

def http_get(host: str, port: int, path: str, timeout: float = 5.0):
    """Perform an HTTPS GET request (no cert verification — test only)."""
    ctx = ssl.create_default_context()
    ctx.check_hostname = False
    ctx.verify_mode = ssl.CERT_NONE
    conn = http.client.HTTPSConnection(host, port, context=ctx, timeout=timeout)
    conn.request("GET", path)
    resp = conn.getresponse()
    body = resp.read().decode()
    headers = {k.lower(): v for k, v in resp.getheaders()}
    conn.close()
    return resp.status, body, headers


# ── Main ─────────────────────────────────────────────────────────────────────

def main() -> int:
    passed = 0
    failed = 0

    def check(description: str, condition: bool, detail: str = ""):
        nonlocal passed, failed
        if condition:
            print(f"  ✅ {description}")
            passed += 1
        else:
            print(f"  ❌ {description}: {detail}")
            failed += 1

    print(f"Server: {SERVER_HOST}:{FACADE_PORT}")
    print(f"CA cert: {CA_CERT}")
    print(f"Tunnel port: {TUNNEL_PORT}")
    print()

    # ── 1. Secret Derivation ─────────────────────────────────────────────
    secret = derive_secret(CA_CERT)
    check("Secret derived from CA cert", len(secret) == 32,
          f"expected 32 bytes, got {len(secret)}")

    # ── 2. Token Generation ──────────────────────────────────────────────
    token = generate_token(TUNNEL_PORT, secret)
    # Token structure: 42 bytes before base64 (2+8+32), base64 expands to 56 chars
    decoded = base64.b64decode(token)
    check("Token decodes to correct size", len(decoded) == 42,
          f"expected 42 bytes, got {len(decoded)}")
    # Extract and verify port
    port_from_token = struct.unpack(">H", decoded[:2])[0]
    check("Token encodes correct port", port_from_token == TUNNEL_PORT,
          f"expected {TUNNEL_PORT}, got {port_from_token}")

    # ── 3. Homepage (GET /) ──────────────────────────────────────────────
    try:
        status, body, headers = http_get(SERVER_HOST, FACADE_PORT, "/")
        check("Homepage returns 200", status == 200, f"got {status}")
        check("Server header spoofs nginx",
              headers.get("server", "") == "nginx",
              f"got '{headers.get('server', '')}'")
        check("Homepage contains 'It works!'",
              "It works!" in body or "Hello" in body,
              f"body preview: {body[:100]}")
        check("Homepage is HTML",
              "<html" in body.lower() or "<!doctype" in body.lower(),
              f"body preview: {body[:100]}")
    except ConnectionRefusedError:
        check("Homepage accessible", False, "connection refused — is the facade server running?")
        return 1
    except Exception as e:
        check("Homepage accessible", False, str(e))
        return 1

    # ── 4. 404 Handling ──────────────────────────────────────────────────
    try:
        status, body, headers = http_get(SERVER_HOST, FACADE_PORT, "/nonexistent-path")
        check("Unknown path returns 404", status == 404, f"got {status}")
    except Exception as e:
        check("404 test", False, str(e))

    # ── 5. WebSocket Upgrade to /connect ─────────────────────────────────
    try:
        sock = ws_upgrade(SERVER_HOST, FACADE_PORT, token)
        check("WebSocket upgrade accepted (101)", True)
        sock.close()
    except AssertionError as e:
        # 404 on /connect is expected if token is invalid — that's the facade
        # silently rejecting. Check if it's a token validation issue.
        check("WebSocket upgrade", False, str(e))
    except ConnectionRefusedError:
        check("WebSocket upgrade", False, "connection refused")
    except Exception as e:
        check("WebSocket upgrade", False, str(e))

    # ── Summary ──────────────────────────────────────────────────────────
    total = passed + failed
    print(f"\n{'='*50}")
    print(f"Results: {passed}/{total} passed")
    if failed > 0:
        print(f"         {failed}/{total} FAILED")
        return 1
    print("🎉 All facade tests passed!")
    return 0


if __name__ == "__main__":
    sys.exit(main())
