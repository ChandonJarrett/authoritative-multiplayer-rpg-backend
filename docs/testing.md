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

Integration tests use the `//go:build integration` build tag. They use `testutil.SkipOnServiceError` (see `internal/testutil/integration.go`) to handle service-connection failures in one of two ways:

- **Locally** (CI not set): skips gracefully with a message.
- **In CI** (`CI=true`): fails hard, because services are always expected to be available.

Covers:

- PostgreSQL connectivity, pool stats, and `SELECT 1`
- Redis connectivity, set/get round-trip, and pool stats
- Transaction commit and rollback behavior
- Migration schema: tables, columns, types, indexes, triggers, functions, and extensions

---

## Race tests

Run the full test suite with Go's race detector enabled.

```bash
make test-race
```

Use before merging any changes that touch shared state or concurrency. Race tests are slower and are not required in every local run; CI runs them automatically.

---

## Coverage

```bash
make coverage              # unit tests only
make coverage-integration  # unit + integration tests
```

Both commands print a per-function summary. The integration coverage profile (`coverage-integration.out`) is the authoritative one for CI reporting.
