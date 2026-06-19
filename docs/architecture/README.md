# Architecture

The backend has two services: the **API server** and the **game server**.

The core rule: **the server owns all game state.** Clients send intent. The server validates, applies, and broadcasts canonical results. Clients render what the server says.

---

## Documents

| Document | What it covers |
|---|---|
| [system-overview.md](system-overview.md) | Services, responsibilities, and how everything connects |
| [runtime.md](runtime.md) | Shared startup bootstrap and shutdown sequence |
| [data-storage.md](data-storage.md) | PostgreSQL, Redis, migrations, and Redis key format |
| [protocol.md](protocol.md) | Protobuf contracts, transports, and code generation |
| [api-server.md](api-server.md) | API server transport, responsibilities, and storage usage |
| [game-server.md](game-server.md) | Game server lifecycle, planned simulation model, loop, and persistence policy |
| [security.md](security.md) | Trust model, boundaries, and rules |

---

## Design goals

- Persistent player data is safe and consistent, PostgreSQL, ACID guarantees.
- Clients cannot directly mutate authoritative game state.
- Real-time gameplay runs over a low-latency transport, ENet/UDP.
- Local development is fully reproducible and tool-managed.
- Clear boundaries between the API layer, game simulation, durable storage, and ephemeral coordination.

## Non-goals

- Peer-to-peer networking.
- Client-authoritative game state.
- Full gameplay implementation, this repository is still infrastructure-first.
- Production deployment topology.
- Browser-native gameplay transport, game uses ENet, not WebSocket.
