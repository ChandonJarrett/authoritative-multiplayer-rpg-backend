# Game Server

The game server owns the authoritative real-time simulation. Clients are untrusted input sources; they never directly mutate state.

---

## Lifecycle

```text
1. Load and validate config
2. Initialize structured logger
3. Set up signal-aware context
4. Connect to PostgreSQL
5. Connect to Redis
6. Create Redis key builder
7. Start game server HTTP health/readiness endpoint on GAME_HTTP_ADDR
8. Start ENet host on GAME_ENET_ADDR
9. Register game server in Redis with TTL
10. Renew game server registry heartbeat until shutdown
11. Accept and process ENet client connections
12. Wait for shutdown signal
13. Stop accepting new ENet clients
14. Stop HTTP server
15. Deregister game server from Redis
16. Close Redis
17. Close PostgreSQL
18. Exit
```

---

## Request-response loop

```text
1. Client sends an InputPacket on unreliable ENet channel 1
2. Server validates the input
3. Server applies valid input during the next simulation tick
4. Server updates authoritative world state
5. Server broadcasts SnapshotPacket to all clients on unreliable channel 1
6. Clients interpolate between snapshots for smooth rendering
```

---

## Tick rates

| Loop | Rate |
|---|---|
| Simulation tick | 64Hz |
| Snapshot broadcast | 32Hz |

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

Clients connect via ENet and send a `JoinRequest` on reliable channel 0, containing a short-lived join token issued by the API server and a `character_id`. The game server validates the token against Redis, acquires a character lock to prevent duplicate loads, and responds with `JoinResponse`.

If validation fails, the client receives an error reason and is disconnected. The entire handshake is verified by the integration test in `internal/app/integration_test.go`.

---

## Channels

| Channel | Delivery | Message types |
|---|---|---|
| 0 | Reliable | `JoinRequest`, `JoinResponse` |
| 1 | Unreliable | `InputPacket` (client to server), `SnapshotPacket` (server to client) |

Unreliable delivery is intentional for snapshots. A late packet is superseded by the next one. Retransmitting stale positional data is worse than dropping it.
