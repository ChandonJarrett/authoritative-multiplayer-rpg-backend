# API Server

The API server owns account, authentication, character management, and game-server handoff.

---

## Transport

The API server uses [ConnectRPC](https://connectrpc.com) to serve a single Go handler on one port that speaks three protocols simultaneously:

- **Connect protocol:** for browser clients, `connect-es` or `fetch`
- **gRPC:** for native clients, internal services, and tooling like `grpcurl`
- **gRPC-Web:** for older browser gRPC clients

No proxy or gateway is needed. The service contract is defined once in `api.proto`.

In development, the handler runs on a standard `net/http` mux with HTTP/1 and cleartext HTTP/2 enabled. In production, TLS should terminate at the load balancer.

---

## Responsibilities

- System ping RPC
- User registration and login
- Session creation and logout/session revocation
- Bearer-session authentication for protected RPCs
- Auth rate limiting for public auth endpoints
- Character listing and creation
- Game-server listing
- Join-token issuance for game-server handoff
- Health and readiness endpoints
- Request logging, RPC logging, request IDs, panic recovery, CORS, and metrics
- Future account and admin workflows

---

## Storage

| Store | Used for |
|---|---|
| PostgreSQL | Durable user and character data |
| Redis | Sessions, join tokens, game server registry, character locks, rate-limit counters |

---

## Mounted routes

| Route | Purpose |
|---|---|
| `/healthz` | Liveness check |
| `/readyz` | Readiness check against PostgreSQL and Redis |
| `/metrics` | Prometheus-style in-memory metrics, when enabled |
| ConnectRPC service paths | System, Auth, Character, and Game RPCs |

---

## Boundaries

The API server must not:

- Run game simulation
- Process movement input
- Broadcast world snapshots
- Hold or mutate active world state

Those responsibilities belong exclusively to the game server.
