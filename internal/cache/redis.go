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

// NewClient creates and returns a new Redis client based on the provided configuration.
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

	if err := client.Ping(ctx).Err(); err != nil {
		_ = client.Close()
		return nil, fmt.Errorf("ping redis: %w", err)
	}

	return client, nil
}

// Health checks the connectivity of the Redis client by sending a PING command.
func Health(ctx context.Context, client *goredis.Client) error {
	if client == nil {
		return ErrNilClient
	}

	if err := client.Ping(ctx).Err(); err != nil {
		return fmt.Errorf("redis health check failed: %w", err)
	}

	return nil
}

// Close gracefully closes the Redis client connection.
func Close(client *goredis.Client) error {
	if client == nil {
		return nil
	}

	if err := client.Close(); err != nil {
		return fmt.Errorf("close redis client: %w", err)
	}

	return nil
}

// Stats represents the connection pool statistics of the Redis client.
type Stats struct {
	PoolHits     uint32
	PoolMisses   uint32
	PoolTimeouts uint32
	TotalConns   uint32
	IdleConns    uint32
	StaleConns   uint32
}

// Snapshot retrieves the current connection pool statistics from the Redis client.
func Snapshot(client *goredis.Client) (Stats, error) {
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
