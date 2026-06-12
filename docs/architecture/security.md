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
- Persistent state changes (position, inventory, progression) only happen inside the server simulation.

**Authentication and sessions:**
- Session tokens are issued by the API server and stored in Redis with a TTL.
- Join tokens are short-lived (60s), single-use, and stored in Redis. They cannot be reused after redemption.
- A join token is the only way for a client to enter the game server, the server never accepts a connection without one.

**Character locking:**
- When a character enters a game server, a lock is set in Redis (20s TTL, renewed by heartbeat).
- This prevents the same character from loading on two game servers simultaneously, which would allow duplication exploits.
- If a game server crashes, the lock expires automatically after 20s.

---

## Storage rules

- **PostgreSQL** owns all durable player data. Redis must never be treated as the source of truth for progression, inventory, or economy.
- **Redis** owns ephemeral coordination. Data in Redis should have TTLs; any permanent Redis data requires a deliberate justification.

---

## Secrets

- Secrets (passwords, API keys, signing keys) belong in `.env` locally and in a dedicated secret manager in deployed environments.
- `.env` is in `.gitignore` and must never be committed. The pre-commit hook enforces this.
- `.env.example` is the only safe config file to commit, it contains no real secrets.