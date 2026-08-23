# Dead-Peer Detection — Design

## Problem

A TCP connection whose peer vanished (power loss, NAT timeout, cable pull)
can appear healthy locally for hours: the stack never learns the peer is
gone until it tries to send, and a quiet tunnel sends nothing. Both roles
previously held such half-open connections indefinitely — the server kept
its single slot occupied and the client never reconnected.

## Chosen Mechanism

Two complementary layers, both transparent to the tunneled byte stream:

1. **TCP keepalive** (`tunnel.keepalive_seconds`, default off).
   Applied with `SetKeepAliveConfig` (Idle=period, Interval=period/3,
   Count=3), so a dead peer is detected in roughly `2 × period` once the
   connection is otherwise idle at the OS level.

2. **Activity-tracked idle deadlines** (`tunnel.idle_timeout_seconds`,
   default off). Every counted byte pushes the connection's read/write
   deadline out by the idle window (`countingConn`). If no traffic flows
   for the whole window — including TUN-side silence — the next deadline
   check fails, the transfer unwinds, and normal recovery takes over.

Both knobs are per-connection settings: changing them requires a restart
(reconnects pick up new values because the client re-reads config per
attempt only for retry scheduling).

## Why Not Application-Level Ping

The tunnel is a *transparent byte pipe* between two TUN devices: every byte
written into one end must exit the other, unmodified and in order.
Injecting PING/PONG control messages would require a framing layer
(length-prefix + message type), which changes the wire protocol, breaks
mixed-version deployments, and costs throughput on the data path. If a
framing layer is ever introduced (e.g., for multipath or compression),
keepalive frames become trivial to add behind version negotiation; until
then, OS-level keepalive plus idle deadlines deliver the operational
outcome with zero protocol impact.

## Recovery Semantics

- **Client**: an idle-timeout or keepalive failure surfaces as a transfer
  error; `connectLoop` treats it like any failed dial and follows the
  jittered reconnect schedule (`tunnel.reconnect.*`).
- **Server**: the transfer ends, the single connection slot is released,
  and the listener keeps accepting. `activeConns` accounting is covered by
  the same path as ordinary disconnects.
