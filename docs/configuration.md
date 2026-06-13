# Configuration Reference

All configuration is loaded from environment variables. Local development also loads values from `.env`.

Use `.env.example` as your local template, it contains safe defaults for every variable. Never commit `.env`.

---

## Quick reference

| Variable | Default | Notes |
|---|---|---|
| `APP_NAME` | `rpg-backend` | Redis key namespace |
| `APP_ENV` | `local` | `local` / `development` / `testing` / `staging` / `production` |
| `LOG_LEVEL` | `info` | `debug` / `info` / `warn` / `error` |
| `LOG_FORMAT` | `text` | `text` / `json` |
| `API_HTTP_ADDR` | `:8080` | API server bind address |
| `GAME_ENET_ADDR` | `:7777` | Game server ENet bind address |
| `GAME_HTTP_ADDR` | `:8081` | Game server HTTP bind address |
| `SHUTDOWN_TIMEOUT` | `10s` | Go duration string |
| `POSTGRES_HOST` | `localhost` | `postgres` inside devcontainer |
| `POSTGRES_PORT` | `5432` | |
| `POSTGRES_USER` | *(required)* | |
| `POSTGRES_PASSWORD` | *(required)* | |
| `POSTGRES_DB` | *(required)* | |
| `POSTGRES_SSLMODE` | `disable` | `disable` / `require` / `verify-ca` / `verify-full` |
| `REDIS_HOST` | `localhost` | `redis` inside devcontainer |
| `REDIS_PORT` | `6379` | |
| `REDIS_PASSWORD` | *(empty)* | |
| `REDIS_DB` | `0` | |

> Do **not** set `POSTGRES_URL` or `REDIS_ADDR` directly. The application builds these internally from the individual variables above.

---

## Application

### `APP_NAME`

Application namespace. Used as the first segment in Redis key construction and in log output.

Default: `rpg-backend`

### `APP_ENV`

Runtime environment. Affects test behaviour (`APP_ENV=testing` is required for integration tests).

Allowed values: `local`, `development`, `testing`, `staging`, `production`

Default: `local`

### `LOG_LEVEL`

Minimum log level to emit.

Allowed values: `debug`, `info`, `warn`, `error`

Default: `info`

### `LOG_FORMAT`

Log output format. Use `json` for production or log aggregation pipelines.

Allowed values: `text`, `json`

Default: `text`

---

## Network

### `API_HTTP_ADDR`

Bind address for the API server HTTP listener.

Default: `:8080`

### `GAME_ENET_ADDR`

Bind address for the game server ENet/UDP listener.

Default: `:7777`

### `GAME_HTTP_ADDR`

Bind address for the game server HTTP listener (health checks, admin endpoints).

Default: `:8081`

### `SHUTDOWN_TIMEOUT`

How long to wait for in-flight work to finish before forcing shutdown. Must be a valid Go duration string (e.g. `500ms`, `10s`, `1m`).

Default: `10s`

---

## PostgreSQL

### `POSTGRES_HOST`

Default: `localhost`  
Inside devcontainer/Compose: `postgres`

### `POSTGRES_PORT`

Default: `5432`

### `POSTGRES_USER`

Required. No default.

### `POSTGRES_PASSWORD`

Required. No default.

### `POSTGRES_DB`

Required. No default.

### `POSTGRES_SSLMODE`

Allowed values: `disable`, `require`, `verify-ca`, `verify-full`

Use `verify-full` in production.

Default: `disable`

### Connection pool

| Variable | Default | Description |
|---|---|---|
| `POSTGRES_MAX_CONNS` | `10` | Maximum open connections |
| `POSTGRES_MIN_CONNS` | `1` | Minimum idle connections |
| `POSTGRES_MAX_CONN_LIFETIME` | `1h` | Max age of a connection |
| `POSTGRES_MAX_CONN_IDLE_TIME` | `30m` | Max idle time before closing |
| `POSTGRES_HEALTH_CHECK_PERIOD` | `1m` | How often to ping idle connections |

All duration values must be valid Go duration strings.

---

## Redis

### `REDIS_HOST`

Default: `localhost`  
Inside devcontainer/Compose: `redis`

### `REDIS_PORT`

Default: `6379`

### `REDIS_PASSWORD`

Default: *(empty)*

### `REDIS_DB`

Redis database index (0–15).

Default: `0`

### Connection pool and timeouts

| Variable | Default | Description |
|---|---|---|
| `REDIS_DIAL_TIMEOUT` | `5s` | Timeout for establishing a new connection |
| `REDIS_READ_TIMEOUT` | `3s` | Timeout for reading a response |
| `REDIS_WRITE_TIMEOUT` | `3s` | Timeout for sending a command |
| `REDIS_POOL_SIZE` | `10` | Maximum number of connections |
| `REDIS_MIN_IDLE_CONNS` | `1` | Minimum idle connections to maintain |

All duration values must be valid Go duration strings.

---

## Generated values

The application derives these internally, do not set them directly:

| Variable | Built from |
|---|---|
| `POSTGRES_URL` | `POSTGRES_HOST` + `PORT` + `USER` + `PASSWORD` + `DB` + `SSLMODE` |
| `REDIS_ADDR` | `REDIS_HOST` + `REDIS_PORT` |

Setting either directly will cause `config.Load()` to return an error.