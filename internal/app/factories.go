// Package app provides shared application bootstrapping for services.
package app

import (
	"context"
	"log/slog"

	"github.com/ChandonJarrett/authoritative-multiplayer-rpg-backend/internal/cache"
	"github.com/ChandonJarrett/authoritative-multiplayer-rpg-backend/internal/config"

	"github.com/jackc/pgx/v5/pgxpool"
)

// ConfigLoader loads application configuration.
type ConfigLoader func() (config.Config, error)

// LoggerFactory creates a structured logger for a named service.
type LoggerFactory func(cfg config.Config, serviceName string) *slog.Logger

// PostgresFactory creates a PostgreSQL connection pool.
type PostgresFactory func(ctx context.Context, cfg config.PostgresConfig) (*pgxpool.Pool, error)

// RedisFactory creates a Redis client.
type RedisFactory func(ctx context.Context, cfg config.RedisConfig) (cache.Client, error)

// ContextFactory creates the root context and its cancellation function.
type ContextFactory func() (context.Context, context.CancelFunc)

// RuntimeDeps contains injectable dependencies for NewRuntimeWithDeps.
// All fields are optional; nil entries are replaced with production defaults.
type RuntimeDeps struct {
	LoadConfig  ConfigLoader
	NewLogger   LoggerFactory
	NewPostgres PostgresFactory
	NewRedis    RedisFactory
	NewContext  ContextFactory
}
