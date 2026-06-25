# Runtime Bootstrap

Both services share a common bootstrap in `internal/app`. Entry points, `cmd/api` and `cmd/game`, are intentionally thin. They only handle service-specific startup after the shared runtime is ready.

The `Runtime` struct holds:

- Loaded and validated `Config`
- `*slog.Logger`
- Signal-aware `context.Context` and `context.CancelFunc`
- `*pgxpool.Pool`, PostgreSQL
- Redis client

---

## API startup sequence

```text
1. Load and validate config
2. Initialize structured logger
3. Set up signal-aware context, SIGINT / SIGTERM
4. Connect to PostgreSQL, ping on connect, fail fast if unavailable
5. Connect to Redis, ping on connect, fail fast if unavailable
6. Create Redis key builder
7. Initialize PostgreSQL and Redis-backed stores
8. Initialize auth, character, and game services
9. Initialize HTTP/RPC metrics
10. Mount health, readiness, metrics, and ConnectRPC routes
11. Install middleware: request ID, panic recovery, logging, observability, CORS
12. Install RPC interceptors: logging, metrics, auth rate limiting, bearer auth
13. Start HTTP listener on API_HTTP_ADDR
14. Wait for shutdown signal
15. Drain in-flight requests until SHUTDOWN_TIMEOUT
16. Close Redis
17. Close PostgreSQL
18. Exit
```

## Game startup sequence

```text
1. Load and validate config
2. Initialize structured logger
3. Set up signal-aware context, SIGINT / SIGTERM
4. Connect to PostgreSQL
5. Connect to Redis
6. Create Redis key builder
7. Create game server with ENet addr, HTTP addr, logger, shutdown timeout, and Redis registry
8. Start HTTP health/readiness endpoint on GAME_HTTP_ADDR
9. Initialize ENet host on GAME_ENET_ADDR
10. Register game server in Redis with TTL heartbeat
11. Accept ENet client connections, redeem join tokens, acquire character locks
12. Start simulation loop, 64Hz
13. Start snapshot broadcast loop, 32Hz
14. Wait for shutdown signal
15. Stop accepting new ENet clients
16. Stop HTTP server
17. Deregister game server from Redis
18. Close Redis
19. Close PostgreSQL
20. Exit
```

---

## Failure behaviour

**On startup:** Invalid config, unavailable PostgreSQL, or unavailable Redis all cause an immediate fatal exit. The process does not start in a degraded state.

**On shutdown:** Resources are released in reverse init order, Redis first, then PostgreSQL. The `defer rt.Close()` call in each entry point guarantees this even if startup fails partway through.

---

## Current state

The API server is a fully operational ConnectRPC service with system, auth, character, and game handoff handlers mounted.

The game server runs its complete lifecycle: ENet host, join-token redemption, character lock acquisition, 64Hz simulation tick, and 32Hz snapshot broadcast. The full end-to-end flow is verified by `internal/app/integration_test.go`.
