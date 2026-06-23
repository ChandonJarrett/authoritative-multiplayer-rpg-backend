# Devcontainer

The devcontainer is the recommended development environment for this project.

It provides:

- Go 1.26.4
- Native ENet library (`libenet-dev`)
- PostgreSQL 18
- Redis 8 (optional auth via `REDIS_PASSWORD`)
- Protobuf tooling (`buf`, `protoc-gen-go`, `protoc-gen-go-grpc`, `protoc-gen-connect-go`)
- `golang-migrate`
- `golangci-lint` (16 enabled linters: errcheck, govet, staticcheck, gosec, revive, gocritic, and more)
- `govulncheck`
- `gofumpt` + `goimports` for formatting (configured in `.golangci.yaml`)

## Getting started

Open the repository in VS Code Dev Containers or the Dev Containers CLI.

On first creation, the container automatically runs `make setup`. If setup fails, rebuild:

> **VS Code:** `Dev Containers: Rebuild Container`

To reset PostgreSQL and Redis volumes (run from the host):

```bash
docker compose down -v
```

## Redis authentication

By default, Redis runs without a password for local development. To enable authentication:

1. Set `REDIS_PASSWORD` in `.env`
2. Run `make env-reset` to recreate the Redis container with auth enabled

All tooling handles both authenticated and unauthenticated Redis:
- `make redis-shell` automatically passes `-a` when `REDIS_PASSWORD` is set
- The Compose healthcheck uses `CMD-SHELL` with `redis-cli -a` for compatibility

## Useful commands

```bash
make setup         # re-run first-time setup
make doctor        # verify tools and native dependencies
make test          # unit + integration tests
make ci-fast       # pre-commit checks (lint, fmt, vet, unit tests)
make lint          # run golangci-lint with 16 linters
make migrate-up    # apply pending migrations
make proto         # regenerate protobuf code
make run-api       # run the API server
make run-game      # run the game server
```