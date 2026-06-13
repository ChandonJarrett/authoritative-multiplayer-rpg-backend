# Game Server

The game server owns the authoritative real-time simulation. Clients are untrusted input sources, they never directly mutate state.

---

## Request-response loop

```
1. Client sends an InputPacket (unreliable ENet channel 1)
2. Server validates the input
3. Server applies valid input during the next simulation tick
4. Server updates authoritative world state
5. Server broadcasts SnapshotPacket to all clients (unreliable channel 1)
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

Clients connect via ENet and send a `JoinRequest` (reliable channel 0) containing a short-lived join token issued by the API server and a `character_id`. The game server validates the token, acquires a character lock in Redis, and responds with `JoinResponse`.

If validation fails, the client receives an error reason and is disconnected.