# Security

The foundation: **clients are untrusted.**

---

## Trust model

| Actor | Trust level |
|---|---|
| Server process | Trusted, owns all authoritative state |
| Client application | Untrusted, treated as a potential attacker |
| Client input | Always validated server-side before applying |
| Client-reported state | Never accepted as authoritative |

---

## Boundaries

**Game server:**

- Client input must be validated before it is applied to simulation state.
- Clients may visually predict movement locally, but visual prediction never becomes authoritative state.
- Persistent state changes, position, inventory, progression, only happen inside the server simulation.

**Authentication and sessions:**

- Passwords are hashed with Argon2id before storage.
- Session tokens are opaque, high-entropy tokens issued by the API server and stored in Redis with a TTL.
- Protected RPCs require `Authorization: Bearer <token>`.
- Logout revokes the active session token.
- Public auth endpoints are rate-limited using Redis-backed counters.
- Join tokens are short-lived, 60s, single-use, and stored in Redis. They cannot be reused after redemption.
- A join token is the only intended way for a client to enter the game server.

**Character locking:**

- When a character enters a game server, a lock is set in Redis, 20s TTL, renewed by heartbeat.
- This prevents the same character from loading on two game servers simultaneously, which would allow duplication exploits.
- If a game server crashes, the lock expires automatically after 20s.
- The full lock acquisition and join path is verified by the integration test in `internal/app/integration_test.go`.

---

## Storage rules

- **PostgreSQL** owns all durable player data. Redis must never be treated as the source of truth for progression, inventory, or economy.
- **Redis** owns ephemeral coordination. Data in Redis should have TTLs; any permanent Redis data requires a deliberate justification.

---

## Error handling

Domain errors are mapped to stable ConnectRPC codes. Internal error details are intentionally hidden from clients.

---

## Secrets

- Secrets, passwords, API keys, signing keys, belong in `.env` locally and in a dedicated secret manager in deployed environments.
- `.env` is in `.gitignore` and must never be committed. The pre-commit hook enforces this.
- `.env.example` is the only safe config file to commit, it contains no real secrets.
