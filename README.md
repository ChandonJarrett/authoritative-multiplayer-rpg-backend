# Authoritative Multiplayer RPG Backend

Real-time, server-authoritative multiplayer RPG backend in Go.

Two services, clearly separated:

- **API server**: ConnectRPC over HTTP/gRPC. Handles auth, accounts, characters, and game handoff.
- **Game server**: Authoritative simulation over ENet/UDP. Owns all real-time game state.

> **Status:** Runtime bootstrap is complete. The API server now mounts ConnectRPC handlers for system, auth, character, and game handoff flows. The game server lifecycle now starts its HTTP health server and Redis registry heartbeat, but the ENet host, join-token redemption, simulation loop, and snapshot broadcast loop are still incomplete. See [Current status](#current-status).

---

## Quick start

This project is **devcontainer-first**. Open it in VS Code Dev Containers or the Dev Containers CLI, everything else is automated.

On first creation, the devcontainer runs `make setup`, which:

- copies `.env.example` -> `.env` if not already present
- installs Git hooks
- installs Go tools
- downloads Go modules
- applies database migrations
- validates the environment

Once the container is ready:

```bash
make test      # run unit + integration tests
make run-api   # start the API server
make run-game  # start the game server
```

---

## Common commands

```bash
# Setup
make setup            # first-time environment setup
make doctor           # verify all tools and native deps are available

# Environment
make env-up           # start PostgreSQL + Redis
make env-down         # stop services
make env-reset        # wipe and restart service volumes

# Migrations
make migrate-up       # apply pending migrations
make migrate-down     # roll back one migration
make migrate-reset    # roll back all, then re-apply
make migrate-version  # show current migration version

# Development
make run-api          # run the API server
make run-game         # run the game server
make proto            # regenerate protobuf code

# Quality
make ci-fast          # lint, fmt, vet, unit tests; run before committing
make ci               # full CI suite

# Tests
make test             # unit + integration tests
make test-race        # tests with the race detector
make coverage         # per-function coverage summary
```

Run `make help` for the complete list.

---

## System overview

```text
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
  ______________     |      _____________
  | PostgreSQL │  <----->   │   Redis   |
  |  durable   |            | ephemeral |
  ______________            _____________
```

**PostgreSQL** stores everything that must survive a restart: users, characters, inventory.  
**Redis** stores short-lived coordination state: sessions, join tokens, server registry, character locks, and rate-limit counters.

---

## Current status

**Implemented:**

- Runtime bootstrap: config loading, logger, PostgreSQL pool, Redis client
- Configuration validation with environment variable defaults
- Redis key builder and TTL constants
- Database transaction helpers
- Migrations: users and characters tables with triggers and indexes
- Protobuf definitions and generated Go code
- Docker Compose local environment
- Devcontainer
- CI quality and test pipeline
- ConnectRPC API server lifecycle
- Health and readiness endpoints
- CORS handling
- Request ID middleware, panic recovery, HTTP logging, and RPC logging
- System ping RPC
- Auth register/login/logout RPC handlers
- Auth interceptor using Redis-backed bearer sessions
- Auth service with Argon2id password hashing and opaque session tokens
- Auth rate limiting with Redis-backed counters
- Character create/list RPC handlers
- Character service and PostgreSQL store
- Game handoff API for game-server listing and join-token issuance
- Redis session, join-token, game-server, character-lock, and rate-limit stores
- API metrics in Prometheus text format
- Game server lifecycle shell with HTTP health/readiness endpoints
- Game server Redis registration and heartbeat lifecycle
- Unit and integration tests for core foundation pieces

**Not yet implemented or incomplete:**

- Production observability: tracing, audit logs, and deeper operational metrics
- Game server ENet host lifecycle
- Game-server join-token redemption
- Character lock acquisition and renewal in the live join path
- Game simulation loop
- World snapshot broadcast loop
- Durable gameplay save/load boundaries beyond the current character data foundation

---

## Documentation

| Doc | Purpose |
|---|---|
| [docs/development.md](docs/development.md) | Environment setup, daily workflow, migrations, proto |
| [docs/configuration.md](docs/configuration.md) | Every environment variable and its default |
| [docs/testing.md](docs/testing.md) | Unit, integration, and race test guide |
| [docs/architecture/README.md](docs/architecture/README.md) | System design and component breakdown |
| [docs/adr/README.md](docs/adr/README.md) | Why key decisions were made |
