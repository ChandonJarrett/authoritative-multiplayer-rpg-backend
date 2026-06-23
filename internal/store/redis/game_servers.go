package redis

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"time"

	"github.com/ChandonJarrett/authoritative-multiplayer-rpg-backend/internal/cache"
	"github.com/ChandonJarrett/authoritative-multiplayer-rpg-backend/internal/domain"
	"github.com/ChandonJarrett/authoritative-multiplayer-rpg-backend/internal/validate"
	goredis "github.com/redis/go-redis/v9"
)

type gameServerPayload struct {
	ID        string    `json:"id"`
	Addr      string    `json:"addr"`
	ExpiresAt time.Time `json:"expires_at"`
}

// GameServerStore stores ephemeral game server registry records in Redis.
type GameServerStore struct {
	client    cache.Client
	keys      cache.KeyBuilder
	serverTTL time.Duration
}

// NewGameServerStore creates a Redis game-server store.
func NewGameServerStore(client cache.Client, keys cache.KeyBuilder, serverTTL time.Duration) (*GameServerStore, error) {
	if client == nil {
		return nil, fmt.Errorf("redis client is required: %w", domain.ErrInvalidArgument)
	}
	if serverTTL <= 0 {
		return nil, fmt.Errorf("server TTL must be > 0: %w", domain.ErrInvalidArgument)
	}
	return &GameServerStore{
		client:    client,
		keys:      keys,
		serverTTL: serverTTL,
	}, nil
}

// RegisterGameServer registers or refreshes a game server heartbeat.
// Callers must ensure server.ID and server.Addr are already validated and non-empty.
func (s *GameServerStore) RegisterGameServer(ctx context.Context, server domain.GameServer) error {
	if s == nil || s.client == nil {
		return domain.ErrUnavailable
	}

	serverKey, err := s.keys.Server(server.ID)
	if err != nil {
		return redisKeyError("build server key", err)
	}

	indexKey, err := s.keys.ServersIndex()
	if err != nil {
		return redisKeyError("build servers index key", err)
	}

	ttl := time.Until(server.ExpiresAt)
	if server.ExpiresAt.IsZero() {
		return fmt.Errorf("game server expiry is required: %w", domain.ErrInvalidArgument)
	}
	if ttl <= 0 {
		return fmt.Errorf("game server already expired: %w", domain.ErrInvalidArgument)
	}

	payload := gameServerPayload{
		ID:        server.ID,
		Addr:      server.Addr,
		ExpiresAt: server.ExpiresAt.UTC(),
	}

	data, err := json.Marshal(payload)
	if err != nil {
		return redisInternal("marshal game server", err)
	}

	if err := s.client.Set(ctx, serverKey, data, ttl).Err(); err != nil {
		return redisUnavailable("register game server", err)
	}

	if err := s.client.SAdd(ctx, indexKey, server.ID).Err(); err != nil {
		return redisUnavailable("index game server", err)
	}

	// Index TTL tracks the configured server TTL so it stays alive as long as any server could.
	if err := s.client.Expire(ctx, indexKey, s.serverTTL).Err(); err != nil {
		return redisUnavailable("expire game server index", err)
	}

	return nil
}

// DeregisterGameServer removes the server heartbeat key.
// The servers index has a TTL and will naturally expire if no servers refresh it.
func (s *GameServerStore) DeregisterGameServer(ctx context.Context, serverID string) error {
	if s == nil || s.client == nil {
		return domain.ErrUnavailable
	}

	serverID, err := validate.RequiredID("server ID", serverID)
	if err != nil {
		return err
	}

	serverKey, err := s.keys.Server(serverID)
	if err != nil {
		return redisKeyError("build server key", err)
	}

	if err := s.client.Del(ctx, serverKey).Err(); err != nil {
		return redisUnavailable("deregister game server", err)
	}

	return nil
}

// ListGameServers returns currently visible game servers.
func (s *GameServerStore) ListGameServers(ctx context.Context) ([]domain.GameServer, error) {
	if s == nil || s.client == nil {
		return nil, domain.ErrUnavailable
	}

	indexKey, err := s.keys.ServersIndex()
	if err != nil {
		return nil, redisKeyError("build servers index key", err)
	}

	serverIDs, err := s.client.SMembers(ctx, indexKey).Result()
	if err != nil {
		return nil, redisUnavailable("list game server IDs", err)
	}

	servers := make([]domain.GameServer, 0, len(serverIDs))
	for _, serverID := range serverIDs {
		serverID, err := validate.RequiredID("server ID", serverID)
		if err != nil {
			continue
		}

		server, err := s.getGameServer(ctx, serverID)
		if err != nil {
			if errors.Is(err, domain.ErrNotFound) {
				continue
			}
			return nil, err
		}

		servers = append(servers, server)
	}

	sort.Slice(servers, func(i, j int) bool {
		return servers[i].ID < servers[j].ID
	})

	return servers, nil
}

// GetGameServerByID returns the game server details by ID.
func (s *GameServerStore) GetGameServerByID(ctx context.Context, serverID string) (domain.GameServer, error) {
	if s == nil || s.client == nil {
		return domain.GameServer{}, domain.ErrUnavailable
	}

	serverID, err := validate.RequiredID("server ID", serverID)
	if err != nil {
		return domain.GameServer{}, err
	}

	server, err := s.getGameServer(ctx, serverID)
	if err != nil {
		return domain.GameServer{}, err
	}

	return server, nil
}

func (s *GameServerStore) getGameServer(ctx context.Context, serverID string) (domain.GameServer, error) {
	serverKey, err := s.keys.Server(serverID)
	if err != nil {
		return domain.GameServer{}, redisKeyError("build server key", err)
	}

	raw, err := s.client.Get(ctx, serverKey).Result()
	if errors.Is(err, goredis.Nil) {
		return domain.GameServer{}, domain.ErrNotFound
	}
	if err != nil {
		return domain.GameServer{}, redisUnavailable("get game server", err)
	}

	var payload gameServerPayload
	if err := json.Unmarshal([]byte(raw), &payload); err != nil {
		return domain.GameServer{}, redisUnavailable("decode game server", err)
	}

	if _, err := validate.RequiredID("server ID", payload.ID); err != nil {
		return domain.GameServer{}, domain.ErrNotFound
	}
	if _, err := validate.RequiredID("server address", payload.Addr); err != nil {
		return domain.GameServer{}, domain.ErrNotFound
	}

	return domain.GameServer{
		ID:        payload.ID,
		Addr:      payload.Addr,
		ExpiresAt: payload.ExpiresAt,
	}, nil
}
