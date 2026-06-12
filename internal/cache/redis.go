// Package cache provides Redis connectivity, key construction, and cache-related utilities.
package cache

import (
	"context"
	"errors"
	"fmt"

	"github.com/ChandonJarrett/authoritative-multiplayer-rpg-backend/internal/config"

	goredis "github.com/redis/go-redis/v9"
)

// ErrNilClient indicates the Redis client is nil.
var ErrNilClient = errors.New("redis client is nil")

// Client is the minimal Redis client behavior used by this package.
type Client interface {
	Ping(ctx context.Context) *goredis.StatusCmd
	Close() error
	PoolStats() *goredis.PoolStats
}

// NewClient creates a Redis client from configuration and verifies connectivity.
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

// Health verifies Redis connectivity with a PING command.
func Health(ctx context.Context, client Client) error {
	if client == nil {
		return ErrNilClient
	}

	if err := client.Ping(ctx).Err(); err != nil {
		return fmt.Errorf("redis health check failed: %w", err)
	}

	return nil
}

// Close closes the Redis client.
func Close(client Client) error {
	if client == nil {
		return nil
	}

	if err := client.Close(); err != nil {
		return fmt.Errorf("close redis client: %w", err)
	}

	return nil
}

// Stats represents Redis connection pool statistics.
type Stats struct {
	PoolHits     uint32
	PoolMisses   uint32
	PoolTimeouts uint32
	TotalConns   uint32
	IdleConns    uint32
	StaleConns   uint32
}

// Snapshot returns Redis connection pool statistics.
func Snapshot(client Client) (Stats, error) {
	if client == nil {
		return Stats{}, ErrNilClient
	}

	s := client.PoolStats()

	return Stats{
		PoolHits:     s.Hits,
		PoolMisses:   s.Misses,
		PoolTimeouts: s.Timeouts,
		TotalConns:   s.TotalConns,
		IdleConns:    s.IdleConns,
		StaleConns:   s.StaleConns,
	}, nil
}
