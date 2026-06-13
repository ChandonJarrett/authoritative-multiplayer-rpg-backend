// Package cache provides Redis connectivity, key construction, and cache-related utilities.
package cache

import (
	"context"
	"errors"
	"fmt"

	"github.com/ChandonJarrett/authoritative-multiplayer-rpg-backend/internal/config"

	goredis "github.com/redis/go-redis/v9"
)

// ErrNilClient is returned when a nil Redis client is passed to a function that requires one.
var ErrNilClient = errors.New("redis client is nil")

// ErrNilPoolStats is returned when PoolStats returns nil.
var ErrNilPoolStats = errors.New("redis pool stats is nil")

// Client is the minimal Redis interface required by this package; *goredis.Client satisfies this interface.
type Client interface {
	Ping(ctx context.Context) *goredis.StatusCmd
	Close() error
	PoolStats() *goredis.PoolStats
}

// NewClient creates and validates a Redis client from configuration.
func NewClient(ctx context.Context, cfg config.RedisConfig) (*goredis.Client, error) {
	client := goredis.NewClient(&goredis.Options{
		Addr:         cfg.Addr,
		Password:     cfg.Password,
		DB:           cfg.DB,
		DialTimeout:  cfg.DialTimeout,
		ReadTimeout:  cfg.ReadTimeout,
		WriteTimeout: cfg.WriteTimeout,
		PoolSize:     cfg.PoolSize,
		MinIdleConns: cfg.MinIdleConns,
	})

	if err := Health(ctx, client); err != nil {
		_ = client.Close()
		return nil, err
	}

	return client, nil
}

// Health verifies Redis connectivity with a PING.
func Health(ctx context.Context, client Client) error {
	if client == nil {
		return ErrNilClient
	}
	if err := client.Ping(ctx).Err(); err != nil {
		return fmt.Errorf("redis ping: %w", err)
	}
	return nil
}

// Close closes the Redis client.
func Close(client Client) error {
	if client == nil {
		return nil
	}
	if err := client.Close(); err != nil {
		return fmt.Errorf("close redis: %w", err)
	}
	return nil
}

// Stats is a snapshot of Redis connection pool statistics.
type Stats struct {
	PoolHits     uint32
	PoolMisses   uint32
	PoolTimeouts uint32
	TotalConns   uint32
	IdleConns    uint32
	StaleConns   uint32
}

// Snapshot returns a point-in-time snapshot of Redis connection pool statistics.
func Snapshot(client Client) (Stats, error) {
	if client == nil {
		return Stats{}, ErrNilClient
	}
	s := client.PoolStats()
	if s == nil {
		return Stats{}, ErrNilPoolStats
	}
	return Stats{
		PoolHits:     s.Hits,
		PoolMisses:   s.Misses,
		PoolTimeouts: s.Timeouts,
		TotalConns:   s.TotalConns,
		IdleConns:    s.IdleConns,
		StaleConns:   s.StaleConns,
	}, nil
}
