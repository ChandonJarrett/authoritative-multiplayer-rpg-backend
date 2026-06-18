// Package main is the entry point for the API server.
package main

import (
	"context"
	"fmt"

	"connectrpc.com/connect"
	"github.com/ChandonJarrett/authoritative-multiplayer-rpg-backend/internal/api"
	"github.com/ChandonJarrett/authoritative-multiplayer-rpg-backend/internal/api/handlers"
	"github.com/ChandonJarrett/authoritative-multiplayer-rpg-backend/internal/api/middleware"
	"github.com/ChandonJarrett/authoritative-multiplayer-rpg-backend/internal/app"
	"github.com/ChandonJarrett/authoritative-multiplayer-rpg-backend/internal/cache"
	"github.com/ChandonJarrett/authoritative-multiplayer-rpg-backend/internal/db"
	"github.com/ChandonJarrett/authoritative-multiplayer-rpg-backend/internal/service"
	postgresstore "github.com/ChandonJarrett/authoritative-multiplayer-rpg-backend/internal/store/postgres"
	redisstore "github.com/ChandonJarrett/authoritative-multiplayer-rpg-backend/internal/store/redis"
)

func main() {
	rt, err := app.NewRuntime("api")
	if err != nil {
		app.Fatal("failed to initialize runtime", err)
	}
	defer rt.Close()

	readyCheck := func(ctx context.Context) error {
		if err := db.Health(ctx, rt.Postgres); err != nil {
			return fmt.Errorf("postgres: %w", err)
		}

		if err := cache.Health(ctx, rt.Redis); err != nil {
			return fmt.Errorf("redis: %w", err)
		}

		return nil
	}

	keys, err := cache.NewKeyBuilderFromConfig(rt.Config)
	if err != nil {
		app.Fatal("failed to create cache key builder", err)
	}

	userStore := postgresstore.NewPostgresUserStore(rt.Postgres)
	sessionStore := redisstore.NewRedisSessionStore(rt.Redis, keys)
	authService, err := service.NewAuthService(userStore, sessionStore)
	if err != nil {
		app.Fatal("failed to create auth service", err)
	}

	characterStore := postgresstore.NewPostgresCharacterStore(rt.Postgres)
	characterService, err := service.NewCharacterService(characterStore)
	if err != nil {
		app.Fatal("failed to create character service", err)
	}

	joinTokenStore := redisstore.NewRedisJoinTokenStore(rt.Redis, keys)
	gameServerStore := redisstore.NewRedisGameServerStore(rt.Redis, keys)
	gameService, err := service.NewGameService(characterStore, joinTokenStore, gameServerStore)
	if err != nil {
		app.Fatal("failed to create game service", err)
	}

	server, err := api.NewServer(api.Options{
		Addr:            rt.Config.APIHTTPAddr,
		Log:             rt.Log,
		ShutdownTimeout: rt.Config.ShutdownTimeout,
		AllowedOrigins:  rt.Config.APIAllowedOrigins,
		UnaryInterceptors: []connect.Interceptor{
			middleware.NewRPCLoggingInterceptor(rt.Log),
			middleware.NewAuthInterceptor(sessionStore, middleware.PublicProcedures()),
		},
		ReadyCheck: readyCheck,
		Handlers: api.Handlers{
			System:    handlers.NewSystemHandler("api"),
			Auth:      handlers.NewAuthHandler(authService),
			Character: handlers.NewCharacterHandler(characterService),
			Game:      handlers.NewGameHandler(gameService),
		},
	})
	if err != nil {
		app.Fatal("failed to create api server", err)
	}

	if err := server.Run(rt.Context); err != nil {
		app.Fatal("api server failed", err)
	}

	rt.Log.Info("api server stopped")
}
