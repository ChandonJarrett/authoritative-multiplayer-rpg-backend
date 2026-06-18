# Authoritative Multiplayer RPG Backend

Real-time, server-authoritative multiplayer RPG backend in Go.

Two services, clearly separated:

- **API server**: ConnectRPC over HTTP/gRPC. Handles auth, accounts, and characters.
- **Game server**: Authoritative simulation over ENet/UDP. Owns all real-time game state.

> **Status:** Runtime bootstrap is complete. ConnectRPC handlers and the ENet game loop are not yet implemented. See [Current status](#current-status).

---

## Quick start

This project is **devcontainer-first**. Open it in VS Code Dev Containers or the Dev Containers CLI, everything else is automated.

On first creation, the devcontainer runs `make setup`, which:

- copies `.env.example` -> `.env` (if not already present)
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

**PostgreSQL** stores everything that must survive a restart: users, characters, inventory.  
**Redis** stores short-lived coordination state: sessions, join tokens, server registry, character locks.

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
- System ping RPC
- Auth register/login RPC handlers
- Auth interceptor using Redis-backed bearer sessions
- Auth service with password hashing and opaque session tokens
- Character create/list RPC handlers
- Character service and PostgreSQL store
- Game handoff API for join-token issuance
- Redis session, join-token, and game-server store foundations
- Unit and integration tests for core foundation pieces

**Not yet implemented or incomplete:**

- Production observability: request logs, metrics, tracing, audit logs
- Auth abuse protection: rate limiting and brute-force controls
- Session revocation/logout flows
- Game server ENet host lifecycle
- Game-server join-token redemption
- Character lock acquisition and renewal
- Game simulation loop
- World snapshot broadcast loop

---

## Documentation

| Doc | Purpose |
|---|---|
| [docs/development.md](docs/development.md) | Environment setup, daily workflow, migrations, proto |
| [docs/configuration.md](docs/configuration.md) | Every environment variable and its default |
| [docs/testing.md](docs/testing.md) | Unit, integration, and race test guide |
| [docs/architecture/README.md](docs/architecture/README.md) | System design and component breakdown |
| [docs/adr/README.md](docs/adr/README.md) | Why key decisions were made |