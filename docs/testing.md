# Testing

Three test layers:

| Layer | Command | Needs services? | Speed |
|---|---|---|---|
| Unit | `make test-unit` | No | Fast |
| Integration | `make test-integration` | Yes | Medium |
| Race | `make test-race` | No | Slow |

Run all tests (unit + integration): `make test`

---

## Unit tests

Fast and self-contained. No PostgreSQL or Redis required. Run anywhere.

```bash
make test-unit
```

Covers:
- Config parsing, defaults, and validation
- Redis key construction and validation
- Logger setup and level filtering
- Transaction helper guard clauses (nil pool, nil function)

---

## Integration tests

Require PostgreSQL and Redis to be running with migrations applied.

```bash
# In the devcontainer (services are always running):
make test-integration

# On a local machine:
make env-up
make migrate-up
make test-integration
```

Integration tests use the `//go:build integration` build tag. On a local machine they **skip gracefully** when services are unavailable. In CI they **fail hard** (the `CI=true` environment variable triggers this, see `internal/testutil/integration.go`).

Covers:
- PostgreSQL connectivity, pool stats, and `SELECT 1`
- Redis connectivity, set/get round-trip, and pool stats
- Transaction commit and rollback behavior
- Migration schema: tables, columns, types, indexes, triggers, and functions

---

## Race tests

Run the full test suite with Go's race detector enabled.

```bash
make test-race
```

Use before merging any changes that touch shared state or concurrency. Race tests are slower and are not required in every local run, CI runs them automatically.

---

## Coverage

```bash
make coverage         # prints per-function summary to stdout
```

To open an interactive HTML report:

```bash
go test -count=1 -coverprofile=coverage.out ./...
go tool cover -html=coverage.out
```

---

## Conventions

- Tests that don't need external services must **not** use `//go:build integration`.
- Tests that need PostgreSQL or Redis **must** use `//go:build integration`.
- Use unique test keys in Redis (e.g. keyed by `t.Name()`) with short TTLs.
- Prefer creating temporary tables in PostgreSQL integration tests; drop them in a `defer` or `t.Cleanup`.
- Never rely on pre-existing rows in the database.