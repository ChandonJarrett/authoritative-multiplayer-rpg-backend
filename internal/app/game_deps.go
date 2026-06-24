package app

import (
	"context"
	"fmt"

	"github.com/google/uuid"

	"github.com/ChandonJarrett/authoritative-multiplayer-rpg-backend/internal/cache"
	"github.com/ChandonJarrett/authoritative-multiplayer-rpg-backend/internal/db"
	"github.com/ChandonJarrett/authoritative-multiplayer-rpg-backend/internal/game"
	"github.com/ChandonJarrett/authoritative-multiplayer-rpg-backend/internal/store"
	postgresstore "github.com/ChandonJarrett/authoritative-multiplayer-rpg-backend/internal/store/postgres"
	redisstore "github.com/ChandonJarrett/authoritative-multiplayer-rpg-backend/internal/store/redis"
)

// gameStores holds all concrete store instances created during game server startup.
type gameStores struct {
	characters     store.CharacterStore
	joinTokens     store.JoinTokenStore
	characterLocks *redisstore.CharacterLockStore
	gameServers    store.GameServerStore
}

// newGameConfig resolves all game server dependencies and builds the configuration.
func newGameConfig(rt *Runtime) (game.GameServerConfig, error) {
	keys, err := cache.NewKeyBuilderFromConfig(rt.Config)
	if err != nil {
		return game.GameServerConfig{}, fmt.Errorf("create cache key builder: %w", err)
	}

	stores, err := newGameStores(rt, keys)
	if err != nil {
		return game.GameServerConfig{}, err
	}

	return newGameServerConfig(rt, stores), nil
}

// newGameStores creates all concrete store instances.
func newGameStores(rt *Runtime, keys cache.KeyBuilder) (gameStores, error) {
	lockStore, err := redisstore.NewCharacterLockStore(rt.Redis, keys, cache.DefaultCharacterLockTTL)
	if err != nil {
		return gameStores{}, fmt.Errorf("create character lock store: %w", err)
	}

	serverStore, err := redisstore.NewGameServerStore(rt.Redis, keys, cache.DefaultServerTTL)
	if err != nil {
		return gameStores{}, fmt.Errorf("create game server store: %w", err)
	}

	return gameStores{
		characters:     postgresstore.NewCharacterStore(rt.Postgres),
		joinTokens:     redisstore.NewJoinTokenStore(rt.Redis, keys),
		characterLocks: lockStore,
		gameServers:    serverStore,
	}, nil
}

// newGameServerConfig builds the game.GameServerConfig from runtime and resolved stores.
func newGameServerConfig(rt *Runtime, stores gameStores) game.GameServerConfig {
	return game.GameServerConfig{
		ENetAddr:        rt.Config.GameENetAddr,
		HTTPAddr:        rt.Config.GameHTTPAddr,
		ServerID:        uuid.NewString(),
		ShutdownTimeout: rt.Config.ShutdownTimeout,
		JoinTokens:      stores.joinTokens,
		CharacterLocks:  stores.characterLocks,
		Characters:      stores.characters,
		GameServers:     stores.gameServers,
		ServerTTL:       cache.DefaultServerTTL,
		LockTTL:         cache.DefaultCharacterLockTTL,
		TickRate:        64,
		MoveSpeed:       0.05,
		ReadyCheck:      newGameReadyCheck(rt),
	}
}

// newGameReadyCheck returns a readiness check that verifies PostgreSQL and Redis connectivity.
func newGameReadyCheck(rt *Runtime) func(ctx context.Context) error {
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
