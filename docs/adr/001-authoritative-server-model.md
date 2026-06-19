# ADR 001: Authoritative Server Model

**Status:** Accepted

---

## Context

Multiplayer games have two approaches to game state ownership:

- **Client-authoritative, peer-to-peer:** Clients simulate locally and synchronise with each other. Cheap to operate, but cheating is trivial, speed hacks, teleportation, item duplication, and state diverges when peers disagree.
- **Server-authoritative:** The server exclusively owns all state. Clients send inputs; the server resolves them and broadcasts canonical results. Requires a real server and is harder to implement, but eliminates entire classes of cheating and keeps state consistent by construction.

For an RPG where persistent economy, progression, and fair combat matter, client-authoritative designs are not viable.

---

## Decision

All game state lives exclusively on the server. Clients are input sources and renderers.

1. Clients capture input locally, movement and actions, and send it to the server.
2. The server applies inputs in a deterministic simulation loop, target 64Hz.
3. The server broadcasts the canonical world state to clients, target 32Hz.
4. Clients interpolate between received snapshots for smooth rendering.

The server performs all collision detection, game logic evaluation, and state mutation. It is the single source of truth.

---

## Consequences

**Benefits:**

- Cheating requires compromising the server, not the client binary.
- No client-side prediction divergence to reconcile across players.
- Replay and debugging are straightforward: record server inputs, replay deterministically.
- Future authoritative validation, anti-cheat, audit trails, is easy to add.

**Trade-offs:**

- Input latency is visible, clients feel inputs only after a server round-trip. Mitigated by client-side visual prediction that does not affect game state.
- Server capacity is required for every active session; there is no free peer-to-peer compute.
- Simulation code must be kept deterministic, no `time.Now()`, no map iteration ordering, no floating-point divergence across platforms.
