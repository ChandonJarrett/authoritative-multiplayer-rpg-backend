# ADR 002: ENet for Game Transport

**Status:** Accepted

---

## Context

Real-time game servers have different transport requirements from web services:

- **TCP:** Guarantees ordered, reliable delivery, but head-of-line blocking means a dropped packet stalls all subsequent data until retransmission. For position updates at 32Hz, a delayed update is worse than a dropped one, by the time it arrives, two newer updates have superseded it.
- **Raw UDP:** No head-of-line blocking, but also no reliability, ordering, or connection management. Implementing those correctly on top of raw UDP is significant work.
- **WebSockets / HTTP:** Designed for request-response and streaming, not high-frequency binary telemetry. Adds unnecessary framing overhead.
- **QUIC:** Solves head-of-line blocking at the stream level, but adds implementation complexity. The Go ecosystem for game-oriented QUIC is immature.
- **ENet:** A purpose-built C library for game networking over UDP. Provides per-channel sequencing, selective reliability (reliable and unreliable channels on the same connection), connection management, and keep-alives, without TCP-style global ordering.

ENet is battle-tested in production games (including Godot's built-in high-level networking) and has a stable C library accessible via CGo.

---

## Decision

Use ENet (via `libenet`) as the transport layer for all game server ↔ client communication.

Channels are allocated by message semantics:

| Channel | Delivery | Messages |
|---|---|---|
| 0 | Reliable | `JoinRequest`, `JoinResponse`, inventory updates, infrequent events |
| 1 | Unreliable | `InputPacket` (client -> server), `SnapshotPacket` (server -> client) |

Unreliable delivery is correct for high-frequency snapshots: a dropped snapshot is simply superseded by the next broadcast. Retransmitting stale positional data would deliver incorrect history.

---

## Consequences

**Benefits:**
- No head-of-line blocking for state broadcasts, a dropped snapshot does not delay the next one.
- Reliable and unreliable delivery coexist on the same connection without a second socket.
- Proven in games at scale; the C library is stable and well-understood.

**Trade-offs:**
- Requires CGo and `libenet` as a native dependency, which adds build-time complexity (resolved by `scripts/install-apt-deps.sh` and the devcontainer).
- Firewall and NAT traversal is harder than TCP; clients in restricted networks may need hole-punching or a relay.
- Not natively accessible from browser clients; a WebSocket bridge would be needed if browser gameplay is ever required.