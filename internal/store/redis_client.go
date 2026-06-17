package store

import (
	"context"
	"time"

	goredis "github.com/redis/go-redis/v9"
)

// RedisCommander defines the subset of Redis commands used by the session store.
type RedisCommander interface {
	Set(ctx context.Context, key string, value any, expiration time.Duration) *goredis.StatusCmd
	Get(ctx context.Context, key string) *goredis.StringCmd
	Del(ctx context.Context, keys ...string) *goredis.IntCmd
	SAdd(ctx context.Context, key string, members ...any) *goredis.IntCmd
	Expire(ctx context.Context, key string, expiration time.Duration) *goredis.BoolCmd
}
