// Package app provides shared application bootstrapping for services.
package app

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/ChandonJarrett/authoritative-multiplayer-rpg-backend/internal/cache"
	"github.com/ChandonJarrett/authoritative-multiplayer-rpg-backend/internal/config"
	"github.com/ChandonJarrett/authoritative-multiplayer-rpg-backend/internal/db"
	"github.com/ChandonJarrett/authoritative-multiplayer-rpg-backend/internal/logger"

	"github.com/jackc/pgx/v5/pgxpool"
	goredis "github.com/redis/go-redis/v9"
)

// ConfigLoader loads application configuration.
type ConfigLoader func() (config.Config, error)

// LoggerFactory creates a service logger from configuration.
type LoggerFactory func(cfg config.Config, serviceName string) *slog.Logger

// PostgresFactory creates a PostgreSQL connection pool.
type PostgresFactory func(ctx context.Context, cfg config.PostgresConfig) (*pgxpool.Pool, error)

// RedisFactory creates a Redis client.
type RedisFactory func(ctx context.Context, cfg config.RedisConfig) (*goredis.Client, error)

// ContextFactory creates the root runtime context and its stop function.
type ContextFactory func() (context.Context, context.CancelFunc)

// RuntimeDeps contains injectable runtime dependencies.
type RuntimeDeps struct {
	LoadConfig  ConfigLoader
	NewLogger   LoggerFactory
	NewPostgres PostgresFactory
	NewRedis    RedisFactory
	NewContext  ContextFactory
}

// Runtime contains shared service dependencies.
type Runtime struct {
	Config config.Config
	Log    *slog.Logger

	Context context.Context
	Stop    context.CancelFunc

	Postgres *pgxpool.Pool
	Redis    *goredis.Client
}

// NewRuntime loads config, initializes logging, connects to PostgreSQL and Redis,
// and returns a shared service runtime.
func NewRuntime(serviceName string) (*Runtime, error) {
	return NewRuntimeWithDeps(serviceName, RuntimeDeps{})
}

// NewRuntimeWithDeps creates a Runtime using injectable dependencies.
func NewRuntimeWithDeps(serviceName string, deps RuntimeDeps) (*Runtime, error) {
	deps = withRuntimeDefaults(deps)

	cfg, err := deps.LoadConfig()
	if err != nil {
		return nil, err
	}

	log := deps.NewLogger(cfg, serviceName)
	ctx, stop := deps.NewContext()

	pool, err := deps.NewPostgres(ctx, cfg.Postgres)
	if err != nil {
		stop()
		return nil, fmt.Errorf("connect postgres: %w", err)
	}

	redisClient, err := deps.NewRedis(ctx, cfg.Redis)
	if err != nil {
		db.Close(pool)
		stop()
		return nil, fmt.Errorf("connect redis: %w", err)
	}

	log.Info("runtime initialized")

	return &Runtime{
		Config:   cfg,
		Log:      log,
		Context:  ctx,
		Stop:     stop,
		Postgres: pool,
		Redis:    redisClient,
	}, nil
}

// Close releases runtime resources.
func (r *Runtime) Close() {
	if r == nil {
		return
	}

	if r.Redis != nil {
		if err := cache.Close(r.Redis); err != nil && r.Log != nil {
			r.Log.Error("failed to close redis", "error", err)
		}
	}

	db.Close(r.Postgres)

	if r.Stop != nil {
		r.Stop()
	}
}

// Fatal logs an error and exits.
func Fatal(msg string, err error) {
	slog.Error(msg, "error", err)
	exit(1)
}

var exit = os.Exit

func withRuntimeDefaults(deps RuntimeDeps) RuntimeDeps {
	if deps.LoadConfig == nil {
		deps.LoadConfig = config.Load
	}
	if deps.NewLogger == nil {
		deps.NewLogger = defaultLoggerFactory
	}
	if deps.NewPostgres == nil {
		deps.NewPostgres = db.NewPool
	}
	if deps.NewRedis == nil {
		deps.NewRedis = cache.NewClient
	}
	if deps.NewContext == nil {
		deps.NewContext = signalContext
	}

	return deps
}

func defaultLoggerFactory(cfg config.Config, serviceName string) *slog.Logger {
	return logger.New(logger.Options{
		Level:      cfg.LogLevel,
		Format:     cfg.LogFormat,
		AddSource:  true,
		SetDefault: true,
		Attrs:      []any{"service", serviceName},
	})
}

func signalContext() (context.Context, context.CancelFunc) {
	return signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
}
