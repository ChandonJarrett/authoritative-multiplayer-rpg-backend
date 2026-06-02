package cache

import (
	"context"
	"errors"
	"fmt"

	"github.com/ChandonJarrett/authoritative-multiplayer-rpg-backend/internal/config"

	goredis "github.com/redis/go-redis/v9"
)

var ErrNilClient = errors.New("redis client is nil")

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

func Health(ctx context.Context, client *goredis.Client) error {
	if client == nil {
		return ErrNilClient
	}

	if err := client.Ping(ctx).Err(); err != nil {
		return fmt.Errorf("redis health check failed: %w", err)
	}

	return nil
}

func Close(client *goredis.Client) error {
	if client == nil {
		return nil
	}

	if err := client.Close(); err != nil {
		return fmt.Errorf("close redis client: %w", err)
	}

	return nil
}

type Stats struct {
	PoolHits     uint32
	PoolMisses   uint32
	PoolTimeouts uint32
	TotalConns   uint32
	IdleConns    uint32
	StaleConns   uint32
}

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
