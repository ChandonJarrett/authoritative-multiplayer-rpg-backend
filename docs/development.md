# Development Guide

This project is **devcontainer-first**.

Use the devcontainer unless you have a specific reason not to. Local development is supported as a fallback, but if behavior differs, the devcontainer wins. CI is the final source of truth.

---

## Devcontainer setup (recommended)

**Host machine requirements:**
- Docker
- VS Code with the Dev Containers extension, or the Dev Containers CLI

Open the repository in the devcontainer. On first creation, the container automatically runs `make setup`, which:

- copies `.env.example` -> `.env` (if missing)
- installs Git hooks
- installs Go tools
- downloads Go modules
- applies database migrations
- validates required tools and native dependencies

Once setup completes, run `make test` to verify everything works.

---

## Daily workflow

```bash
make ci-fast       # pre-commit checks: lint, fmt, vet, unit tests
make test          # unit + integration tests
make proto         # regenerate protobuf after editing .proto files
make run-api       # start the API server
make run-game      # start the game server
```

---

## Database migrations

Migrations live in `migrations/`. Each has a `.up.sql` and a matching `.down.sql`.

```bash
make migrate-up       # apply all pending migrations
make migrate-down     # roll back one migration
make migrate-reset    # roll back all, then re-apply from scratch
make migrate-version  # show the current migration version
```

> `make setup` already runs `make migrate-up`, so you only need these manually for rollbacks or after adding new migrations.

---

## Shell access

```bash
make db-shell      # open a psql shell inside the postgres container
make redis-shell   # open a redis-cli shell inside the redis container
```

---

## Protocol Buffers

Proto source files live in `proto/`. Generated Go code lives in `internal/protocol/` and is committed to the repository so CI can detect drift.

```bash
make proto           # regenerate from .proto sources, then fmt
make proto-check     # verify committed files match sources (used in CI)
make proto-lint      # lint .proto files against buf rules
make proto-breaking  # check for wire-breaking changes against main
```

Add a new RPC: edit the `.proto` file -> `make proto` -> implement the generated interface.

---

## Local machine setup (advanced fallback)

**Requirements:**
- Go 1.26.4
- Docker and Docker Compose
- `libenet-dev`, `pkg-config`, `build-essential`, `postgresql-client`, `redis-tools`

On Debian/Ubuntu, install OS dependencies:

```bash
sudo bash scripts/install-apt-deps.sh
```

Then:

```bash
make env-up    # start PostgreSQL and Redis in Docker
make setup     # install tools, apply migrations, validate environment
make test      # verify everything works
```

### Environment variable notes

Inside the devcontainer, Compose sets two host overrides automatically:

```
POSTGRES_HOST=postgres
REDIS_HOST=redis
```

On a local machine, the defaults from `.env.example` (`localhost`) work as-is when using `make env-up`. Do not change `.env.example` to use Docker service names, local workflows still need `localhost`.