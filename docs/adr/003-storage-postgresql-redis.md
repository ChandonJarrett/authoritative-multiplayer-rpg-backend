# ADR 003: PostgreSQL + Redis Storage Split

**Status:** Accepted

---

## Context

The backend needs two distinct kinds of storage:

1. **Durable relational data:** user accounts, characters, inventory, progression. Must survive process restarts, must be consistent under concurrent writes, and will be queried relationally, e.g. load all characters for this user.
2. **Ephemeral coordination data:** active sessions, short-lived join tokens, server heartbeats, character locks, rate-limit counters. Valid only while the relevant process is alive, must expire automatically, and must be accessible quickly from multiple server instances.

Using a single store for both is a poor fit: a relational database is overkill and too slow for TTL-based coordination; a key-value store cannot express relational queries or enforce referential integrity. Either choice forces an unacceptable compromise on one axis.

---

## Decision

Use **PostgreSQL** for persistent relational data and **Redis** for ephemeral coordination. They are independent services with separate connection pools and are never used interchangeably.

**PostgreSQL owns:**

- Users and authentication, email and password hash
- Characters, owner, name, world position, timestamps
- Future persistent game data, inventory, quests, economy

**Redis owns:**

- Join tokens, 60s TTL, single-use
- Active sessions, 2h TTL
- User sessions mapping, for session revocation and future broadcast workflows
- Game server registry, 10s TTL, renewed by heartbeat; crashed servers deregister automatically
- Character locks, 20s TTL, prevents a character loading on two game servers simultaneously
- Rate-limit counters, window configured by environment

All Redis keys are built through `internal/cache.KeyBuilder`, which enforces the `{app}:{env}:{type}:{id}` namespace format and prevents environment collisions.

---

## Consequences

**Benefits:**

- Each store is used for what it does best; no compromises on either axis.
- PostgreSQL ACID guarantees cover all durable writes.
- Redis TTLs eliminate the need for background cleanup jobs.
- Crashed game servers deregister themselves automatically when their registry TTL expires.
- Both stores scale independently, PostgreSQL read replicas and Redis Cluster.

**Trade-offs:**

- Two infrastructure dependencies to operate, monitor, and back up.
- Operations spanning both stores cannot be made atomic without a distributed transaction or saga pattern.
- Connection pool configuration and health checks must be maintained for both.
