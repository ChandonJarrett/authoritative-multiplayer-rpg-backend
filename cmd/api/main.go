// Package main is the entry point for the API server.
package main

import (
	"context"
	"fmt"

	"connectrpc.com/connect"
	"github.com/ChandonJarrett/authoritative-multiplayer-rpg-backend/internal/api"
	"github.com/ChandonJarrett/authoritative-multiplayer-rpg-backend/internal/app"
	"github.com/ChandonJarrett/authoritative-multiplayer-rpg-backend/internal/cache"
	"github.com/ChandonJarrett/authoritative-multiplayer-rpg-backend/internal/db"
	"github.com/ChandonJarrett/authoritative-multiplayer-rpg-backend/internal/service"
	"github.com/ChandonJarrett/authoritative-multiplayer-rpg-backend/internal/store"
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

	userStore := store.NewPostgresUserStore(rt.Postgres)
	sessionStore := store.NewRedisSessionStore(rt.Redis, keys)
	authService := service.NewAuthService(userStore, sessionStore)

	characterStore := store.NewPostgresCharacterStore(rt.Postgres)
	characterService := service.NewCharacterService(characterStore)

	joinTokenStore := store.NewRedisJoinTokenStore(rt.Redis, keys)
	gameServerStore := store.NewRedisGameServerStore(rt.Redis, keys)
	handoffService := service.NewGameHandoffService(characterStore, joinTokenStore, gameServerStore)

	server, err := api.NewServer(api.Options{
		Addr:            rt.Config.APIHTTPAddr,
		Log:             rt.Log,
		ShutdownTimeout: rt.Config.ShutdownTimeout,
		ReadyCheck:      readyCheck,
		SystemHandler:   api.NewSystemHandler("api"),
		AllowedOrigins: []string{
			"http://localhost:3000",
			"http://localhost:5173",
			"http://127.0.0.1:3000",
			"http://127.0.0.1:5173",
		},
		AuthHandler: api.NewAuthHandler(authService),
		UnaryInterceptors: []connect.Interceptor{
			api.NewAuthInterceptor(sessionStore),
		},
		CharacterHandler: api.NewCharacterHandler(characterService),
		GameHandler:      api.NewGameHandler(handoffService),
	})
	if err != nil {
		app.Fatal("failed to create api server", err)
	}

	if err := server.Run(rt.Context); err != nil {
		app.Fatal("api server failed", err)
	}

	rt.Log.Info("api server stopped")
}
