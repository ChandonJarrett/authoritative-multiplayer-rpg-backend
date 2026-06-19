package app

import (
	"context"
	"fmt"
	"time"

	"connectrpc.com/connect"

	"github.com/ChandonJarrett/authoritative-multiplayer-rpg-backend/internal/api"
	"github.com/ChandonJarrett/authoritative-multiplayer-rpg-backend/internal/api/handlers"
	"github.com/ChandonJarrett/authoritative-multiplayer-rpg-backend/internal/api/middleware"
	"github.com/ChandonJarrett/authoritative-multiplayer-rpg-backend/internal/cache"
	"github.com/ChandonJarrett/authoritative-multiplayer-rpg-backend/internal/db"
	"github.com/ChandonJarrett/authoritative-multiplayer-rpg-backend/internal/service"
	postgresstore "github.com/ChandonJarrett/authoritative-multiplayer-rpg-backend/internal/store/postgres"
	redisstore "github.com/ChandonJarrett/authoritative-multiplayer-rpg-backend/internal/store/redis"
)

const apiServiceName = "api"

// RunAPI initializes and runs the API server.
func RunAPI() error {
	rt, err := NewRuntime(apiServiceName)
	if err != nil {
		return fmt.Errorf("initialize runtime: %w", err)
	}
	defer rt.Close()

	server, err := NewAPIServer(rt)
	if err != nil {
		return fmt.Errorf("create api server: %w", err)
	}

	if err := server.Run(rt.Context); err != nil {
		return err
	}

	rt.Log.Info("api server stopped")
	return nil
}

// NewAPIServer wires API dependencies and returns a runnable API server.
func NewAPIServer(rt *Runtime) (*api.Server, error) {
	if rt == nil {
		return nil, fmt.Errorf("runtime is nil")
	}

	keys, err := cache.NewKeyBuilderFromConfig(rt.Config)
	if err != nil {
		return nil, fmt.Errorf("create cache key builder: %w", err)
	}

	userStore := postgresstore.NewPostgresUserStore(rt.Postgres)
	characterStore := postgresstore.NewPostgresCharacterStore(rt.Postgres)

	sessionStore := redisstore.NewRedisSessionStore(rt.Redis, keys)
	joinTokenStore := redisstore.NewRedisJoinTokenStore(rt.Redis, keys)
	gameServerStore := redisstore.NewRedisGameServerStore(rt.Redis, keys)

	authService, err := service.NewAuthService(userStore, sessionStore)
	if err != nil {
		return nil, fmt.Errorf("create auth service: %w", err)
	}

	characterService, err := service.NewCharacterService(characterStore)
	if err != nil {
		return nil, fmt.Errorf("create character service: %w", err)
	}

	gameService, err := service.NewGameService(characterStore, joinTokenStore, gameServerStore)
	if err != nil {
		return nil, fmt.Errorf("create game service: %w", err)
	}

	return api.NewServer(api.Options{
		Addr:            rt.Config.APIHTTPAddr,
		Log:             rt.Log,
		ShutdownTimeout: rt.Config.ShutdownTimeout,
		AllowedOrigins:  rt.Config.APIAllowedOrigins,
		UnaryInterceptors: []connect.Interceptor{
			middleware.NewRPCLoggingInterceptor(rt.Log),
			middleware.NewAuthRateLimitInterceptor(middleware.RateLimitConfig{
				Window: time.Minute,
				Burst:  10,
			}),
			middleware.NewAuthInterceptor(sessionStore, middleware.PublicProcedures()),
		},
		ReadyCheck: newAPIReadyCheck(rt),
		Handlers: api.Handlers{
			System:    handlers.NewSystemHandler(apiServiceName),
			Auth:      handlers.NewAuthHandler(authService),
			Character: handlers.NewCharacterHandler(characterService),
			Game:      handlers.NewGameHandler(gameService),
		},
	})
}

func newAPIReadyCheck(rt *Runtime) api.ReadyCheck {
	return func(ctx context.Context) error {
		if err := db.Health(ctx, rt.Postgres); err != nil {
			return fmt.Errorf("postgres: %w", err)
		}
		if err := cache.Health(ctx, rt.Redis); err != nil {
			return fmt.Errorf("redis: %w", err)
		}
		return nil
	}
}
