# Protocol

Protocol Buffers define the shared typed contracts for all communication. The `.proto` files are the single source of truth. Generated Go code is committed so CI can detect drift.

---

## File layout

```text
proto/rpg/v1/          <- proto source files, edit these
internal/protocol/     <- generated Go code, do not edit by hand
buf.yaml               <- buf module config and lint rules
buf.gen.yaml           <- code generation config, plugins and output paths
```

---

## Proto file responsibilities

| File | Purpose |
|---|---|
| `common.proto` | Shared value types used by both API and game, `Vec2`, identifiers |
| `api.proto` | ConnectRPC service definitions: system, auth, characters, game handoff |
| `game.proto` | ENet packet messages: join handshake, movement input, world snapshots |

`game.proto` contains only message types, no service definitions. The game server encodes and decodes them directly by channel and packet framing.

---

## API services

| Service | Purpose |
|---|---|
| `SystemService` | Ping/connectivity check |
| `AuthService` | Register, login, logout |
| `CharacterService` | Create and list player characters |
| `GameService` | List game servers and issue join tokens |

---

## Transport split

| Transport | Used for | Protocol |
|---|---|---|
| ConnectRPC, HTTP/gRPC | API server, browser and native clients | `api.proto` service definitions |
| ENet / UDP | Game server, high-frequency real-time traffic | `game.proto` plain messages |

> **Rule:** Control plane over ConnectRPC. Simulation plane over ENet/UDP.  
> Do not use ConnectRPC for movement input or snapshot broadcasts. Do not use ENet/UDP for account or character APIs.

---

## ConnectRPC

A single Go handler serves three protocols simultaneously on one port:

| Protocol | Used by |
|---|---|
| Connect, HTTP/1.1 or HTTP/2, JSON or binary | Browser clients via `connect-es` or `fetch` |
| gRPC, HTTP/2, binary protobuf | Native clients, internal services, `grpcurl` |
| gRPC-Web | Older browser gRPC clients |

No proxy or gateway required.

---

## ENet channels

| Channel | Delivery | Message types |
|---|---|---|
| 0 | Reliable | `JoinRequest`, `JoinResponse` |
| 1 | Unreliable | `InputPacket` client -> server, `SnapshotPacket` server -> client |

Unreliable delivery is intentional for snapshots. A late packet is superseded by the next one. Retransmitting stale positional data is worse than dropping it.

---

## Code generation

Three plugins are invoked by `buf generate`:

| Plugin | Output |
|---|---|
| `protoc-gen-go` | Go types for all messages |
| `protoc-gen-go-grpc` | gRPC server and client interfaces |
| `protoc-gen-connect-go` | ConnectRPC handler interfaces and client stubs |

All output goes to `internal/protocol/` with `paths=source_relative`.

Plugins are managed via `go.mod`'s `tool` block, no separate installation required.

```bash
make proto           # regenerate from .proto sources
make proto-check     # verify committed files match sources, CI
make proto-lint      # lint .proto files
make proto-breaking  # check for wire-breaking changes against main
```
