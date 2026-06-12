# System Overview

The backend is split into two services that own distinct concerns and never share responsibilities.

```
 Browser / Client
         |
         V
 ________________       ________________
 |  API server  |       |  Game server |
 |  ConnectRPC  |   ->  │  ENet / UDP  |
 |  HTTP / gRPC |       │              |
 ________________       ________________
              |            |
              \____________/
                     |
  ______________     |        _____________
  | PostgreSQL │  <----->     │   Redis   |
  |  durable   |              | ephemeral |
  ______________              _____________
```

---

## API server

Owns account, authentication, and character management.

**Responsibilities:**
- User registration and login
- Session creation
- Character listing and creation
- Join-token issuance (hands off a player to the game server)
- Future account and admin APIs
- Health and readiness endpoints

**Must not:**
- Run game simulation logic
- Process movement input
- Broadcast world snapshots
- Own active world state

---

## Game server

Owns the authoritative real-time simulation.

**Responsibilities:**
- ENet host lifecycle
- Client connection handling
- Join-token validation
- Character lock acquisition
- Authoritative simulation tick (target: 64Hz)
- World snapshot broadcasts to clients (target: 32Hz)
- Session heartbeat updates in Redis
- Graceful disconnect handling

**Must not:**
- Perform per-frame PostgreSQL writes
- Trust client-supplied state
- Mutate durable player data outside of explicit save boundaries (load, checkpoint, logout, zone transfer)

---

## Storage split

| Store | Owns | Why |
|---|---|---|
| **PostgreSQL** | Users, characters, inventory, progression | Durable, relational, ACID |
| **Redis** | Sessions, join tokens, server registry, character locks | Ephemeral, TTL-based, sub-millisecond |

---

## Directory layout

```
cmd/
  api/main.go       <- API server entry point
  game/main.go      <- Game server entry point

internal/
  app/              <- Shared runtime bootstrap
  cache/            <- Redis client + key builder
  config/           <- Config loading + validation
  db/               <- PostgreSQL pool + transaction helpers
  logger/           <- Structured logger (slog)
  protocol/         <- Generated protobuf code (do not edit by hand)

migrations/         <- SQL migration files
proto/              <- Protobuf source files
docs/               <- Documentation
```