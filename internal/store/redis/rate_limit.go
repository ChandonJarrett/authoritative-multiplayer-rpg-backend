package redis

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/ChandonJarrett/authoritative-multiplayer-rpg-backend/internal/cache"
	"github.com/ChandonJarrett/authoritative-multiplayer-rpg-backend/internal/domain"
)

// RateLimiter implements fixed-window Redis-backed rate limiting.
type RateLimiter struct {
	client cache.Client
	keys   cache.KeyBuilder
	window time.Duration
	limit  int
	now    func() time.Time
}

// NewRateLimiter creates a Redis-backed fixed-window limiter.
func NewRateLimiter(client cache.Client, keys cache.KeyBuilder, window time.Duration, limit int) (*RateLimiter, error) {
	if client == nil {
		return nil, domain.ErrUnavailable
	}
	if window <= 0 {
		return nil, fmt.Errorf("rate limit window must be > 0: %w", domain.ErrInvalidArgument)
	}
	if limit <= 0 {
		return nil, fmt.Errorf("rate limit burst must be > 0: %w", domain.ErrInvalidArgument)
	}
	return &RateLimiter{
		client: client,
		keys:   keys,
		window: window,
		limit:  limit,
		now:    time.Now,
	}, nil
}

// Allow returns true if key is still within the configured limit.
func (l *RateLimiter) Allow(ctx context.Context, key string) (bool, error) {
	if l == nil || l.client == nil {
		return false, domain.ErrUnavailable
	}

	key = sanitizeRateLimitKey(key)
	if key == "" {
		key = "unknown"
	}

	redisKey, err := l.redisKey(key)
	if err != nil {
		return false, err
	}

	count, err := l.client.Incr(ctx, redisKey).Result()
	if err != nil {
		return false, redisUnavailable("increment rate limit", err)
	}

	if count == 1 {
		if err := l.client.Expire(ctx, redisKey, l.window).Err(); err != nil {
			return false, redisUnavailable("expire rate limit", err)
		}
	}

	return count <= int64(l.limit), nil
}

func (l *RateLimiter) redisKey(key string) (string, error) {
	now := l.now
	if now == nil {
		now = time.Now
	}

	bucket := now().UTC().UnixNano() / l.window.Nanoseconds()
	redisKey, err := l.keys.RateLimit("auth", key, strconv.FormatInt(bucket, 10))
	if err != nil {
		return "", redisKeyError("build rate limit key", err)
	}

	return redisKey, nil
}

func sanitizeRateLimitKey(key string) string {
	key = strings.TrimSpace(strings.ToLower(key))
	key = strings.ReplaceAll(key, ":", "_")
	key = strings.ReplaceAll(key, " ", "_")
	key = strings.ReplaceAll(key, "\t", "_")
	key = strings.ReplaceAll(key, "\n", "_")
	key = strings.ReplaceAll(key, "\r", "_")
	return key
}
