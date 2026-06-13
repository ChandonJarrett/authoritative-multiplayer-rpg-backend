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

// ErrNilPool is returned when a nil pool is passed to a function that requires one.
var ErrNilPool = errors.New("postgres pool is nil")

// PoolStatsProvider abstracts the statistics returned by *pgxpool.Stat.
// Exposed as an interface so unit tests can supply a lightweight stub
// instead of requiring a live database connection.
type PoolStatsProvider interface {
	AcquireCount() int64
	AcquireDuration() time.Duration
	AcquiredConns() int32
	CanceledAcquireCount() int64
	ConstructingConns() int32
	EmptyAcquireCount() int64
	IdleConns() int32
	MaxConns() int32
	TotalConns() int32
}

// pgxPoolFactory creates a pool from a parsed config; replaceable in tests.
type pgxPoolFactory func(context.Context, *pgxpool.Config) (*pgxpool.Pool, error)

// NewPool creates a PostgreSQL connection pool and verifies connectivity.
func NewPool(ctx context.Context, cfg config.PostgresConfig) (*pgxpool.Pool, error) {
	return newPool(ctx, cfg, pgxpool.NewWithConfig)
}

func newPool(ctx context.Context, cfg config.PostgresConfig, factory pgxPoolFactory) (*pgxpool.Pool, error) {
	poolCfg, err := pgxpool.ParseConfig(cfg.URL)
	if err != nil {
		return nil, fmt.Errorf("parse postgres config: %w", err)
	}

	applyPoolConfig(poolCfg, cfg)

	pool, err := factory(ctx, poolCfg)
	if err != nil {
		return nil, fmt.Errorf("create postgres pool: %w", err)
	}

	if err := Health(ctx, pool); err != nil {
		pool.Close()
		return nil, err
	}

	return pool, nil
}

func applyPoolConfig(dst *pgxpool.Config, src config.PostgresConfig) {
	dst.MaxConns = src.MaxConns
	dst.MinConns = src.MinConns
	dst.MaxConnLifetime = src.MaxConnLifetime
	dst.MaxConnIdleTime = src.MaxConnIdleTime
	dst.HealthCheckPeriod = src.HealthCheckPeriod
}

// Health checks PostgreSQL connectivity with a ping.
func Health(ctx context.Context, pool *pgxpool.Pool) error {
	if pool == nil {
		return ErrNilPool
	}
	if err := pool.Ping(ctx); err != nil {
		return fmt.Errorf("postgres ping: %w", err)
	}

	return nil
}

// Close closes the PostgreSQL pool.
func Close(pool *pgxpool.Pool) {
	if pool != nil {
		pool.Close()
	}
}

// Stats is a snapshot of PostgreSQL connection pool statistics.
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

// Snapshot returns a point-in-time snapshot of PostgreSQL connection pool statistics.
func Snapshot(pool *pgxpool.Pool) (Stats, error) {
	if pool == nil {
		return Stats{}, ErrNilPool
	}
	return SnapshotFromStats(pool.Stat()), nil
}

// SnapshotFromStats builds a Stats value from a PoolStatsProvider.
// Exposed so unit tests can verify stat mapping without a live pool.
func SnapshotFromStats(s PoolStatsProvider) Stats {
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
	}
}
