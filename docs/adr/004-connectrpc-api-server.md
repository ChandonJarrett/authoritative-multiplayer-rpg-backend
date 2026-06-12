# ADR 004: ConnectRPC for the API Server

**Status:** Accepted

---

## Context

The API server needs to serve browser-facing HTTP traffic alongside typed gRPC contracts. The options considered:

| Option | Problem |
|---|---|
| **Pure gRPC** | Browsers cannot call gRPC directly, HTTP/2 framing is not exposed through the browser Fetch API. A translation proxy would always be required. |
| **gRPC-gateway** | Generates a reverse-proxy that transcodes REST/JSON to gRPC via `google.api.http` annotations. The REST surface is awkward, streaming support is limited, and it couples proto files to a transport concern. |
| **Separate BFF service** | Flexible, but adds a full network hop and service to maintain for no functional benefit at this stage. |
| **Pure REST + OpenAPI** | Decouples from proto, loses schema-first codegen, requires a separate type system for browser clients. |
| **ConnectRPC** | A single Go handler serves browser clients (via the Connect protocol), native gRPC clients, and gRPC-Web clients simultaneously. No proxy, no gateway, no additional service. |

---

## Decision

Use [ConnectRPC](https://connectrpc.com) as the protocol layer for the API server.

Service definitions live in `api.proto`. The `protoc-gen-connect-go` plugin generates handler interfaces and typed client stubs. The API server mounts the handler on a standard `net/http` mux wrapped with `h2c` for cleartext HTTP/2 in development.

```
Browser (connect-es or fetch)    ->  Connect protocol (HTTP/1.1 or HTTP/2, JSON or binary)
Native client / grpcurl          ->  gRPC (HTTP/2, binary protobuf)
                                          |
                                          V
                               API server ConnectRPC handler
                                          |
                                          V
                                  PostgreSQL / Redis
```

---

## Consequences

**Benefits:**
- One handler, one port, three protocols (Connect, gRPC, gRPC-Web), no proxy or gateway.
- `api.proto` is the single contract; no REST annotations or OpenAPI specs to maintain separately.
- Browser clients get a typed TypeScript client via `connect-es` from the same proto files.
- Standard `net/http` mux means middleware (logging, auth, CORS, rate limiting) applies uniformly.
- Adding a new RPC: edit `api.proto` -> `make proto` -> implement the generated interface.

**Trade-offs:**
- Browser clients require `connect-es` (or equivalent) rather than plain `fetch` against a hand-rolled REST API.
- gRPC streaming works natively; browser streaming requires the Connect or gRPC-Web protocol, not SSE or WebSocket.

---

## Rules

1. All API service contracts are defined in `api.proto`.
2. Do not hand-write JSON HTTP handlers for operations that could be defined as RPCs.
3. Do not use ConnectRPC for real-time gameplay traffic, that belongs on the game server via ENet/UDP.
4. Browser clients use the Connect protocol. Native clients may use Connect or gRPC.