# System Overview

The backend is split into two services that own distinct concerns and never share responsibilities.

```text
 Browser / Client
         |
         V
 ________________       ________________
 |  API server  |       |  Game server |
 |  ConnectRPC  |       |  ENet / UDP  |
 |  HTTP / gRPC |       |              |
 ________________       ________________
              |            |
              \____________/
                     |
  ______________     |      _____________
  | PostgreSQL |     |      |   Redis    |
  |  durable   |     |      | ephemeral  |
  ______________     |      _____________
                     |
              Handoff via Redis
             (join tokens, locks)
```

---

## API server

Owns account, authentication, character management, and game handoff.

**Responsibilities:**

- System ping RPC
- User registration, login, and logout
- Session creation and bearer-session authentication
- Auth rate limiting
- Character listing and creation
- Game-server listing
- Join-token issuance, hands off a player to the game server
- Future account and admin APIs
- Health, readiness, and metrics endpoints

**Must not:**

- Run game simulation logic
- Process movement input
- Broadcast world snapshots
- Own active world state

---

## Game server

Owns the authoritative real-time simulation.

**Responsibilities:**

- Game server process lifecycle with graceful shutdown
- HTTP health/readiness endpoint
- Redis game-server registration with heartbeat renewal
- ENet host lifecycle and client connection handling
- Join-token validation and character lock acquisition
- Authoritative simulation tick at 64Hz
- World snapshot broadcast to clients at 32Hz
- Graceful disconnect handling and deregistration

**Must not:**

- Perform per-frame PostgreSQL writes
- Trust client-supplied state
- Mutate durable player data outside of explicit save boundaries: load, checkpoint, logout, zone transfer

---

## Storage split

| Store | Owns | Why |
|---|---|---|
| **PostgreSQL** | Users, characters, inventory, progression | Durable, relational, ACID |
| **Redis** | Sessions, join tokens, server registry, character locks, rate limits | Ephemeral, TTL-based, sub-millisecond |

---

## Directory layout

```text
cmd/
  api/main.go       <- API server entry point
  game/main.go      <- Game server entry point
internal/
  api/              <- ConnectRPC server, handlers, middleware
  app/              <- Shared runtime bootstrap and service wiring
  auth/             <- Password hashing and token generation
  cache/            <- Redis client + key builder
  config/           <- Config loading + validation
  db/               <- PostgreSQL pool + transaction helpers
  domain/           <- Shared domain models and errors
  game/             <- Game server lifecycle shell
  logger/           <- Structured logger, slog
  observability/    <- HTTP/RPC metrics helpers
  protocol/         <- Generated protobuf code, do not edit by hand
  service/          <- Auth, character, and game handoff services
  store/            <- Persistence interfaces and implementations
  validate/         <- Input validation helpers
migrations/         <- SQL migration files
proto/              <- Protobuf source files
docs/               <- Documentation
```
