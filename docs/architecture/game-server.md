# Game Server

The game server owns the authoritative real-time simulation. Clients are untrusted input sources, they never directly mutate state.

The current implementation has the game server lifecycle shell in place: it starts runtime dependencies, exposes HTTP health/readiness endpoints, registers itself in Redis, renews its registry heartbeat, and shuts down gracefully. The ENet host and simulation loops are still incomplete.

---

## Current lifecycle

```text
1. Load and validate config
2. Initialize structured logger
3. Set up signal-aware context
4. Connect to PostgreSQL
5. Connect to Redis
6. Create Redis key builder
7. Start game server HTTP health/readiness endpoint on GAME_HTTP_ADDR
8. Register game server in Redis with TTL
9. Renew game server registry heartbeat until shutdown
10. Wait for shutdown signal
11. Stop HTTP server
12. Deregister game server from Redis
13. Close Redis
14. Close PostgreSQL
15. Exit
```

---

## Planned request-response loop

```text
1. Client sends an InputPacket, unreliable ENet channel 1
2. Server validates the input
3. Server applies valid input during the next simulation tick
4. Server updates authoritative world state
5. Server broadcasts SnapshotPacket to all clients, unreliable channel 1
6. Clients interpolate between snapshots for smooth rendering
```

---

## Target rates

| Loop | Target rate |
|---|---|
| Simulation tick | 64Hz |
| Snapshot broadcast | 32Hz |

These are architectural targets, not hard-coded guarantees.

---

## What clients cannot mutate

Clients send intent. The server decides outcomes. Clients do not directly control:

- Position
- Inventory
- Combat outcomes
- Progression or experience
- Economy state
- Any persistent character state

---

## Persistence policy

The game server avoids per-frame PostgreSQL writes. Durable saves happen only at explicit boundaries:

| Boundary | Example |
|---|---|
| Character load | Load position and inventory at join |
| Checkpoint | Periodic low-frequency save during a session |
| Logout | Save final state on graceful disconnect |
| Zone transfer | Save before moving to another server |

---

## Join handshake

Clients connect via ENet and send a `JoinRequest`, reliable channel 0, containing a short-lived join token issued by the API server and a `character_id`. The game server validates the token, acquires a character lock in Redis, and responds with `JoinResponse`.

If validation fails, the client receives an error reason and is disconnected.

> This handshake is the intended design. Join-token redemption, lock acquisition in the live join path, and ENet connection handling are not complete yet.
