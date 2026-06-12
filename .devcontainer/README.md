# Devcontainer

The devcontainer is the recommended development environment for this project.

It provides:

- Go 1.26.4
- Native ENet library (`libenet-dev`)
- PostgreSQL 18
- Redis 8
- Protobuf tooling (`buf`, `protoc-gen-go`, `protoc-gen-go-grpc`, `protoc-gen-connect-go`)
- `golang-migrate`
- `golangci-lint`
- `govulncheck`

## Getting started

Open the repository in VS Code Dev Containers or the Dev Containers CLI.

On first creation, the container automatically runs `make setup`. If setup fails, rebuild:

> **VS Code:** `Dev Containers: Rebuild Container`

To reset PostgreSQL and Redis volumes (run from the host):

```bash
docker compose down -v
```

## Useful commands

```bash
make setup         # re-run first-time setup
make doctor        # verify tools and native dependencies
make test          # unit + integration tests
make ci-fast       # pre-commit checks (lint, fmt, vet, unit tests)
make migrate-up    # apply pending migrations
make proto         # regenerate protobuf code
make run-api       # run the API server
make run-game      # run the game server
```