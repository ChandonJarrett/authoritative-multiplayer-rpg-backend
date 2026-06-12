# Runtime Bootstrap

Both services share a common bootstrap in `internal/app`. Entry points (`cmd/api`, `cmd/game`) are intentionally thin, they only handle service-specific startup after the shared runtime is ready.

The `Runtime` struct holds:
- Loaded and validated `Config`
- `*slog.Logger`
- Signal-aware `context.Context` and `context.CancelFunc`
- `*pgxpool.Pool` (PostgreSQL)
- `*redis.Client` (Redis)

---

## API startup sequence

```
1. Load and validate config
2. Initialize structured logger
3. Set up signal-aware context (SIGINT / SIGTERM)
4. Connect to PostgreSQL (ping on connect; fail fast if unavailable)
5. Connect to Redis (ping on connect; fail fast if unavailable)
6. Mount ConnectRPC handler          <- TODO
7. Start HTTP listener on API_HTTP_ADDR  <- TODO
8. Wait for shutdown signal
9. Drain in-flight requests          <- TODO
10. Close Redis
11. Close PostgreSQL
12. Exit
```

## Game startup sequence

```
1. Load and validate config
2. Initialize structured logger
3. Set up signal-aware context (SIGINT / SIGTERM)
4. Connect to PostgreSQL
5. Connect to Redis
6. Register game server in Redis (with TTL heartbeat)  <- TODO
7. Initialize ENet host on GAME_ENET_ADDR              <- TODO
8. Start HTTP health endpoint on GAME_HTTP_ADDR        <- TODO
9. Start simulation loop (64Hz)                        <- TODO
10. Start snapshot broadcast loop (32Hz)               <- TODO
11. Wait for shutdown signal
12. Stop accepting new clients
13. Release session locks and deregister from Redis    <- TODO
14. Close Redis
15. Close PostgreSQL
16. Exit
```

---

## Failure behaviour

**On startup:** Invalid config, unavailable PostgreSQL, or unavailable Redis all cause an immediate fatal exit. The process does not start in a degraded state.

**On shutdown:** Resources are released in reverse init order, Redis first, then PostgreSQL. The `defer rt.Close()` call in each entry point guarantees this even if startup panics partway through.

---

## Current state

Both services currently initialize the shared runtime and wait for a shutdown signal. No listeners or loops are started yet.

- The API server does not mount ConnectRPC handlers or bind `API_HTTP_ADDR`.
- The game server does not initialize ENet, run a simulation loop, or bind `GAME_ENET_ADDR` / `GAME_HTTP_ADDR`.

Steps marked `<- TODO` in the sequences above are not yet implemented.