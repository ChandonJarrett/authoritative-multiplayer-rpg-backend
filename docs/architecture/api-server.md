# API Server

The API server owns account, authentication, and character management.

---

## Transport

The API server uses [ConnectRPC](https://connectrpc.com) to serve a single Go handler on one port that speaks three protocols simultaneously:

- **Connect protocol:** for browser clients (`connect-es` or `fetch`)
- **gRPC:** for native clients, internal services, and tooling like `grpcurl`
- **gRPC-Web:** for older browser gRPC clients

No proxy or gateway is needed. The service contract is defined once in `api.proto`.

In development, the handler runs on a standard `net/http` mux with `h2c` (cleartext HTTP/2). In production, TLS terminates at the load balancer.

---

## Responsibilities

- User registration and login
- Session creation
- Character listing and creation
- Join-token issuance for game-server handoff
- Future account and admin workflows
- Health and readiness endpoints

---

## Storage

| Store | Used for |
|---|---|
| PostgreSQL | Durable user and character data |
| Redis | Sessions, join tokens, short-lived coordination state |

---

## Boundaries

The API server must not:
- Run game simulation
- Process movement input
- Broadcast world snapshots
- Hold or mutate active world state

Those responsibilities belong exclusively to the game server.