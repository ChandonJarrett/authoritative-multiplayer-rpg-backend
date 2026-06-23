package app

import (
	"context"
	"fmt"
	"net/http"

	"connectrpc.com/connect"

	"github.com/ChandonJarrett/authoritative-multiplayer-rpg-backend/internal/api"
	"github.com/ChandonJarrett/authoritative-multiplayer-rpg-backend/internal/api/handlers"
	"github.com/ChandonJarrett/authoritative-multiplayer-rpg-backend/internal/cache"
	"github.com/ChandonJarrett/authoritative-multiplayer-rpg-backend/internal/db"
	"github.com/ChandonJarrett/authoritative-multiplayer-rpg-backend/internal/observability"
	"github.com/ChandonJarrett/authoritative-multiplayer-rpg-backend/internal/service"
	"github.com/ChandonJarrett/authoritative-multiplayer-rpg-backend/internal/store"
	postgresstore "github.com/ChandonJarrett/authoritative-multiplayer-rpg-backend/internal/store/postgres"
	redisstore "github.com/ChandonJarrett/authoritative-multiplayer-rpg-backend/internal/store/redis"
)

// apiStores holds all concrete store instances created during API server startup.
type apiStores struct {
	user        store.UserStore
	character   store.CharacterStore
	session     store.SessionStore
	joinToken   store.JoinTokenStore
	gameServer  store.GameServerStore
	authLimiter api.Limiter
}

// apiDeps groups all resolved API server dependencies.
type apiDeps struct {
	sessionStore store.SessionStore
	authLimiter  api.Limiter

	authService      *service.AuthService
	characterService *service.CharacterService
	gameService      *service.GameService

	metrics *observability.Metrics
}

// newAPIDeps creates all API server dependencies from shared runtime resources.
func newAPIDeps(rt *Runtime) (apiDeps, error) {
	keys, err := cache.NewKeyBuilderFromConfig(rt.Config)
	if err != nil {
		return apiDeps{}, fmt.Errorf("create cache key builder: %w", err)
	}

	stores, err := newAPIStores(rt, keys)
	if err != nil {
		return apiDeps{}, err
	}

	return newAPIServices(stores)
}

// newAPIStores creates all concrete store instances.
func newAPIStores(rt *Runtime, keys cache.KeyBuilder) (apiStores, error) {
	sessionStore, err := redisstore.NewSessionStore(rt.Redis, keys, cache.DefaultSessionTTL)
	if err != nil {
		return apiStores{}, fmt.Errorf("create session store: %w", err)
	}

	gameServerStore, err := redisstore.NewGameServerStore(rt.Redis, keys, cache.DefaultServerTTL)
	if err != nil {
		return apiStores{}, fmt.Errorf("create game server store: %w", err)
	}

	authLimiter, err := redisstore.NewRateLimiter(
		rt.Redis,
		keys,
		rt.Config.AuthRateLimitWindow,
		rt.Config.AuthRateLimitBurst,
	)
	if err != nil {
		return apiStores{}, fmt.Errorf("create auth rate limiter: %w", err)
	}

	return apiStores{
		user:        postgresstore.NewUserStore(rt.Postgres),
		character:   postgresstore.NewCharacterStore(rt.Postgres),
		session:     sessionStore,
		joinToken:   redisstore.NewJoinTokenStore(rt.Redis, keys),
		gameServer:  gameServerStore,
		authLimiter: authLimiter,
	}, nil
}

// newAPIServices creates all service instances from resolved stores.
func newAPIServices(stores apiStores) (apiDeps, error) {
	authService, err := service.NewAuthService(stores.user, stores.session)
	if err != nil {
		return apiDeps{}, fmt.Errorf("create auth service: %w", err)
	}

	characterService, err := service.NewCharacterService(stores.character)
	if err != nil {
		return apiDeps{}, fmt.Errorf("create character service: %w", err)
	}

	gameService, err := service.NewGameService(
		stores.character,
		stores.joinToken,
		stores.gameServer,
		cache.DefaultJoinTokenTTL,
	)
	if err != nil {
		return apiDeps{}, fmt.Errorf("create game service: %w", err)
	}

	return apiDeps{
		sessionStore:     stores.session,
		authLimiter:      stores.authLimiter,
		authService:      authService,
		characterService: characterService,
		gameService:      gameService,
		metrics:          observability.NewMetrics(apiServiceName),
	}, nil
}

// newAPIServerOptions builds the API server options struct from runtime and dependencies.
func newAPIServerOptions(rt *Runtime, deps apiDeps) api.Options {
	return api.Options{
		Addr:            rt.Config.APIHTTPAddr,
		Log:             rt.Log,
		ShutdownTimeout: rt.Config.ShutdownTimeout,
		AllowedOrigins:  rt.Config.APIAllowedOrigins,
		UnaryInterceptors: []connect.Interceptor{
			api.NewRPCLoggingInterceptor(rt.Log),
			observability.RPCInterceptor(deps.metrics),
			api.NewAuthRateLimitInterceptor(api.RateLimitConfig{
				Window:  rt.Config.AuthRateLimitWindow,
				Burst:   rt.Config.AuthRateLimitBurst,
				Limiter: deps.authLimiter,
			}),
			api.NewAuthInterceptor(deps.sessionStore, api.PublicProcedures()),
		},
		HTTPMiddleware: func(next http.Handler) http.Handler {
			return observability.HTTPMiddleware(deps.metrics, next)
		},
		MetricsHandler: deps.metrics.Handler(),
		ReadyCheck:     newAPIReadyCheck(rt),
		Handlers: api.Handlers{
			System:    handlers.NewSystemHandler(apiServiceName),
			Auth:      handlers.NewAuthHandler(deps.authService),
			Character: handlers.NewCharacterHandler(deps.characterService),
			Game:      handlers.NewGameHandler(deps.gameService),
		},
	}
}

// newAPIReadyCheck returns a readiness check that verifies PostgreSQL and Redis connectivity.
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
