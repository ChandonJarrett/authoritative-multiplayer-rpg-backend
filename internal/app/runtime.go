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
)

// Runtime holds shared dependencies available to both the API and game servers.
type Runtime struct {
	Config config.Config
	Log    *slog.Logger

	Context context.Context
	Stop    context.CancelFunc

	Postgres *pgxpool.Pool
	Redis    cache.Client
}

// NewRuntime initializes the shared runtime with production defaults.
func NewRuntime(serviceName string) (*Runtime, error) {
	return NewRuntimeWithDeps(serviceName, RuntimeDeps{})
}

// NewRuntimeWithDeps initializes the runtime using injectable dependencies.
// On any failure, previously acquired resources are released before returning.
func NewRuntimeWithDeps(serviceName string, deps RuntimeDeps) (*Runtime, error) {
	deps = withDefaults(deps)

	cfg, err := deps.LoadConfig()
	if err != nil {
		return nil, fmt.Errorf("load config: %w", err)
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

	log.Info("runtime initialized", "service", serviceName, "env", cfg.Env)

	return &Runtime{
		Config:   cfg,
		Log:      log,
		Context:  ctx,
		Stop:     stop,
		Postgres: pool,
		Redis:    redisClient,
	}, nil
}

// Close releases runtime resources in reverse initialization order (redis, postgres, context).
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

// Fatal logs a startup failure and terminates the process.
// It uses the global slog logger, which is set during runtime initialization.
func Fatal(msg string, err error) {
	slog.Error(msg, "error", err)
	exit(1)
}

// exit is the process exit function. Replaced in internal tests to prevent actual process termination.
var exit = os.Exit

func withDefaults(deps RuntimeDeps) RuntimeDeps {
	if deps.LoadConfig == nil {
		deps.LoadConfig = config.Load
	}
	if deps.NewLogger == nil {
		deps.NewLogger = defaultLogger
	}
	if deps.NewPostgres == nil {
		deps.NewPostgres = db.NewPool
	}
	if deps.NewRedis == nil {
		deps.NewRedis = newRedisClient
	}
	if deps.NewContext == nil {
		deps.NewContext = signalContext
	}
	return deps
}

func defaultLogger(cfg config.Config, serviceName string) *slog.Logger {
	return logger.New(logger.Options{
		Level:      cfg.LogLevel,
		Format:     cfg.LogFormat,
		AddSource:  true,
		SetDefault: true,
		Attrs:      []any{"service", serviceName},
	})
}

// newRedisClient is the production RedisFactory.
func newRedisClient(ctx context.Context, cfg config.RedisConfig) (cache.Client, error) {
	return cache.NewClient(ctx, cfg)
}

func signalContext() (context.Context, context.CancelFunc) {
	return signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
}
