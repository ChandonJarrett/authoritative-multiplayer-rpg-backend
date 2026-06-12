# Data and Storage

The backend uses two stores intentionally. Each is used only for what it is best at.

---

## PostgreSQL: durable relational data

PostgreSQL is the source of truth for anything that must survive a process restart.

**Current schema:**
- `users`: id (UUID), email (citext, unique), password_hash
- `characters`: id (UUID), user_id (FK -> users), name (citext), map_id, position (x/y/z), timestamps

**Future:**
- Inventory
- Quests and progression
- Economy data

PostgreSQL enforces referential integrity (`user_id` -> `users`) and uses triggers to maintain `updated_at` automatically.

---

## Redis: ephemeral coordination

Redis stores short-lived state. All keys should have TTLs unless there is a deliberate reason for permanence.

| Key type | TTL | Purpose |
|---|---|---|
| Join token | 60 s | Single-use token for game server handoff |
| Session | 2 h | Active authenticated session |
| User sessions | 2 h | Set of session IDs for a user (for broadcast) |
| Server | 10 s | Game server registry entry (renewed by heartbeat) |
| Server sessions | ... | Sessions active on a specific game server |
| Character lock | 20 s | Prevents the same character loading on two servers |

A game server that crashes deregisters itself automatically when its 10 s TTL expires, no external cleanup required.

Default TTL constants are defined in `internal/cache/keys.go`.

---

## Redis key format

All Redis keys are built through `internal/cache.KeyBuilder`, never constructed by hand.

**Format:** `{app}:{env}:{type}:{id}`

**Examples:**
```
rpg-backend:production:session:abc123
rpg-backend:production:join_token:xyz789
rpg-backend:production:character_lock:char-uuid
rpg-backend:production:server:game-server-1
```

The `{app}:{env}` prefix prevents key collisions between environments sharing a Redis instance (e.g. `staging` and `production`).

`KeyBuilder` validates all segments at construction time; empty values, colons, and whitespace are rejected.

---

## Migrations

Migration files live in `migrations/` and follow the `golang-migrate` naming convention:

```
NNN_description.up.sql
NNN_description.down.sql
```

**Rules:**
- Every migration must have a matching rollback.
- Migrations should be small and independently reviewable.
- Destructive rollbacks (dropping tables, columns) must be explicit.
- Shared database objects (functions, extensions) go in `000_shared` so later migrations can depend on them.

**Current migrations:**

| File | What it does |
|---|---|
| `000_shared` | Enables `citext` extension; creates `set_updated_at()` trigger function |
| `001_users` | Creates `users` table with updated_at trigger |
| `002_characters` | Creates `characters` table with FK, index, and updated_at trigger |