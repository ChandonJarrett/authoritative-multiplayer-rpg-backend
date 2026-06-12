// Package db provides PostgreSQL connectivity and transaction utilities.
package db

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/ChandonJarrett/authoritative-multiplayer-rpg-backend/internal/config"

	"github.com/jackc/pgx/v5/pgxpool"
)

// ErrNilPool indicates the PostgreSQL pool is nil.
var ErrNilPool = errors.New("postgres pool is nil")

// NewPool creates a PostgreSQL connection pool and verifies connectivity.
func NewPool(ctx context.Context, cfg config.PostgresConfig) (*pgxpool.Pool, error) {
	poolCfg, err := pgxpool.ParseConfig(cfg.URL)
	if err != nil {
		return nil, fmt.Errorf("parse postgres config: %w", err)
	}

	poolCfg.MaxConns = cfg.MaxConns
	poolCfg.MinConns = cfg.MinConns
	poolCfg.MaxConnLifetime = cfg.MaxConnLifetime
	poolCfg.MaxConnIdleTime = cfg.MaxConnIdleTime
	poolCfg.HealthCheckPeriod = cfg.HealthCheckPeriod

	pool, err := pgxpool.NewWithConfig(ctx, poolCfg)
	if err != nil {
		return nil, fmt.Errorf("create postgres pool: %w", err)
	}

	if err := Health(ctx, pool); err != nil {
		pool.Close()
		return nil, err
	}

	return pool, nil
}

// Health checks PostgreSQL connectivity by pinging the database.
func Health(ctx context.Context, pool *pgxpool.Pool) error {
	if pool == nil {
		return ErrNilPool
	}

	if err := pool.Ping(ctx); err != nil {
		return fmt.Errorf("postgres health check failed: %w", err)
	}

	return nil
}

// Close closes the PostgreSQL pool.
func Close(pool *pgxpool.Pool) {
	if pool != nil {
		pool.Close()
	}
}

// Stats represents PostgreSQL connection pool statistics.
type Stats struct {
	AcquireCount         int64
	AcquireDuration      time.Duration
	AcquiredConns        int32
	CanceledAcquireCount int64
	ConstructingConns    int32
	EmptyAcquireCount    int64
	IdleConns            int32
	MaxConns             int32
	TotalConns           int32
}

// Snapshot captures PostgreSQL connection pool statistics.
func Snapshot(pool *pgxpool.Pool) (Stats, error) {
	if pool == nil {
		return Stats{}, ErrNilPool
	}

	s := pool.Stat()

	return Stats{
		AcquireCount:         s.AcquireCount(),
		AcquireDuration:      s.AcquireDuration(),
		AcquiredConns:        s.AcquiredConns(),
		CanceledAcquireCount: s.CanceledAcquireCount(),
		ConstructingConns:    s.ConstructingConns(),
		EmptyAcquireCount:    s.EmptyAcquireCount(),
		IdleConns:            s.IdleConns(),
		MaxConns:             s.MaxConns(),
		TotalConns:           s.TotalConns(),
	}, nil
}
