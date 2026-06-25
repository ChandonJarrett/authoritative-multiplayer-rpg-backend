# Authoritative Multiplayer RPG Backend

A production-grade, server-authoritative multiplayer RPG backend written in Go. Designed with a clear separation between the control plane and the simulation plane, using industry-proven infrastructure and a fully validated integration test that exercises the complete end-to-end flow.

## At a glance

- **Two specialized services**, an API server (ConnectRPC, HTTP/gRPC) for account and character management, and a game server (ENet/UDP) for authoritative real-time simulation.
- **The server owns all state.** Clients send only intent; the server validates, simulates, and broadcasts canonical results at fixed tick rates.
- **Durable PostgreSQL** for users, characters, and progression; **ephemeral Redis** for sessions, join tokens, server registry, and rate limiting. Each store does what it does best.
- **Full integration test** proves the complete API-to-game-server handoff: register, authenticate, create a character, obtain a join token, connect via ENet, join the game world, receive snapshots, send movement input, and verify world state updates. Both servers shut down cleanly.
- **Devcontainer-first** development with one-command setup, comprehensive Makefile targets, multi-linter CI pipeline, and migration management.

---

## Quick start

This project is **devcontainer-first**. Open it in VS Code Dev Containers or the Dev Containers CLI, everything else is automated.

On first creation, the devcontainer runs `make setup`, which:

- copies `.env.example` to `.env` if not already present
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
make fmt              # format all source files (gofumpt + goimports)
make lint             # run 16 golangci-lint linters
make ci-fast          # lint, fmt, vet, unit tests; run before committing
make ci               # full CI suite (includes vuln check + race detector)

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

**PostgreSQL** stores everything that must survive a restart: users, characters, inventory.  
**Redis** stores short-lived coordination state: sessions, join tokens, server registry, character locks, and rate-limit counters.  
**Handoff:** The API server issues a short-lived join token stored in Redis. The game server redeems it, acquires a character lock, and the client transitions from the API control plane to the game simulation plane seamlessly.

---

## What is implemented

The full end-to-end flow is operational, verified by an integration test that starts both servers, runs through the complete lifecycle, and shuts them down cleanly:

- **Runtime bootstrap:** config loading, structured logging, PostgreSQL connection pool, Redis client
- **Configuration:** environment variable defaults and validation for all 25+ settings
- **Migrations:** users and characters tables with citext, triggers, indexes, and foreign keys
- **Protobuf contracts:** common types, API service definitions, game packet messages, generated Go code
- **API server:** ConnectRPC handler serving Connect, gRPC, and gRPC-Web on a single port
  - System ping
  - User registration, login, and logout with Argon2id password hashing
  - Bearer-session authentication interceptor backed by Redis
  - Auth rate limiting with Redis fixed-window counters
  - Character creation, listing, and PostgreSQL-backed persistence
  - Game server listing and join-token issuance for handoff
  - Health, readiness, and Prometheus metrics endpoints
  - Middleware: request IDs, panic recovery, structured HTTP/RPC logging, CORS
- **Game server:** authoritative simulation over ENet/UDP
  - ENet host lifecycle with connection handling
  - Join-token redemption and character lock acquisition
  - 64Hz simulation tick driving entity movement
  - 32Hz world snapshot broadcast to connected clients
  - HTTP health and readiness endpoints
  - Redis server registry with heartbeat renewal
  - Graceful shutdown with deregistration
- **Storage:** PostgreSQL for durable data, Redis for ephemeral coordination
  - Session management, join tokens, server registry, character locks, rate limiting
  - Structured Redis key format with environment namespace isolation
- **Quality:** 16-linter CI pipeline, gofumpt formatting, unit tests, integration tests, race detector
- **Development:** devcontainer-first setup, one-command environment, comprehensive Makefile

**Roadmap:** production observability (tracing, audit logs), durable gameplay save and load boundaries, inventory and progression systems.

---

## Documentation

| Doc | Purpose |
|---|---|
| [docs/development.md](docs/development.md) | Environment setup, daily workflow, migrations, proto |
| [docs/configuration.md](docs/configuration.md) | Every environment variable and its default |
| [docs/testing.md](docs/testing.md) | Unit, integration, and race test guide |
| [docs/architecture/README.md](docs/architecture/README.md) | System design and component breakdown |
| [docs/adr/README.md](docs/adr/README.md) | Why key decisions were made |
