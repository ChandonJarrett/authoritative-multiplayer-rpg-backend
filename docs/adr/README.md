# Architecture Decision Records

Significant architectural decisions are recorded here. Each ADR captures the context, the decision, and the consequences so future contributors understand why things are the way they are.

| # | Title | Status |
|---|---|---|
| [001](001-authoritative-server-model.md) | Authoritative server model | Accepted |
| [002](002-enet-udp-transport.md) | ENet for game transport | Accepted |
| [003](003-storage-postgresql-redis.md) | PostgreSQL + Redis storage split | Accepted |
| [004](004-connectrpc-api-server.md) | ConnectRPC for the API server | Accepted |

## Format

Each ADR follows the structure: **Context -> Decision -> Consequences**.

When a decision is revisited, mark the old ADR as superseded and link to the new one, do not delete or modify historical records.