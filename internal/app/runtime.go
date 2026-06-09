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
	cfg, err := config.Load()
	if err != nil {
		return nil, err
	}

	log := logger.New(logger.Options{
		Level:      cfg.LogLevel,
		Format:     cfg.LogFormat,
		AddSource:  true,
		SetDefault: true,
		Attrs:      []any{"service", serviceName},
	})

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)

	pool, err := db.NewPool(ctx, cfg.Postgres)
	if err != nil {
		stop()
		return nil, fmt.Errorf("connect postgres: %w", err)
	}

	redisClient, err := cache.NewClient(ctx, cfg.Redis)
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
		if err := cache.Close(r.Redis); err != nil {
			if r.Log != nil {
				r.Log.Error("failed to close redis", "error", err)
			}
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
	os.Exit(1)
}
